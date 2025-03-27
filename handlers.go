package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IlMeloIl/RSS/internal/database"
	"github.com/google/uuid"
)

func HandlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("login command takes only one argument: login <username>")
	}

	username := cmd.args[0]
	ctx := context.Background()

	_, err := s.db.GetUser(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User with name %s not in database", username)
			os.Exit(1)
		}
		return fmt.Errorf("error fetching user from database")
	}

	s.config.SetUser(username)
	fmt.Println("User has been set to", username)
	return nil
}

func HandlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("register command takes only one argument: register <username>")
	}

	username := cmd.args[0]
	ctx := context.Background()

	_, err := s.db.GetUser(context.Background(), username)
	if err == nil {
		os.Exit(1)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("database error: %w", err)
	}

	_, err = s.db.CreateUser(ctx, database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: username})
	if err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	fmt.Printf("User %s successfully created!\n", username)
	s.config.SetUser(username)
	return nil

}

func HandlerReset(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("reset command doesn't need argument: reset")
	}

	ctx := context.Background()

	if err := s.db.ResetUsers(ctx); err != nil {
		return fmt.Errorf("error reseting users table: %w", err)
	}

	log.Println("users table reseted")
	return nil
}

func HandlerUsers(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("users command doesn't need argument: users")
	}

	ctx := context.Background()
	users, err := s.db.GetUsers(ctx)
	if err != nil {
		return fmt.Errorf("error geting all users from db: %w", err)
	}

	currentUser := s.config.GetUser()
	for _, user := range users {
		if user == currentUser.CurrentUserName {
			fmt.Printf("* %s (current)\n", user)
		} else {
			fmt.Printf("* %s\n", user)
		}
	}
	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error making new request with context: %w", err)
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error returning resp: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro reading all from response body: %w", err)
	}

	v := RSSFeed{}
	if err := xml.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("error unmarshaling xml: %w", err)
	}

	v.Channel.Title = html.UnescapeString(v.Channel.Title)
	v.Channel.Description = html.UnescapeString(v.Channel.Description)

	for i := range v.Channel.Item {
		v.Channel.Item[i].Title = html.UnescapeString(v.Channel.Item[i].Title)
		v.Channel.Item[i].Description = html.UnescapeString(v.Channel.Item[i].Description)
	}

	return &v, nil

}

func HandlerAgg(s *state, cmd command) error {
	ctx := context.Background()

	rssFeed, err := fetchFeed(ctx, "https://www.wagslane.dev/index.xml")
	if err != nil {
		return fmt.Errorf("error from fetchFeed: %w", err)
	}

	jsonData, err := json.MarshalIndent(rssFeed, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil

}

func HandlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("addfeed command needs two arguments: addfeed <name> <url>")
	}

	name := cmd.args[0]
	url := cmd.args[1]
	userID := user.ID

	ctx := context.Background()
	_, err := s.db.CreateFeed(ctx, database.CreateFeedParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: name, Url: url, UserID: userID})
	if err != nil {
		return fmt.Errorf("error creating feed: %w", err)
	}

	followCmd := command{
		name: "follow",
		args: []string{url},
	}

	if err = HandlerFollow(s, followCmd, user); err != nil {
		return fmt.Errorf("error handler follow in handler add feed: %w", err)
	}

	return nil
}

func HandlerFeeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("feeds command doesn't take any arguments: feeds")
	}

	ctx := context.Background()
	feedsFromDB, err := s.db.GetFeeds(ctx)
	if err != nil {
		return fmt.Errorf("error getting feeds from db: %w", err)
	}

	for _, feed := range feedsFromDB {
		username, err := s.db.GetUserFromID(ctx, feed.UserID)
		if err != nil {
			return fmt.Errorf("error getting user by id: %w", err)
		}
		fmt.Printf("%s\n", feed.Name)
		fmt.Printf("URL: %s\n", feed.Url)
		fmt.Printf("Added by: %s\n", username)
		fmt.Println(strings.Repeat("-", 50))
	}
	return nil
}

func HandlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("follow command needs one argument: follow <url>")
	}

	url := cmd.args[0]

	ctx := context.Background()

	feed, err := s.db.GetFeedByURL(ctx, url)
	if err != nil {
		return fmt.Errorf("error getting feed from db by url: %w", err)
	}

	_, err = s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), UserID: user.ID, FeedID: feed.ID})
	if err != nil {
		return fmt.Errorf("error creating feed follow: %w", err)
	}

	fmt.Printf("%s is now following %s!\n", user.Name, feed.Name)

	return nil
}

func HandlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("following command doesn't need any argumenmts: following")
	}

	ctx := context.Background()

	feedFollows, err := s.db.GetFeedFollowsUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("error getting feed follows from db: %w", err)
	}

	fmt.Printf("%s is following:\n", s.config.CurrentUserName)
	for _, feedFollow := range feedFollows {
		fmt.Printf(" * %s\n", feedFollow.FeedName)
	}
	return nil
}

func HandlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("unfollow command takes one argument: unfollow <url>")
	}

	ctx := context.Background()
	url := cmd.args[0]
	feed, err := s.db.GetFeedByURL(ctx, url)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("feed not in database")
		}
		return fmt.Errorf("error getting feed by url: %w", err)
	}

	if err = s.db.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{UserID: user.ID, Url: url}); err != nil {
		return fmt.Errorf("error deleting feed follow from url: %w", err)
	}

	fmt.Printf("%s just unfollowed %s\n", user.Name, feed.Name)

	return nil
}
