package main

import (
	"context"
	"database/sql"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"gator/internal/rss"
	"io"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
)

type State struct {
	Db     database.Querier
	Config *config.Config
	Client *rss.HttpClient
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	state := &State{Db: database.New(db), Config: &cfg, Client: rss.NewClient()}

	w := io.Writer(os.Stdout)

	r := NewRegistry()
	r.Register("login", "login using your username", "login <username>", 1, false, HandlerLogin)
	r.Register("register", "register a username that doesn't exist", "register <username>", 1, false, HandlerRegister)
	r.Register("reset", "reset all sql tables", "reset", 0, false, HandlerReset)
	r.Register("users", "get all registered users", "users", 0, false, HandlerGetUsers)
	r.Register("agg", "aggregate followed feeds", "agg", 1, false, HandlerAgg)
	r.Register("addfeed", "add and follow a feed", "addfeed <feed name> <feed url>", 2, false, WithLoggedIn(HandlerAddFeed))
	r.Register("feeds", "return all added feeds", "feeds", 0, false, HandlerFeeds)
	r.Register("follow", "follow a feed added by another user", "follow <feed url>", 1, false, WithLoggedIn(HandlerFollow))
	r.Register("following", "return all feeds you are following", "following>", 0, false, WithLoggedIn(HandlerFollowing))
	r.Register("unfollow", "unfollow a feed", "unfollow <feed url>", 1, false, WithLoggedIn(HandlerUnfollow))
	r.Register("browse", "browse latest published posts", "browse <optional limit>", 1, true, WithLoggedIn(HandlerBrowse))
	r.Register("removefeed", "remove an added feed", "removefeed <feed url>", 1, false, HandlerRemoveFeed)

	cmd, err := NewCommand(os.Args[1:]...) // indexed by one to exclude the program's name.
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := r.Run(ctx, w, state, cmd); err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
