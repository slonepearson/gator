package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gator/internal/database"
	"gator/internal/rss"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var ErrInvalidCmd = errors.New("invalid command")
var ErrNotEnoughArgs = errors.New("not enough arguments provided")
var ErrTooManyArgs = errors.New("too many arguments provided")
var ErrAlreadyRegistered = errors.New("username already registered")
var ErrUserNotRegistered = errors.New("user is not registered, use 'register <username>'")
var ErrPostAlreadyAdded = errors.New("Post already added from feed")

type Command struct {
	Name string
	Args []string
}

type RegisteredCommand struct {
	handler      handler
	description  string
	usage        string
	expectedArgs int
	optional     bool
}

type handler func(ctx context.Context, w io.Writer, s *State, cmd Command) error

type Registry struct {
	Handlers map[string]RegisteredCommand
}

func (r *Registry) Register(name string, desc string, usage string, expectedArgs int, optional bool, handler handler) {
	r.Handlers[name] = RegisteredCommand{
		handler:      handler,
		description:  desc,
		usage:        usage,
		expectedArgs: expectedArgs,
		optional:     optional,
	}
}

func (r *Registry) Run(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	regCmd, ok := r.Handlers[cmd.Name]
	if !ok {
		return ErrInvalidCmd
	}
	if !regCmd.optional {
		if len(cmd.Args) != regCmd.expectedArgs {
			return fmt.Errorf("invalid number of arguments.\nUsage: %s\n", regCmd.usage)
		}
	}
	if len(cmd.Args) > regCmd.expectedArgs {
		return fmt.Errorf("invalid number of optional arguments.\nUsage: %s\n", regCmd.usage)
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

	fmt.Fprintf(w, "User %v followed the feed: %v\n", feedFollow.UserName, feedFollow.FeedName)
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

	fmt.Fprintf(w, "%v unfollowed %v\n", user.Name, feed.Name)

	return nil
}

func HandlerAgg(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	timeBetweenReqs, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return err
	}

	if timeBetweenReqs.Seconds() < 20 {
		return fmt.Errorf("time between requests has to be atleast 60 seconds")
	}

	ticker := time.NewTicker(timeBetweenReqs)
	defer ticker.Stop()

	fmt.Fprintf(w, "Scraping posts every %v...\n", timeBetweenReqs)
	if err := scrapeFeeds(ctx, w, s); err != nil {
		fmt.Fprintf(w, "%v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := scrapeFeeds(ctx, w, s); err != nil {
				fmt.Fprintf(w, "%v\n", err)
			}
		}
	}
}

func scrapeFeeds(ctx context.Context, w io.Writer, s *State) error {
	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	feedMeta, err := s.Db.GetNextFeedToFetch(fetchCtx)
	if err != nil {
		return err
	}
	feed, lastModifiedStr, err := rss.FetchFeed(fetchCtx, s.Client, feedMeta.LastModified, feedMeta.Url)
	if err != nil {
		return err
	}
	feed.UnescapeRssFeed()

	now := time.Now()
	sqlNow := sql.NullTime{Time: now, Valid: true}
	lastModified, err := parseDate(lastModifiedStr)
	sqlLastModified := sql.NullTime{Time: lastModified, Valid: err == nil}
	markFeedFetchedParmas := database.MarkFeedFetchedParams{
		ID:            feedMeta.ID,
		LastFetchedAt: sqlNow,
		UpdatedAt:     sqlNow.Time,
		LastModified:  sqlLastModified,
	}
	if err := s.Db.MarkFeedFetched(fetchCtx, markFeedFetchedParmas); err != nil {
		return err
	}
	cancel()
	newPosts := 0
	for _, post := range feed.Channel.Item {
		err := savePost(ctx, s, feedMeta.ID, post, now)
		if err != nil {
			if errors.Is(err, ErrPostAlreadyAdded) {
				continue
			}
			fmt.Fprintf(w, "%v\n", err)
		}
		newPosts++
	}

	if newPosts == 0 {
		fmt.Fprintf(w, "No new posts from feed.\n")
	} else {
		fmt.Fprintf(w, "%d posts added from feed.\n", newPosts)
	}
	return nil
}

func savePost(ctx context.Context, s *State, feedId uuid.UUID, post rss.RSSItem, now time.Time) error {
	postCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	title := sql.NullString{String: post.Title, Valid: false}
	if post.Title != "" {
		title.Valid = true
	}

	description := sql.NullString{String: post.Description, Valid: false}
	if post.Description != "" {
		description.Valid = true
	}

	published_at, err := parseDate(post.PubDate)
	if err != nil {
		return err
	}

	postParams := database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   now,
		Title:       title,
		Url:         post.Link,
		Description: description,
		PublishedAt: published_at,
		FeedID:      feedId,
	}

	_, err = s.Db.CreatePost(postCtx, postParams)
	if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
		return ErrPostAlreadyAdded
	} else if err != nil {
		return err
	}
	return nil
}

func HandlerBrowse(ctx context.Context, w io.Writer, s *State, cmd Command, user database.User) error {

	var limit int32
	if len(cmd.Args) != 0 {
		i, err := strconv.ParseInt(cmd.Args[0], 10, 32)
		if err != nil {
			return err
		}
		limit = int32(i)
	} else {
		limit = 2
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	getPostParams := database.GetPostForUserParams{
		UserID: user.ID,
		Limit:  limit,
	}

	userPosts, err := s.Db.GetPostForUser(ctx, getPostParams)
	if err != nil {
		return err
	}

	for _, post := range userPosts {
		fmt.Fprintf(w, "%v\n", post.Title.String)
		fmt.Fprintf(w, "%v\n", post.Url)
		fmt.Fprintf(w, "Published: %v\n", post.PublishedAt.Format(time.RFC3339))
		fmt.Fprintf(w, "%v\n", post.Description.String)
		fmt.Fprintf(w, "---\n\n")
	}

	return nil
}

func HandlerRemoveFeed(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cencel := context.WithTimeout(ctx, 5*time.Second)
	defer cencel()

	if err := s.Db.RemoveFeed(ctx, cmd.Args[0]); err != nil {
		return err
	}
	return nil
}

func parseDate(dateStr string) (time.Time, error) {
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
