package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/slonepearson/gator/internal/database"
	"github.com/slonepearson/gator/internal/rss"
)

func HandlerRegister(ctx context.Context, w io.Writer, s *State, cmd Command) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now()

	userArgs := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
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

	fmt.Fprintf(w, "Login successful! Welcome %s\n", user.Name)
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
	if err := s.Db.ResetPosts(ctx); err != nil {
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
	now := time.Now()

	feedParams := database.AddFeedParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
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
		CreatedAt: now,
		UpdatedAt: now,
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
		fmt.Fprintf(w, "---\n\n")
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

	now := time.Now()
	feedFollowParams := database.AddFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
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
	lastModified, err := ParseDate(lastModifiedStr)
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
		fmt.Fprintf(w, "No new posts from %v.\n", feedMeta.Name)
	} else {
		fmt.Fprintf(w, "%d new posts from %v.\n", newPosts, feedMeta.Name)
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

	published_at, err := ParseDate(post.PubDate)
	if err != nil {
		return err
	}
	uuid, err := uuid.NewV7()
	if err != nil {
		return err
	}
	postParams := database.CreatePostParams{
		ID:          uuid,
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
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	next := fs.Bool("next", false, "show older posts")
	prev := fs.Bool("prev", false, "show newer posts")
	limit := fs.Int("limit", 2, "limits number of posts")

	if err := fs.Parse(cmd.Args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var userPosts []database.Post

	if *next && s.Config.LastReadBottom != "" && s.Config.LastReadBottomUuid != "" {
		pubAt, err := ParseDate(s.Config.LastReadBottom)
		if err != nil {
			return err
		}
		uuid, err := uuid.Parse(s.Config.LastReadBottomUuid)
		if err != nil {
			return err
		}
		getUserNextPostParams := database.GetUsersNextPostsParams{
			UserID:      user.ID,
			PublishedAt: pubAt,
			ID:          uuid,
			Limit:       int32(*limit),
		}
		nextPosts, err := s.Db.GetUsersNextPosts(ctx, getUserNextPostParams)
		if err != nil {
			return err
		}
		if len(nextPosts) < 1 {
			return fmt.Errorf("no older posts found.")
		}
		userPosts = nextPosts
	} else if *prev && s.Config.LastReadTop != "" && s.Config.LastReadTopUuid != "" {
		pubAt, err := ParseDate(s.Config.LastReadTop)
		if err != nil {
			return err
		}
		uuid, err := uuid.Parse(s.Config.LastReadTopUuid)
		if err != nil {
			return err
		}
		getUserLastPostParams := database.GetUsersLastPostsParams{
			UserID:      user.ID,
			PublishedAt: pubAt,
			ID:          uuid,
			Limit:       int32(*limit),
		}
		prevPosts, err := s.Db.GetUsersLastPosts(ctx, getUserLastPostParams)
		if err != nil {
			return err
		}
		if len(prevPosts) < 1 {
			return fmt.Errorf("no newer posts found")
		}
		slices.Reverse(prevPosts)
		userPosts = prevPosts
	} else {
		getPostParams := database.GetPostForUserParams{
			UserID: user.ID,
			Limit:  int32(*limit),
		}
		posts, err := s.Db.GetPostForUser(ctx, getPostParams)
		if err != nil {
			return err
		}
		if len(posts) < 1 {
			return fmt.Errorf("no posts have been added")
		}
		userPosts = posts
	}

	numOfPosts := len(userPosts)
	err := s.Config.SetLastRead(
		userPosts[0].PublishedAt.Format(time.RFC3339),
		userPosts[numOfPosts-1].PublishedAt.Format(time.RFC3339),
		userPosts[0].ID.String(),
		userPosts[numOfPosts-1].ID.String(),
	)
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
