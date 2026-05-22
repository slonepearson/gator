package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/slonepearson/gator/internal/config"
	"github.com/slonepearson/gator/internal/database"
	"github.com/slonepearson/gator/internal/rss"
)

var ErrInvalidCmd = errors.New("invalid command")
var ErrNotEnoughArgs = errors.New("not enough arguments provided")
var ErrTooManyArgs = errors.New("too many arguments provided")
var ErrAlreadyRegistered = errors.New("username already registered")
var ErrUserNotRegistered = errors.New("user is not registered, use 'register <username>'")
var ErrPostAlreadyAdded = errors.New("Post already added from feed")

type State struct {
	Db     database.Querier
	Config *config.Config
	Client *rss.HttpClient
}

type Command struct {
	Name string
	Args []string
}

type RegisteredCommand struct {
	handler     handlerFunc
	description string
	usage       string
	minArgs     int
	maxArgs     int
	optional    bool
}

type handlerFunc func(ctx context.Context, w io.Writer, s *State, cmd Command) error
type handlerFuncWAuth func(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error

type Registry struct {
	Handlers map[string]RegisteredCommand
}

func (r *Registry) Register(name string, desc string, usage string, minArgs int, maxArgs int, optional bool, handler handlerFunc) {
	r.Handlers[name] = RegisteredCommand{
		handler:     handler,
		description: desc,
		usage:       usage,
		minArgs:     minArgs,
		maxArgs:     maxArgs,
		optional:    optional,
	}
}

func (r *Registry) LoadRegistry() {
	r.Register("login", "login using your username", "login <username>", 1, 1, false, HandlerLogin)
	r.Register("register", "register a username that doesn't exist", "register <username>", 1, 1, false, HandlerRegister)
	r.Register("reset", "reset all sql tables", "reset", 0, 0, false, HandlerReset)
	r.Register("users", "get all registered users", "users", 0, 0, false, HandlerGetUsers)
	r.Register("agg", "aggregate followed feeds", "agg", 1, 1, false, HandlerAgg)
	r.Register("addfeed", "add and follow a feed", "addfeed <feed name> <feed url>", 2, 2, false, WithLoggedIn(HandlerAddFeed))
	r.Register("feeds", "return all added feeds", "feeds", 0, 0, false, HandlerFeeds)
	r.Register("follow", "follow a feed added by another user", "follow <feed url>", 1, 1, false, WithLoggedIn(HandlerFollow))
	r.Register("following", "return all feeds you are following", "following>", 0, 0, false, WithLoggedIn(HandlerFollowing))
	r.Register("unfollow", "unfollow a feed", "unfollow <feed url>", 1, 1, false, WithLoggedIn(HandlerUnfollow))
	r.Register("browse", "browse through the saved posts from feeds you followed", "browse | browse --next | browse --prev | --limit <number> can be used", 0, 3, true, WithLoggedIn(HandlerBrowse))
	r.Register("removefeed", "remove an added feed", "removefeed <feed url>", 1, 1, false, HandlerRemoveFeed)
}

func (r *Registry) Run(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	regCmd, ok := r.Handlers[cmd.Name]
	if !ok {
		return ErrInvalidCmd
	}

	if len(cmd.Args) < regCmd.minArgs {
		return fmt.Errorf("not enough arguments provided.\nUsage: %s\n", regCmd.usage)
	}

	if len(cmd.Args) > regCmd.maxArgs {
		return fmt.Errorf("too many arguments provided.\nUsage: %s\n", regCmd.usage)
	}

	return regCmd.handler(ctx, w, s, cmd)
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

func IsValidUrl(u string) error {
	_, err := url.ParseRequestURI(u)
	if err != nil {
		return err
	}
	return nil
}

func ParseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	layouts := []string{
		time.RFC822,
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		timeFormat, err := time.Parse(layout, dateStr)
		if err == nil {
			return timeFormat, nil
		}
	}
	return time.Time{}, fmt.Errorf("Error: parsing post published date: %v", dateStr)
}
