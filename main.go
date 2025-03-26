package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/IlMeloIl/RSS/internal/config"
	"github.com/IlMeloIl/RSS/internal/database"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

type state struct {
	db     *database.Queries
	config *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	if f, ok := c.cmds[cmd.name]; ok {
		return f(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.name)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("login command needs username argument: login <username>")
	}
	if len(cmd.args) > 1 {
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

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("register command needs username argument: register <username>")
	}
	if len(cmd.args) > 1 {
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

func handlerReset(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("reset command doesn't need argument: reset")
	}

	ctx := context.Background()
	if err := s.db.ResetUsers(ctx); err != nil {
		return fmt.Errorf("error reseting users table: %w", err)
	}

	log.Println("users table reseted")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.args) > 0 {
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

func main() {
	cfg := config.Read()

	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		os.Exit(1)
	}
	defer db.Close()

	dbQueries := database.New(db)

	s := &state{
		db:     dbQueries,
		config: &cfg,
	}

	cmds := &commands{cmds: make(map[string]func(*state, command) error)}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)

	argsPassedByUser := os.Args
	if len(argsPassedByUser) < 2 {
		fmt.Println("No command passed: go run . <command> <args>")
		os.Exit(1)
	}

	cmd := command{name: argsPassedByUser[1], args: argsPassedByUser[2:]}
	if err := cmds.run(s, cmd); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

}
