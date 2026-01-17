package sshcmd

import (
	"fmt"
	"os"
	"strings"
)

// RemoteArgs builds the remote command executed on the server host.
func RemoteArgs(app string, agentProvider string, actionArgs []string) []string {
	if strings.TrimSpace(agentProvider) == "" {
		agentProvider = "codex"
	}
	remote := []string{"vibehost-server", "--agent", agentProvider, app}
	remote = append(remote, actionArgs...)
	if value := strings.TrimSpace(os.Getenv("VIBEHOST_AGENT_CHECK")); value != "" {
		prefix := []string{"env", "VIBEHOST_AGENT_CHECK=" + value}
		return append(prefix, remote...)
	}
	return remote
}

// BuildArgs builds the ssh argument list for a target host and remote command.
func BuildArgs(host string, remoteArgs []string, tty bool) []string {
	return BuildArgsWithLocalForward(host, remoteArgs, tty, nil)
}

type LocalForward struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
}

// BuildArgsWithLocalForward builds the ssh argument list for a target host and optional local forward.
func BuildArgsWithLocalForward(host string, remoteArgs []string, tty bool, forward *LocalForward) []string {
	args := []string{}
	if tty {
		args = append(args, "-tt")
	} else {
		args = append(args, "-T")
	}
	if forward != nil {
		remoteHost := strings.TrimSpace(forward.RemoteHost)
		if remoteHost == "" {
			remoteHost = "localhost"
		}
		args = append(args, "-L", fmt.Sprintf("%d:%s:%d", forward.LocalPort, remoteHost, forward.RemotePort))
	}
	args = append(args, host)
	return append(args, remoteArgs...)
}
