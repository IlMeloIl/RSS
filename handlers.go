package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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

func parseRSSDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822,
		time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("couldn't parse date: %s", dateStr)
}

func stripHTMLTags(s string) string {
	re := regexp.MustCompile("<[^>]*>")
	s = re.ReplaceAllString(s, "")

	s = strings.TrimSpace(s)

	return s
}

func scrapeFeeds(s *state) error {

	ctx := context.Background()

	feeds, err := s.db.GetFeeds(ctx)
	if err != nil {
		return fmt.Errorf("error getting feeds: %w", err)
	}
	fmt.Printf("Total feeds in database: %d\n", len(feeds))

	for _, feedInfo := range feeds {
		feed, err := s.db.GetFeedByURL(ctx, feedInfo.Url)
		if err != nil {
			continue
		}
		fmt.Printf("Feed: %s, Last fetched: %v\n", feed.Name, feed.LastFetchedAt)
	}

	feed, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("error getting next feed to fetch: %w", err)
	}

	fmt.Printf("\nFETCHING FEED: %s (ID: %s)\n", feed.Name, feed.ID)
	fmt.Printf("Last fetched at: %v\n", feed.LastFetchedAt)

	if err = s.db.MarkFeedFetched(ctx, feed.ID); err != nil {
		return fmt.Errorf("error marking feed as fetched: %w", err)
	}

	updatedFeed, _ := s.db.GetFeedByURL(ctx, feed.Url)
	fmt.Printf("Feed marked as fetched. New last_fetched_at: %v\n", updatedFeed.LastFetchedAt)

	rssFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("error fetching feed in scrape feeds: %w", err)
	}

	for _, item := range rssFeed.Channel.Item {

		var publishedAt sql.NullTime
		if item.PubDate != "" {
			parsedTime, err := parseRSSDate(item.PubDate)
			if err != nil {
				fmt.Printf("Warning: couldn't parse date '%s': %v\n", item.PubDate, err)
			} else {
				publishedAt = sql.NullTime{
					Time:  parsedTime,
					Valid: true,
				}
			}
		}

		description := item.Description
		cleanDescription := stripHTMLTags(description)

		if cleanDescription == "" || cleanDescription == "Comments" {
			cleanDescription = "[No description available]"
		}

		_, err := s.db.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       sql.NullString{String: item.Title, Valid: true},
			Url:         item.Link,
			Description: sql.NullString{String: cleanDescription, Valid: cleanDescription != ""},
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") ||
				strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			fmt.Printf("Error saving post: %v\n", err)
		} else {
			fmt.Printf("Saved post: %s\n", item.Title)
		}
	}
	return nil
}

func HandlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("command agg needs one argument: agg <time_between_reqs> --> <5s>, <1m>, <1h>")
	}

	timeBetweenReqsStr := cmd.args[0]
	timeBetweenReqs, err := time.ParseDuration(timeBetweenReqsStr)
	if err != nil {
		return fmt.Errorf("error parsing time duration: %w", err)
	}

	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			fmt.Printf("error scraping feeds: %v", err)
		}
	}
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

func HandlerBrowse(s *state, cmd command, user database.User) error {
	var limit int32 = 2

	if len(cmd.args) > 1 {
		return fmt.Errorf("browse takes at most one arguemnt: browse <limit>")
	}

	if len(cmd.args) == 1 {
		parsedLimit, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("error converting arg into int")
		}
		limit = int32(parsedLimit)
	}

	ctx := context.Background()
	posts, err := s.db.GetPostsForUser(ctx, database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("error getting posts for user: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("no posts found")
		return nil
	}

	fmt.Printf("Found %d posts:\n\n", len(posts))

	for i, post := range posts {
		fmt.Printf("Post #%d:\n", i+1)
		if post.Title.Valid {
			fmt.Printf("Title: %s\n", post.Title.String)
		} else {
			fmt.Println("Title: [No title]")
		}
		fmt.Printf("Feed: %s\n", post.FeedName)

		if post.PublishedAt.Valid {
			fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format(time.RFC1123))
		}

		fmt.Printf("URL: %s\n", post.Url)

		if post.Description.Valid && post.Description.String != "" {
			descriptionPreview := post.Description.String
			if len(descriptionPreview) > 100 {
				descriptionPreview = descriptionPreview[:100] + "..."
			}
			fmt.Printf("Description: %s\n", descriptionPreview)
		}

		fmt.Println(strings.Repeat("-", 50))
	}
	return nil
}
