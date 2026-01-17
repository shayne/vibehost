package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"vibehost/internal/config"
	"vibehost/internal/sshcmd"
	"vibehost/internal/target"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost <app> | vibehost <app>@<host> | vibehost <app> snapshot | vibehost <app> snapshots | vibehost <app> restore <snapshot> | vibehost config [options]")
		os.Exit(2)
	}

	if args[0] == "config" {
		handleConfig(args[1:])
		return
	}

	if len(args) > 3 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost <app> | vibehost <app>@<host> | vibehost <app> snapshot | vibehost <app> snapshots | vibehost <app> restore <snapshot> | vibehost config [options]")
		os.Exit(2)
	}

	targetArg := args[0]
	actionArgs := []string{}
	if len(args) == 2 {
		switch args[1] {
		case "snapshot":
			actionArgs = []string{"snapshot"}
		case "snapshots":
			actionArgs = []string{"snapshots"}
		default:
			fmt.Fprintln(os.Stderr, "Usage: vibehost <app> snapshot | vibehost <app> snapshots")
			os.Exit(2)
		}
	}
	if len(args) == 3 {
		if args[1] != "restore" || strings.TrimSpace(args[2]) == "" {
			fmt.Fprintln(os.Stderr, "Usage: vibehost <app> restore <snapshot>")
			os.Exit(2)
		}
		actionArgs = []string{"restore", args[2]}
	}

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	resolved, err := target.Resolve(targetArg, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid target: %v\n", err)
		os.Exit(2)
	}

	if _, err := exec.LookPath("ssh"); err != nil {
		fmt.Fprintln(os.Stderr, "ssh is required but was not found in PATH")
		os.Exit(1)
	}

	agentProvider := cfg.AgentProvider
	if strings.TrimSpace(agentProvider) == "" {
		agentProvider = "codex"
	}
	remoteArgs := sshcmd.RemoteArgs(resolved.App, agentProvider, actionArgs)
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

type hostPairs []string

func (h *hostPairs) String() string {
	return strings.Join(*h, ",")
}

func (h *hostPairs) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("host mapping cannot be empty")
	}
	*h = append(*h, value)
	return nil
}

func handleConfig(args []string) {
	fs := flag.NewFlagSet("vibehost config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	defaultHost := fs.String("default-host", "", "set default host")
	agent := fs.String("agent", "", "set default agent provider")
	var setHosts hostPairs
	fs.Var(&setHosts, "set-host", "set host alias mapping as alias=host (repeatable)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	cfg, path, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if fs.NFlag() == 0 && fs.NArg() == 0 {
		showConfig(cfg, path)
		return
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected extra arguments for config command")
		os.Exit(2)
	}

	updated := false
	if *defaultHost != "" {
		cfg.DefaultHost = *defaultHost
		updated = true
	}
	if *agent != "" {
		cfg.AgentProvider = *agent
		updated = true
	}
	if len(setHosts) > 0 {
		if cfg.Hosts == nil {
			cfg.Hosts = map[string]string{}
		}
		for _, entry := range setHosts {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "invalid host mapping %q (expected alias=host)\n", entry)
				os.Exit(2)
			}
			alias := strings.TrimSpace(parts[0])
			host := strings.TrimSpace(parts[1])
			if alias == "" || host == "" {
				fmt.Fprintf(os.Stderr, "invalid host mapping %q (expected alias=host)\n", entry)
				os.Exit(2)
			}
			cfg.Hosts[alias] = host
		}
		updated = true
	}
	if !updated {
		showConfig(cfg, path)
		return
	}

	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "wrote config to %s\n", path)
}

func showConfig(cfg config.Config, path string) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to format config: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Config path: %s\n%s\n", path, string(data))
}
