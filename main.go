package main

import (
	"fmt"
	"os"

	"github.com/IlMeloIl/RSS/internal/config"
)

type state struct {
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

	s.config.SetUser(cmd.args[0])
	fmt.Println("User has been set to", cmd.args[0])
	return nil
}

func main() {
	cfg := config.Read()

	s := &state{config: &cfg}
	cmds := &commands{cmds: make(map[string]func(*state, command) error)}

	cmds.register("login", handlerLogin)

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
