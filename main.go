package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/IlMeloIl/RSS/internal/config"
	"github.com/IlMeloIl/RSS/internal/database"

	_ "github.com/lib/pq"
)

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

	cmds.register("login", HandlerLogin)
	cmds.register("register", HandlerRegister)
	cmds.register("reset", HandlerReset)
	cmds.register("users", HandlerUsers)
	cmds.register("agg", HandlerAgg)
	cmds.register("addfeed", HandlerAddFeed)
	cmds.register("feeds", HandlerFeeds)

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
