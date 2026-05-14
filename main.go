package main

import (
	"fmt"
	"gator/internal/commands"
	"gator/internal/config"
	"io"
	"os"
)

func main() {
	state, err := config.NewState()
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	w := io.Writer(os.Stdin)
	handlers := commands.NewRegistry()
	handlers.Register("login", commands.HandlerLogin)

	cmd, err := commands.NewCommand(os.Args[1:]...) // indexed by one to exclude the program's name.
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := handlers.Run(w, state, cmd); err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		os.Exit(1)
	}
}
