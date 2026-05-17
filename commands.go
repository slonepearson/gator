package main

import (
	"context"
	"errors"
	"fmt"
	"gator/internal/database"
	"gator/internal/rss"
	"io"
	"net/url"
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

	user, err := s.Db.GetUserByName(context.Background(), strings.ToLower(cmd.Args[0]))

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
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	users, err := s.Db.GetUsers(ctx)
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
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	if err := s.Db.ResetUsers(ctx); err != nil {
		return err
	}
	if err := s.Db.ResetFeeds(ctx); err != nil {
		return err
	}

	s.Config.SetUser("")
	fmt.Fprint(w, "Tables have been successfully reset")

	return nil
}

func HandlerAgg(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) > 1 {
		return ErrTooManyArgs
	}

	feedUrl := "https://www.wagslane.dev/index.xml"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rssFeed, err := rss.FetchFeed(ctx, feedUrl)
	if err != nil {
		return err
	}
	rssFeed.UnescapeRssFeed()
	fmt.Fprint(w, rssFeed)
	return nil
}

func HandlerAddFeed(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) < 2 {
		return ErrNotEnoughArgs
	}
	if len(cmd.Args) > 2 {
		return ErrTooManyArgs
	}
	_, err := url.ParseRequestURI(cmd.Args[1])
	if err != nil {
		return fmt.Errorf("invalid URL: %v", cmd.Args[1])
	}

	ctx, cancle := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancle()

	user, err := s.Db.GetUserByName(ctx, s.Config.CurrentUserName)
	if err != nil {
		return err
	}
	feedParams := database.AddFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Args[0],
		Url:       cmd.Args[1],
		UserID:    user.ID,
	}

	feed, err := s.Db.AddFeed(ctx, feedParams)
	if err != nil {
		return err
	}

	feedFollowParams := database.AddFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	_, err = s.Db.AddFeedFollow(ctx, feedFollowParams)

	if err != nil {
		return err
	}

	fmt.Fprintf(w, "%v has added and followed feed %v", user.Name, feed.Name)
	return nil
}

func HandlerFeeds(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return ErrTooManyArgs
	}

	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()

	feeds, err := s.Db.GetFeeds(ctx)

	if err != nil {
		return err
	}
	if len(feeds) < 1 {
		fmt.Fprint(w, "no feeds have been added")
		return nil
	}

	for _, feed := range feeds {
		user, err := s.Db.GetUserById(ctx, feed.UserID)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "Feed name: %v\n", feed.Name)
		fmt.Fprintf(w, "Feed URL: %v\n", feed.Url)
		fmt.Fprintf(w, "Created by: %v\n", user.Name)
	}
	return nil
}

func HandlerFollow(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return ErrNotEnoughArgs
	}
	if len(cmd.Args) > 1 {
		return ErrTooManyArgs
	}

	ctx, cancle := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancle()

	user, err := s.Db.GetUserByName(ctx, s.Config.CurrentUserName)
	if err != nil {
		return err
	}

	feed, err := s.Db.GetFeedByUrl(ctx, cmd.Args[0])
	if err != nil {
		return err
	}

	feedFollowParams := database.AddFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	feedFollow, err := s.Db.AddFeedFollow(ctx, feedFollowParams)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "User %v followed the feed: %v", feedFollow.UserName, feedFollow.FeedName)
	return nil
}

func HandlerFollowing(w io.Writer, s *State, cmd Command) error {
	if len(cmd.Args) > 0 {
		return ErrTooManyArgs
	}

	ctx, cancle := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancle()

	user, err := s.Db.GetUserByName(ctx, s.Config.CurrentUserName)
	if err != nil {
		return err
	}

	feeds, err := s.Db.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		return nil
	}

	if len(feeds) < 1 {
		fmt.Fprintf(w, "%v is not following any feeds\n", user.Name)
		return nil
	}

	fmt.Fprintf(w, "%v is following:\n", user.Name)
	for _, feed := range feeds {
		fmt.Fprintf(w, "* %v\n", feed.FeedName)
	}
	return nil
}
