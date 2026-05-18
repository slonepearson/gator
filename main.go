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
	"time"

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
	if err := db.Ping(); err != nil {
		fmt.Printf("Error: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	state := &State{
		Db:     database.New(db),
		Config: &cfg,
		Client: rss.NewClient(30 * time.Second),
	}

	w := io.Writer(os.Stdout)

	handlers := NewRegistry()
	handlers.Register("login", HandlerLogin)
	handlers.Register("register", HandlerRegister)
	handlers.Register("reset", HandlerReset)
	handlers.Register("users", HandlerGetUsers)
	handlers.Register("agg", HandlerAgg)
	handlers.Register("addfeed", WithLoggedIn(HandlerAddFeed))
	handlers.Register("feeds", HandlerFeeds)
	handlers.Register("follow", WithLoggedIn(HandlerFollow))
	handlers.Register("following", WithLoggedIn(HandlerFollowing))

	cmd, err := NewCommand(os.Args[1:]...) // indexed by one to exclude the program's name.
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := handlers.Run(ctx, w, state, cmd); err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
