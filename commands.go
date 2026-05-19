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
var ErrNotEnoughArgs = errors.New("not enough arguments provided")
var ErrTooManyArgs = errors.New("too many arguments provided")
var ErrAlreadyRegistered = errors.New("username already registered")
var ErrUserNotRegistered = errors.New("user is not registered, use 'register <username>'")

type Command struct {
	Name string
	Args []string
}

type RegisteredCommand struct {
	h            handler
	description  string
	usage        string
	expectedArgs int
}

type handler func(ctx context.Context, w io.Writer, s *State, cmd Command) error

type Registry struct {
	Handlers map[string]RegisteredCommand
}

func (c *Registry) Register(name string, desc string, usage string, expectedArgs int, handler handler) {
	c.Handlers[name] = RegisteredCommand{
		h:            handler,
		description:  desc,
		usage:        usage,
		expectedArgs: expectedArgs,
	}
}

func (c *Registry) Run(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	command, ok := c.Handlers[cmd.Name]
	if !ok {
		return ErrInvalidCmd
	}

	return command.h(ctx, w, s, cmd)
}

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

func NewRegistry() *Registry {
	return &Registry{Handlers: map[string]RegisteredCommand{}}
}

func WithExpectArgs(r *Registry, next handler) handler {
	return func(ctx context.Context, w io.Writer, s *State, cmd Command) error {
		regCmd, ok := r.Handlers[cmd.Name]
		if !ok {
			return ErrInvalidCmd
		}
		if len(cmd.Args) != regCmd.expectedArgs {
			return fmt.Errorf("invalid number of arguments.\nUsage: %s\n", regCmd.usage)
		}
		return next(ctx, w, s, cmd)
	}
}

func WithLoggedIn(next func(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error) handler {
	return func(ctx context.Context, w io.Writer, s *State, cmd Command) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		user, err := s.Db.GetUserByName(ctx, s.Config.CurrentUserName)
		if err != nil {
			return err
		}
		return next(ctx, w, s, cmd, user)
	}
}

func IsValidUrl(u string) error {
	_, err := url.ParseRequestURI(u)
	if err != nil {
		return err
	}
	return nil
}

func HandlerRegister(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	userArgs := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      strings.ToLower(cmd.Args[0]), // prevent duplicates from case sensitivity.
	}
	user, err := s.Db.CreateUser(ctx, userArgs)
	if err != nil {
		return ErrAlreadyRegistered
	}

	s.Config.SetUser(user.Name)
	fmt.Fprintf(w, "%v was successfully registered\n", user.Name)
	return nil
}

func HandlerLogin(ctx context.Context, w io.Writer, s *State, cmd Command) error {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user, err := s.Db.GetUserByName(ctx, strings.ToLower(cmd.Args[0]))

	if err != nil {
		return ErrUserNotRegistered
	}

	if err := s.Config.SetUser(user.Name); err != nil {
		return err
	}

	fmt.Fprint(w, "Login successful!\n")
	return nil
}

func HandlerGetUsers(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

func HandlerReset(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

func HandlerAgg(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	feedUrl := "https://www.wagslane.dev/index.xml"

	rssFeed, err := rss.FetchFeed(s.Client, ctx, feedUrl)

	if err != nil {
		return err
	}

	rssFeed.UnescapeRssFeed()
	fmt.Fprint(w, rssFeed)
	return nil
}

func HandlerAddFeed(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := IsValidUrl(cmd.Args[1]); err != nil {
		return fmt.Errorf("invalid URL: %v", cmd.Args[0])
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

func HandlerFeeds(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

func HandlerFollow(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

func HandlerFollowing(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

func HandlerUnfollow(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	feed, err := s.Db.GetFeedByUrl(ctx, cmd.Args[0])
	if err != nil {
		return err
	}

	feedFollowParams := database.RemoveFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}

	if err := s.Db.RemoveFeedFollow(ctx, feedFollowParams); err != nil {
		return err
	}

	fmt.Fprintf(w, "%v unfollowed %v", user.Name, feed.Name)

	return nil
}
