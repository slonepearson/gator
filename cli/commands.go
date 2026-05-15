package main

import (
	"errors"
	"fmt"
	"io"
)

var ErrInvalidCmd = errors.New("invalid command")
var ErrNotEnoughArgs = errors.New("not enough argument provided")
var ErrTooManyArgs = errors.New("too many argument provided")

type Command struct {
	Name string
	Args []string
}

type Registry struct {
	Handlers map[string]func(w io.Writer, s *State, cmd Command) error
}

func (c *Registry) Register(name string, handler func(w io.Writer, s *State, cmd Command) error) {
	c.Handlers[name] = handler
}

func (c *Registry) Run(w io.Writer, s *State, cmd Command) error {
	command, ok := c.Handlers[cmd.Name]
	if !ok {
		return ErrInvalidCmd
	}

	return command(w, s, cmd)
}

/*
This function returns a new Command struct or an error.
The first provided argument will be considered the commands name.
Every argument after that will be considered the command's arguments.
An ErrNotEnoughArgs error will be returned if a command name is not provided.
*/
func NewCommand(args ...string) (Command, error) {
	if len(args) < 1 {
		return Command{}, ErrNotEnoughArgs
	}

	commandName := args[0]
	commandArgs := []string{}
	if len(args) > 1 {
		commandArgs = args[1:]
	}
	cmd := Command{Name: commandName, Args: commandArgs}
	return cmd, nil
}

func NewRegistry() Registry {
	return Registry{Handlers: map[string]func(w io.Writer, s *State, cmd Command) error{}}
}

func HandlerLogin(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage login <username>: %w ", ErrNotEnoughArgs)
	}
	if len(cmd.Args) > 1 {
		return fmt.Errorf("usage login <username>: %w", ErrTooManyArgs)
	}
	if err := s.Config.SetUser(cmd.Args[0]); err != nil {
		return err
	}
	fmt.Fprint(w, "login successful!\n")
	return nil
}
