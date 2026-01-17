package sshcmd

import "fmt"

// RemoteArgs builds the remote command executed on the server host.
func RemoteArgs(app string, agentProvider string) []string {
	if agentProvider == "" {
		agentProvider = "codex"
	}
	return []string{fmt.Sprintf("vibehost-server --agent %s %s", agentProvider, app)}
}

// BuildArgs builds the ssh argument list for a target host and remote command.
func BuildArgs(host string, remoteArgs []string) []string {
	args := []string{"-tt", host}
	return append(args, remoteArgs...)
}
