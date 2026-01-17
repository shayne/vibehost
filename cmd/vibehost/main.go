package main

import (
	"fmt"
	"os"

	"vibehost/internal/config"
	"vibehost/internal/target"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost <app> or vibehost <app>@<host>")
		os.Exit(2)
	}

	arg := os.Args[1]
	if arg == "config" {
		fmt.Fprintln(os.Stderr, "config command not implemented yet")
		os.Exit(2)
	}

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	resolved, err := target.Resolve(arg, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid target: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("vibehost target resolved: app=%s host=%s\n", resolved.App, resolved.Host)
	fmt.Println("server connection not implemented yet")
}
