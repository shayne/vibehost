# vibehost

`vibehost` is a CLI-first, agent-native app host. You run `vibehost <app>` locally and get dropped into an agent session inside a persistent Ubuntu container on a remote host (default agent: Codex). Containers keep their filesystem state between sessions and can run systemd services.

## Quick start (end-to-end)

### 1) Install the client

```bash
curl -fsSL https://raw.githubusercontent.com/shayne/vibehost/main/bin/vibehost-install | \
  bash
```

Verify:

```bash
vibehost --version
```

Optional overrides:

```bash
VIBEHOST_REPO=OWNER/REPO VIBEHOST_VERSION=v0.1.0 bash
```

### 2) Bootstrap a host (once per VM)

Ensure you can SSH into the host (for example, `myhost` in `~/.ssh/config`). Then:

```bash
vibehost bootstrap myhost
```

Optional: set it as your default host (and default agent) so you can omit `@host` later:

```bash
vibehost config --host myhost --agent codex
```

### 3) Start an app session

```bash
vibehost myapp
```

If this is the first run, the server will prompt to create the container. Press Enter to accept.

### 4) Use Codex to build a hello-world app

In the agent session, use a prompt like:

```
Create a beautiful hello-world web app with a simple, tasteful landing page. Keep it running as a service so I can open it from my laptop.
```

By convention, apps listen on container port 8080. Make sure your server binds to `0.0.0.0` (not `127.0.0.1`) so the host port mapping works.

### 5) Open the app in your local browser

While the session is active, `vibehost` starts a localhost proxy to the host port. For the first app on a host, the mapping is:

- container `8080` -> host `8080` -> local `http://localhost:8080`

So you can open:

```
http://localhost:8080
```

If you run multiple apps on the same host, the next app maps to 8081, then 8082, etc.
`vibehost` prints the local proxy URL when it starts the session.

## How it works

- Client: `vibehost` CLI on your machine.
- Server: `vibehost-server` on the host VM (runs via SSH).
- Container: Ubuntu + systemd + agent tooling + built-in skills.

Flow: `vibehost myapp` -> SSH -> server CLI -> Docker container -> agent session.

## Common commands

```bash
vibehost myapp
vibehost myapp@hostb
vibehost myapp snapshot
vibehost myapp snapshots
vibehost myapp restore latest
vibehost myapp shell
vibehost bootstrap [<host>]
vibehost config --host myhost --agent codex
```

## Development

See DEVELOPMENT.md for local setup, build/test workflow, and E2E/integration scripts.
