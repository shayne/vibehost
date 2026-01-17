package sshcmd

import "testing"

func TestRemoteArgsDefaultsAgent(t *testing.T) {
	args := RemoteArgs("myapp", "", nil)
	if len(args) < 4 {
		t.Fatalf("expected at least 4 args, got %d", len(args))
	}
	if args[0] != "vibehost-server" || args[1] != "--agent" || args[2] != "codex" || args[3] != "myapp" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestRemoteArgsIncludesAgentCheckEnv(t *testing.T) {
	t.Setenv("VIBEHOST_AGENT_CHECK", "vibehost-agent-check")
	args := RemoteArgs("myapp", "codex", nil)
	if len(args) < 6 {
		t.Fatalf("expected env-prefixed args, got %v", args)
	}
	if args[0] != "env" || args[1] != "VIBEHOST_AGENT_CHECK=vibehost-agent-check" {
		t.Fatalf("unexpected env prefix: %#v", args[:2])
	}
	if args[2] != "vibehost-server" || args[3] != "--agent" || args[4] != "codex" || args[5] != "myapp" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildArgsTTY(t *testing.T) {
	remote := []string{"vibehost-server", "--agent", "codex", "myapp"}
	args := BuildArgs("host-a", remote, true)
	if len(args) < 2 {
		t.Fatalf("expected args, got %v", args)
	}
	if args[0] != "-tt" {
		t.Fatalf("expected -tt, got %q", args[0])
	}
	if args[1] != "host-a" {
		t.Fatalf("expected host-a, got %q", args[1])
	}
}

func TestBuildArgsNoTTY(t *testing.T) {
	remote := []string{"vibehost-server", "--agent", "codex", "myapp", "snapshot"}
	args := BuildArgs("host-a", remote, false)
	if len(args) < 2 {
		t.Fatalf("expected args, got %v", args)
	}
	if args[0] != "-T" {
		t.Fatalf("expected -T, got %q", args[0])
	}
	if args[1] != "host-a" {
		t.Fatalf("expected host-a, got %q", args[1])
	}
}

func TestBuildArgsWithLocalForward(t *testing.T) {
	remote := []string{"vibehost-server", "--agent", "codex", "myapp"}
	forward := &LocalForward{
		LocalPort:  8080,
		RemoteHost: "localhost",
		RemotePort: 8080,
	}
	args := BuildArgsWithLocalForward("host-a", remote, true, forward)
	if len(args) < 4 {
		t.Fatalf("expected args, got %v", args)
	}
	if args[0] != "-tt" {
		t.Fatalf("expected -tt, got %q", args[0])
	}
	if args[1] != "-L" {
		t.Fatalf("expected -L, got %q", args[1])
	}
	if args[2] != "8080:localhost:8080" {
		t.Fatalf("unexpected forward args: %q", args[2])
	}
	if args[3] != "host-a" {
		t.Fatalf("expected host-a, got %q", args[3])
	}
}
