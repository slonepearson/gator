package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/slonepearson/gator/internal/config"
	"github.com/slonepearson/gator/internal/database"
	"github.com/slonepearson/gator/internal/rss"

	_ "github.com/lib/pq"
)

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
	r.LoadRegistry()

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
