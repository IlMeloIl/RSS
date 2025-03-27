package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IlMeloIl/RSS/internal/database"
)

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(s *state, cmd command) error {
	return func(s *state, cmd command) error {
		currentUsername := s.config.GetUser().CurrentUserName

		if currentUsername == "" {
			return fmt.Errorf("user must be logged in to user that command")
		}

		ctx := context.Background()
		user, err := s.db.GetUser(ctx, currentUsername)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("user doesn't exist in db")
			}
			return fmt.Errorf("error getting user from db: %w", err)
		}

		return handler(s, cmd, user)

	}
}
