package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"vibehost/internal/server"
)

const defaultImage = "vibehost:latest"

func main() {
	fs := flag.NewFlagSet("vibehost-server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agentProvider := fs.String("agent", "codex", "agent provider to run (codex, claude, gemini)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost-server [--agent provider] <app>")
		os.Exit(2)
	}

	app := strings.TrimSpace(fs.Arg(0))
	if app == "" {
		fmt.Fprintln(os.Stderr, "app name is required")
		os.Exit(2)
	}

	agentArgs, err := agentCommand(*agentProvider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid agent provider: %v\n", err)
		os.Exit(2)
	}

	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(os.Stderr, "docker is required but was not found in PATH")
		os.Exit(1)
	}

	state, statePath, err := server.LoadState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load server state: %v\n", err)
		os.Exit(1)
	}

	containerName := fmt.Sprintf("vibehost-%s", app)
	exists, err := containerExists(containerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to inspect container: %v\n", err)
		os.Exit(1)
	}

	port, ok := state.PortForApp(app)
	stateDirty := false
	if exists && !ok {
		if discovered, found, err := containerPort(containerName); err != nil {
			fmt.Fprintf(os.Stderr, "failed to read container port: %v\n", err)
			os.Exit(1)
		} else if found {
			state.SetPort(app, discovered)
			port = discovered
			stateDirty = true
		}
	}

	if port == 0 {
		port = state.AssignPort(app)
		stateDirty = true
	}

	if !exists {
		if !promptCreate(app) {
			fmt.Fprintln(os.Stderr, "aborted")
			os.Exit(1)
		}

		if err := dockerRun(containerName, port); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create container: %v\n", err)
			os.Exit(1)
		}
	} else {
		running, err := containerRunning(containerName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to check container state: %v\n", err)
			os.Exit(1)
		}
		if !running {
			if err := dockerStart(containerName); err != nil {
				fmt.Fprintf(os.Stderr, "failed to start container: %v\n", err)
				os.Exit(1)
			}
		}
	}

	if stateDirty {
		if err := server.SaveState(statePath, state); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save server state: %v\n", err)
			os.Exit(1)
		}
	}

	if err := dockerExec(containerName, agentArgs); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "failed to exec shell: %v\n", err)
		os.Exit(1)
	}
}

func promptCreate(app string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "App %s does not exist. Create? [Y/n]: ", app)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" || input == "y" || input == "yes" {
		return true
	}
	return false
}

func containerExists(name string) (bool, error) {
	cmd := exec.Command("docker", "inspect", name)
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func containerRunning(name string) (bool, error) {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name).Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func containerPort(name string) (int, bool, error) {
	out, err := exec.Command("docker", "port", name, "8080/tcp").Output()
	if err != nil {
		return 0, false, err
	}

	re := regexp.MustCompile(`:(\d+)$`)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		port, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		return port, true, nil
	}

	return 0, false, nil
}

func dockerRun(name string, port int) error {
	args := []string{
		"run",
		"-d",
		"--name",
		name,
		"-p",
		fmt.Sprintf("%d:8080", port),
		defaultImage,
		"sleep",
		"infinity",
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dockerStart(name string) error {
	cmd := exec.Command("docker", "start", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dockerExec(name string, agentArgs []string) error {
	if len(agentArgs) == 0 {
		agentArgs = []string{"/bin/bash"}
	}
	args := append([]string{"exec", "-it", name}, agentArgs...)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func agentCommand(provider string) ([]string, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	switch normalized {
	case "", "codex":
		return []string{"codex"}, nil
	case "claude", "claude-code":
		return []string{"claude"}, nil
	case "gemini":
		return []string{"gemini"}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}
