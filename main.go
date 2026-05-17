package main

import (
	"database/sql"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"io"
	"os"

	_ "github.com/lib/pq"
)

type State struct {
	Db     database.Querier
	Config *config.Config
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

	state := &State{Db: database.New(db), Config: &cfg}

	w := io.Writer(os.Stdout)

	handlers := NewRegistry()
	handlers.Register("login", HandlerLogin)
	handlers.Register("register", HandlerRegister)
	handlers.Register("reset", HandlerReset)
	handlers.Register("users", HandlerGetUsers)
	handlers.Register("agg", HandlerAgg)
	handlers.Register("addfeed", HandlerAddFeed)
	handlers.Register("feeds", HandlerFeeds)

	cmd, err := NewCommand(os.Args[1:]...) // indexed by one to exclude the program's name.
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := handlers.Run(w, state, cmd); err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
