package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/shayne/vibehost/internal/config"
	"github.com/shayne/vibehost/internal/sshcmd"
	"github.com/shayne/vibehost/internal/target"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost [--agent provider] <app> | vibehost [--agent provider] <app>@<host> | vibehost [--agent provider] <app> snapshot | vibehost [--agent provider] <app> snapshots | vibehost [--agent provider] <app> restore <snapshot> | vibehost <app> shell | vibehost bootstrap [<host>] | vibehost config [options]")
		return
	}

	if args[0] == "config" {
		handleConfig(args[1:])
		return
	}
	if args[0] == "bootstrap" {
		handleBootstrap(args[1:])
		return
	}

	fs := flag.NewFlagSet("vibehost", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agentOverride := fs.String("agent", "", "agent provider override (codex, claude, gemini)")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	remaining := fs.Args()
	if len(remaining) < 1 || len(remaining) > 3 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost [--agent provider] <app> | vibehost [--agent provider] <app>@<host> | vibehost [--agent provider] <app> snapshot | vibehost [--agent provider] <app> snapshots | vibehost [--agent provider] <app> restore <snapshot> | vibehost <app> shell | vibehost bootstrap [<host>] | vibehost config [options]")
		os.Exit(2)
	}

	targetArg := remaining[0]
	actionArgs := []string{}
	if len(remaining) == 2 {
		switch remaining[1] {
		case "snapshot":
			actionArgs = []string{"snapshot"}
		case "snapshots":
			actionArgs = []string{"snapshots"}
		case "shell":
			actionArgs = []string{"shell"}
		default:
			fmt.Fprintln(os.Stderr, "Usage: vibehost [--agent provider] <app> snapshot | vibehost [--agent provider] <app> snapshots | vibehost <app> shell")
			os.Exit(2)
		}
	}
	if len(remaining) == 3 {
		if remaining[1] != "restore" || strings.TrimSpace(remaining[2]) == "" {
			fmt.Fprintln(os.Stderr, "Usage: vibehost [--agent provider] <app> restore <snapshot>")
			os.Exit(2)
		}
		actionArgs = []string{"restore", remaining[2]}
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
	if strings.TrimSpace(*agentOverride) != "" {
		agentProvider = *agentOverride
	}
	interactive := len(actionArgs) == 0 || (len(actionArgs) == 1 && actionArgs[0] == "shell")
	tty := interactive && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	if interactive && !tty {
		fmt.Fprintln(os.Stderr, "interactive sessions require a TTY; run from a terminal or use snapshot/restore commands")
		os.Exit(1)
	}
	extraEnv := map[string]string{}
	var openServer *http.Server
	var remoteSocket *sshcmd.RemoteSocketForward
	if interactive {
		server, port, err := startOpenListener()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to start xdg-open listener: %v\n", err)
			os.Exit(1)
		}
		openServer = server
		extraEnv["VIBEHOST_XDG_OPEN_SOCKET"] = xdgOpenSocketPath()
		remoteSocket = &sshcmd.RemoteSocketForward{
			RemotePath: xdgOpenSocketPath(),
			LocalHost:  "localhost",
			LocalPort:  port,
		}
	}
	remoteArgs := sshcmd.RemoteArgs(resolved.App, agentProvider, actionArgs, extraEnv)
	var forward *sshcmd.LocalForward
	if interactive && !isLocalHost(resolved.Host) {
		hostPort, err := resolveHostPort(resolved, agentProvider)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := ensureLocalPortAvailable(hostPort); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		forward = &sshcmd.LocalForward{
			LocalPort:  hostPort,
			RemoteHost: "localhost",
			RemotePort: hostPort,
		}
	}

	sshArgs := sshcmd.BuildArgsWithForwards(resolved.Host, remoteArgs, tty, forward, remoteSocket)

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if openServer != nil {
			_ = openServer.Close()
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "failed to start ssh: %v\n", err)
		os.Exit(1)
	}
	if openServer != nil {
		_ = openServer.Close()
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

	host := fs.String("host", "", "set default host (alias for --default-host)")
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
	resolvedHost := ""
	if *host != "" && *defaultHost != "" && *host != *defaultHost {
		fmt.Fprintln(os.Stderr, "conflicting --host and --default-host values")
		os.Exit(2)
	}
	if *host != "" {
		resolvedHost = *host
	} else if *defaultHost != "" {
		resolvedHost = *defaultHost
	}
	if resolvedHost != "" {
		cfg.DefaultHost = resolvedHost
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

func handleBootstrap(args []string) {
	fs := flag.NewFlagSet("vibehost bootstrap", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "Usage: vibehost bootstrap [<host>]")
		os.Exit(2)
	}

	hostArg := ""
	if fs.NArg() == 1 {
		hostArg = fs.Arg(0)
	}

	cfg, path, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	resolved, err := target.ResolveHost(hostArg, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid host: %v\n", err)
		os.Exit(2)
	}

	if _, err := exec.LookPath("ssh"); err != nil {
		fmt.Fprintln(os.Stderr, "ssh is required but was not found in PATH")
		os.Exit(1)
	}

	tty := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	if !tty {
		fmt.Fprintln(os.Stderr, "bootstrap may require sudo; run from a terminal if you are prompted for a password")
	}

	command := bootstrapCommand(bootstrapScript())
	remoteArgs := []string{"bash", "-lc", command}
	sshArgs := sshcmd.BuildArgs(resolved.Host, remoteArgs, tty)
	sshArgs = append([]string{"-o", "LogLevel=ERROR"}, sshArgs...)

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
	if strings.TrimSpace(cfg.DefaultHost) == "" && strings.TrimSpace(hostArg) != "" {
		cfg.DefaultHost = hostArg
		if err := config.Save(path, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap complete, but failed to save default host: %v\n", err)
			fmt.Fprintf(os.Stderr, "run `vibehost config --host %s` to set it manually\n", hostArg)
		} else {
			fmt.Fprintf(os.Stdout, "default host set to %s\n", hostArg)
		}
	}
	fmt.Fprintln(os.Stdout, "bootstrap complete")
}

func bootstrapScript() string {
	return `set -euo pipefail

if [ ! -f /etc/os-release ]; then
  echo "missing /etc/os-release; cannot verify OS" >&2
  exit 1
fi

. /etc/os-release
if [ "${ID:-}" != "ubuntu" ]; then
  echo "unsupported OS: ${ID:-unknown}; expected ubuntu" >&2
  exit 1
fi

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  if ! need_cmd sudo; then
    echo "sudo is required to bootstrap as a non-root user" >&2
    exit 1
  fi
  SUDO="sudo"
  if ! sudo -n true 2>/dev/null; then
    echo "sudo password may be required during bootstrap" >&2
  fi
fi

if ! need_cmd curl && ! need_cmd wget; then
  $SUDO apt-get update -y
  $SUDO apt-get install -y curl ca-certificates
fi

if ! need_cmd docker; then
  if need_cmd curl; then
    curl -fsSL https://get.docker.com | $SUDO sh
  else
    wget -qO- https://get.docker.com | $SUDO sh
  fi
fi

if need_cmd systemctl; then
  $SUDO systemctl enable --now docker
fi

if [ "$(id -u)" -ne 0 ]; then
  if ! getent group docker >/dev/null 2>&1; then
    $SUDO groupadd docker
  fi
  if ! id -nG "$USER" | tr ' ' '\n' | grep -qx docker; then
    $SUDO usermod -aG docker "$USER"
    echo "added $USER to docker group; run 'newgrp docker' or reconnect to apply" >&2
  fi
fi

VIBEHOST_SERVER_REPO="${VIBEHOST_SERVER_REPO:-shayne/vibehost}"
VIBEHOST_SERVER_VERSION="${VIBEHOST_SERVER_VERSION:-latest}"
VIBEHOST_SERVER_INSTALL_DIR="${VIBEHOST_SERVER_INSTALL_DIR:-/usr/local/bin}"
VIBEHOST_SERVER_BIN="${VIBEHOST_SERVER_BIN:-vibehost-server}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64|amd64)
    arch="amd64"
    ;;
  arm64|aarch64)
    arch="arm64"
    ;;
  *)
    echo "unsupported architecture: $arch_raw" >&2
    exit 1
    ;;
esac

if [ "$os" != "linux" ]; then
  echo "unsupported OS: $os; expected linux" >&2
  exit 1
fi

asset="vibehost-server-${os}-${arch}"
if [ "$VIBEHOST_SERVER_VERSION" = "latest" ]; then
  download_url="https://github.com/${VIBEHOST_SERVER_REPO}/releases/latest/download/${asset}"
else
  version="$VIBEHOST_SERVER_VERSION"
  case "$version" in
    v*)
      ;;
    *)
      version="v$version"
      ;;
  esac
  download_url="https://github.com/${VIBEHOST_SERVER_REPO}/releases/download/${version}/${asset}"
fi

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

if need_cmd curl; then
  curl -fsSL "$download_url" >"$tmp_file"
else
  wget -qO- "$download_url" >"$tmp_file"
fi

$SUDO install -m 0755 "$tmp_file" "$VIBEHOST_SERVER_INSTALL_DIR/$VIBEHOST_SERVER_BIN"
`
}

func bootstrapCommand(script string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return "echo " + shellQuote(encoded) + " | base64 -d | bash"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func resolveHostPort(resolved target.Resolved, agentProvider string) (int, error) {
	remoteArgs := sshcmd.RemoteArgs(resolved.App, agentProvider, []string{"port"}, nil)
	sshArgs := sshcmd.BuildArgs(resolved.Host, remoteArgs, false)
	cmd := exec.Command("ssh", sshArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return 0, fmt.Errorf("failed to resolve host port: %s", trimmed)
	}
	portText := strings.TrimSpace(string(output))
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("unexpected host port response: %q", portText)
	}
	return port, nil
}

func ensureLocalPortAvailable(port int) error {
	if port <= 0 {
		return fmt.Errorf("invalid host port %d", port)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("localhost port %d is unavailable: %v", port, err)
	}
	_ = listener.Close()
	return nil
}

func isLocalHost(host string) bool {
	normalized := strings.TrimSpace(host)
	if normalized == "" {
		return false
	}
	if at := strings.LastIndex(normalized, "@"); at >= 0 {
		normalized = normalized[at+1:]
	}
	normalized = strings.TrimSpace(normalized)
	if strings.HasPrefix(normalized, "[") {
		if end := strings.Index(normalized, "]"); end > 0 {
			normalized = normalized[1:end]
		}
	} else if colon := strings.Index(normalized, ":"); colon > 0 {
		normalized = normalized[:colon]
	}
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	switch normalized {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func xdgOpenSocketPath() string {
	return "/tmp/vibehost-open.sock"
}

func startOpenListener() (*http.Server, int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/open" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			raw := strings.TrimSpace(r.Form.Get("url"))
			cleaned, err := validateOpenURL(raw)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := openURL(cleaned); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	return server, port, nil
}

func validateOpenURL(raw string) (string, error) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return "", fmt.Errorf("missing url")
	}
	if strings.ContainsAny(cleaned, "\r\n\t") {
		return "", fmt.Errorf("invalid url")
	}
	parsed, err := url.Parse(cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid url")
	}
	if !parsed.IsAbs() || parsed.Host == "" {
		return "", fmt.Errorf("invalid url")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return cleaned, nil
	default:
		return "", fmt.Errorf("unsupported url scheme")
	}
}

func openURL(raw string) error {
	switch runtime.GOOS {
	case "darwin":
		if path, err := exec.LookPath("open"); err == nil {
			return exec.Command(path, raw).Start()
		}
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", raw).Start()
	default:
		if path, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command(path, raw).Start()
		}
		if path, err := exec.LookPath("open"); err == nil {
			return exec.Command(path, raw).Start()
		}
	}
	return fmt.Errorf("no opener available")
}
