package main

import (
	"fmt"
	"os"
	"os/exec"

	"vibehost/internal/config"
	"vibehost/internal/sshcmd"
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

	if _, err := exec.LookPath("ssh"); err != nil {
		fmt.Fprintln(os.Stderr, "ssh is required but was not found in PATH")
		os.Exit(1)
	}

	remoteArgs := sshcmd.RemoteArgs(resolved.App)
	sshArgs := sshcmd.BuildArgs(resolved.Host, remoteArgs)

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "failed to start ssh: %v\n", err)
		os.Exit(1)
	}
}
