package main

import (
	"context"
	"errors"
	"fmt"
	"gator/internal/database"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCmd = errors.New("invalid command")
var ErrNotEnoughArgs = errors.New("not enough argument provided")
var ErrTooManyArgs = errors.New("too many argument provided")
var ErrAlreadyRegistered = errors.New("username already registered")
var ErrUserNotRegistered = errors.New("user is not registered, use 'register <username>'")

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

func HandlerRegister(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage register <username>: %w ", ErrNotEnoughArgs)
	}
	if len(cmd.Args) > 1 {
		return fmt.Errorf("usage register <username>: %w", ErrTooManyArgs)
	}

	userArgs := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      strings.ToLower(cmd.Args[0]), // prevent duplicates from case sensitivity.
	}
	user, err := s.Db.CreateUser(context.Background(), userArgs)
	if err != nil {
		return ErrAlreadyRegistered
	}

	s.Config.SetUser(user.Name)
	fmt.Fprintf(w, "%v was successfully registered\n", user.Name)
	return nil
}

func HandlerLogin(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage login <username>: %w ", ErrNotEnoughArgs)
	}
	if len(cmd.Args) > 1 {
		return fmt.Errorf("usage login <username>: %w", ErrTooManyArgs)
	}

	user, err := s.Db.GetUser(context.Background(), strings.ToLower(cmd.Args[0]))

	if err != nil {
		return ErrUserNotRegistered
	}

	if err := s.Config.SetUser(user.Name); err != nil {
		return err
	}

	fmt.Fprint(w, "Login successful!\n")
	return nil
}

func HandlerGetUsers(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return ErrTooManyArgs
	}

	users, err := s.Db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	if len(users) == 0 {
		fmt.Fprint(w, "No registered users\n")
		return nil
	}

	currentUser := s.Config.CurrentUserName

	for _, user := range users {
		if user.Name == currentUser {
			fmt.Fprintf(w, "* %v (current)\n", user.Name)
		} else {
			fmt.Fprintf(w, "* %v\n", user.Name)
		}
	}
	return nil
}

func HandlerReset(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return ErrTooManyArgs
	}
	if err := s.Db.ResetUsers(context.Background()); err != nil {
		return err
	}
	s.Config.SetUser("")
	fmt.Fprint(w, "user table has been successfully reset")

	return nil
}
