package main

import (
	"fmt"
	"gator/internal/config"
	"io"
	"os"
)

func main() {
	cfg, err := config.Read()
	w := io.Writer(os.Stdin)
	if err != nil {
		fmt.Fprintf(w, "%v\n", err)
	}
	cfg.SetUser("jessie")

	cfg, err = config.Read()
	if err != nil {
		fmt.Fprintf(w, "%v\n", err)
	}
	fmt.Printf("%#v", cfg)
}
