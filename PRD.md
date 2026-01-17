# Project: vibehost

## Summary
`vibehost` is a CLI-first, agent-native application host composed of three distinct parts:
1) Client CLI (`vibehost`) that users run locally.
2) Server daemon/CLI that runs on the host VM and manages sessions, containers, and PTY multiplexing.
3) Docker container image that provides the agent runtime for apps.

UX: A user runs `vibehost myapp` and is dropped into an AI agent session (default: Codex) inside an Ubuntu-based container that is preconfigured with system services, skills, and access approvals. The client connects over SSH, executes the server-side program, and the server daemon provides full TTY/PTY support, multiplexing, and container lifecycle.
Containers are persistent: the filesystem state inside each app container is preserved across sessions and is treated as a mini system.

Note on execution authority: This PRD is a living document. The LLM is expected to expand or refine it as progress is made when gaps are discovered. Not every decision is specified here; if a needed decision is not covered, the LLM has full authority to choose the best path, document it, and proceed.
Research guidance: The LLM should actively use web search to inform decisions, since it is 2026 and its knowledge cutoff is 2024; verify any potentially changed facts, tooling, or best practices with up-to-date sources.

## Problem Statement
Building and running "vibe-coded" apps with agent assistance typically requires manual environment setup, port wiring, and service management. Users want a simple, consistent command that provisions a ready-to-go agent runtime with background service support and predictable networking.

## Goals
- Provide a single CLI command to enter an agent session in a preconfigured container.
- Make the three-part architecture explicit (client CLI, server daemon, container image).
- Persist container filesystem state across sessions (app containers are long-lived).
- Implement all primary components in Go; bash is allowed for scripts and glue where needed.
- Support multiple agent providers (Codex default; Claude Code and Gemini as alternatives).
- Enable background services (systemd), web apps, and cron jobs without extra setup.
- Make snapshot/restore of app state easy from both the host CLI and inside the container.
- Support multiple remote hosts via simple `app@host` addressing.

## Non-goals
- No host-based TLS termination in this phase; rely on VM host + port mapping.
- No web UI for management.

## Users and Use Cases
### Primary Users
- Developers who want a quick, repeatable agent environment to build apps.
- Tinkerers building multiple small services with minimal setup.

### Core Use Cases
- Start an app: `vibehost myapp` opens an agent session in a container.
- Target another host: `vibehost myapp@hostb`.
- Run a web app on container port 8080; host daemon maps to 8080, 8081, etc.
- Configure background services or cron tasks via built-in skills.
- Snapshot state: `vibehost myapp snapshot`.
- Restore state: `vibehost myapp restore <snapshot>`.
- Trigger snapshot/restore from inside the container via internal commands.

## Scope
### In Scope
- Client CLI (`vibehost`) with host selection and app targeting.
- Server daemon/CLI on VM to manage persistent container lifecycle, TTY/PTY sessions, and multiplexing.
- Ubuntu-based container image with systemd support and agent tools.
- Preinstalled agent tooling via `npx -y ...@latest` for Codex, Claude Code, Gemini.
- Built-in skills for service setup (web, cron, background services).
- Port mapping strategy: container 8080 -> host 8080 (first app), host 8081, 8082 for additional apps.
- Snapshot/restore via Docker commit (or equivalent) triggered from host or container.
- Client-side installation script (curl | bash) to install the `vibehost` binary.
- Host bootstrap flow over SSH to prepare a new Ubuntu host for `vibehost`.
- Client-side reverse proxy/tunnel so localhost access works during an active session.
- Forwarding `xdg-open` calls from container to the client machine.
- Project documentation: README (overview + hello-world example) and DEVELOPMENT (setup + workflow).

### Out of Scope
- TLS termination, domains, or ingress configuration.
- Web UI for management and observability.

## Functional Requirements (ordered)
- [x] R1: CLI command `vibehost <app>` opens an agent session in the app container on the default host.
  - Acceptance: The client connects via SSH to the host, executes the server program, and the user is placed into a TTY/PTY-backed agent session.
- [x] R2: CLI supports host targeting via `vibehost <app>@<host>`.
  - Acceptance: Command routes to specified host from XDG config without manual flags.
- [x] R3: Server daemon/CLI manages container lifecycle and port mappings per app.
  - Acceptance: First app maps container port 8080 to host 8080; subsequent apps map to 8081, 8082, etc.
- [x] R4: Containers include systemd support and can run background services.
  - Acceptance: A basic systemd service can be installed and started inside the container.
- [x] R5: Agent tooling is preinstalled and selectable (Codex default; Claude Code, Gemini alternatives).
  - Acceptance: User can choose provider at launch time or via config.
- [x] R6: Built-in skills guide agents on web services, cron jobs, and background services.
  - Acceptance: Skills are available in the agent runtime and are discoverable.
- [x] R7: Snapshot and restore are supported via CLI and in-container commands.
  - Acceptance: `vibehost <app> snapshot` creates a restorable image; `restore` can roll back state.
- [x] R8: Provide a client install script that supports `curl | bash` to install `vibehost`.
  - Acceptance: A new user can install the client in one command on a fresh machine and run `vibehost --version` successfully.
- [x] R9: Provide a host bootstrap command that connects via SSH and prepares an Ubuntu host.
  - Acceptance: `vibehost bootstrap <host>` installs Docker, configures the server daemon, and validates required dependencies; fails fast on non-Ubuntu.
  - Acceptance: If the SSH user is non-root, warn about sudo usage and ensure Docker group membership; prompt to `newgrp docker` or reconnect as needed.
- [x] R10: While `vibehost <app>` is running, the client provides a localhost reverse proxy to the app's host port mapping.
  - Acceptance: Visiting `http://localhost:<host-port>` on the client reaches the app running in the container without manual tunnel setup.
- [x] R11: The container knows its internal and externally reachable ports via environment variables.
  - Acceptance: `VIBEHOST_APP_PORT=8080` and `VIBEHOST_HOST_PORT=<host-port>` are available inside the container.
- [ ] R12: `xdg-open` calls inside the container are forwarded to the client machine.
  - Acceptance: A call to `xdg-open http://localhost:<host-port>` inside the container opens the user’s local browser.
- [ ] R13: Provide project documentation for users and contributors.
  - Acceptance: `README.md` provides a complete start-to-finish flow from zero setup to:
    - installing the client
    - bootstrapping a host
    - starting `vibehost myapp`
    - using Codex to create a hello-world HTTP server
    - accessing the app in a local browser via localhost proxy
  - Acceptance: `DEVELOPMENT.md` explains local setup, build/test workflow, and how to run integration/E2E scripts.
- [ ] R14: The repo will exist at https://github.com/shayne/vibehost so all references to the GitHub project, install URLs, docker container registry names, go.mod, etc. should all be based on this location
  - Acceptance: there are no placeholders in the project for any repo or owner and the go module uses the github path and all imports use the github module name

## UX and CLI Design
- `vibehost <app>`: enter agent session on default host.
- `vibehost <app>@<host>`: enter agent session on a named host.
- `vibehost <app> snapshot`: create snapshot of app container.
- `vibehost <app> snapshots`: list snapshots for the app.
- `vibehost <app> restore <snapshot>`: restore app state.
- `vibehost <app> shell`: open a shell without agent (optional).
- `vibehost bootstrap [<host>]`: prepare a remote Ubuntu host over SSH (default host if omitted).
- In-container: `vibehost-container snapshot|snapshots|restore <snapshot>`.
- `vibehost config`: set defaults (host, agent provider, etc.).
- `vibehost --agent <provider> <app>`: override agent provider for this session.

## System Architecture
- Client: `vibehost` CLI on user machine.
- Server: VM running `vibehost` daemon/CLI with SSH entrypoint.
- Container: Ubuntu base with systemd, agent tooling, and skills.
- Flow: client -> SSH -> server CLI -> daemon -> Docker -> app container.

## Key Interaction Flow
1) User runs `vibehost myapp`.
2) Client SSHs to host and runs the server CLI (TTY/PTY required).
3) Server checks for app container; if missing, prompt to create (default yes).
4) Server starts/attaches container, launches configured agent TUI.
5) Client starts a localhost reverse proxy for the app's host port.
6) User interacts in a full TTY session backed by the server (multiplexed if needed).
7) If the container triggers `xdg-open`, it is forwarded to the client.

## Configuration
- XDG config includes default host and named hosts.
- App naming is `<app>` with optional `@<host>` suffix.
- Client install script target (binary name, install dir) is configurable via env vars.

## Assumptions
- Remote VM is reachable from the client.
- Docker is installed on the VM.
- Agent tools can be installed via `npx -y ...@latest`.
- Port 8080 is the default internal port for web apps.
- The host VM is a Linux system with root access for setup.
- Bootstrap targets Ubuntu and exits on non-Ubuntu hosts.
- The repo uses `mise` for all tooling and task execution.

## Risks and Open Issues
- Port collision and mapping strategy for many apps on the same host.
- Snapshot size and performance for large app states.
- Security model for "full approvals" in agent sessions.
- Localhost reverse proxy conflicts with existing client ports.
- Host bootstrap reliability across Ubuntu versions and minimal images.

## Success Metrics
- Time to first agent session from fresh install (< 2 minutes).
- Successful web app launch on mapped port with no manual wiring.
- Snapshot/restore success rate > 95% in basic scenarios.
- Client install success rate > 95% using curl | bash on a fresh machine.
- Host bootstrap completes in < 5 minutes on a fresh Ubuntu VM.
- Localhost reverse proxy success rate > 95% for active sessions.
- `xdg-open` forwarding success rate > 90% for URLs.
- Documentation completeness: README + DEVELOPMENT present and up-to-date.

## Milestones (initial)
- M1: CLI + daemon + container bring-up with agent session.
- M2: Port mapping and multi-host selection.
- M3: Snapshot/restore and in-container control commands.
- M4: Skills for web/cron/background services.

## End-to-End POC Plan (start to finish)
### Phase 0: Repo + Tooling Setup
- Ensure `mise` is installed and configured for the project.
- Use `mise` for all tools/tasks (build, lint, test, etc.).
- Work in a git repo; commit at logical checkpoints.
- Host provisioning: use web search for current Ubuntu instructions and install system dependencies with `apt` and/or `curl | bash` (or equivalent) as required to run the program, containers, and E2E tests on this host.

### Phase 1: Container Image
- Define Dockerfile for Ubuntu + systemd + agent tooling.
- Bake in skills and default agent configuration.
- Build the image via `mise` task.
- Validate systemd works inside the container.

### Phase 2: Server Daemon/CLI
- Implement server program that:
  - Manages containers per app name.
  - Allocates ports and stores mappings.
  - Provides TTY/PTY session handling and multiplexing.
  - Prompts to create app container if missing (default yes).
  - Emits host port mapping to the client for proxy setup.
- Expose a server CLI entrypoint for SSH execution.

### Phase 3: Client CLI
- Implement `vibehost` client:
  - Resolves host via config (`vibehost config`).
  - SSHs to host and runs server CLI with TTY passthrough.
  - Supports `vibehost <app>` and snapshot/restore commands.
  - Starts/stops a localhost reverse proxy tied to the session lifecycle.
  - Forwards `xdg-open` requests from the container to the client.
  - Provides `vibehost bootstrap` for host setup over SSH.

### Phase 3.5: Installation + Bootstrap
- Publish a client install script that supports `curl | bash`.
- Implement `vibehost bootstrap` to prepare an Ubuntu host.
- Bootstrap flow must:
  - Detect Ubuntu and exit with a clear error if not.
  - Install Docker and any server daemon dependencies.
  - Ensure the SSH user can run Docker (sudo + docker group).
  - Prompt when a reconnect/newgrp is required.

Progress Notes:
- Implemented initial client-side target parsing + config-based host/alias resolution in Go (no SSH yet).
- Added SSH invocation with TTY passthrough in the client CLI; errors clearly if SSH is missing.
- Made SSH TTY allocation conditional for interactive sessions and added ssh argument tests.
- Added initial `vibehost-server` CLI that provisions/starts Docker containers, persists port allocations, and attaches an interactive shell.
- Implemented `vibehost config` command to view/update config with default host, agent provider, and host alias mappings.
- Wired agent provider selection through the client config into `vibehost-server`, which now launches provider commands (`codex`, `claude`, `gemini`) inside the container.
- Added client/server snapshot and restore commands that create Docker snapshot images and restore from named or `latest` snapshots.
- Added a base container Dockerfile (Ubuntu + systemd + Node.js) and agent CLI wrappers that invoke `npx` for Codex, Claude Code, and Gemini.
- Updated server container run/restore flow to boot systemd (`/sbin/init`) with cgroup/tmpfs mounts so background services can run.
- Added built-in skills for web services, cron jobs, and background services to the container image.
- Added `vibehost <app> snapshots` to list available snapshots with app name and timestamp tags.
- Baked a demo systemd unit into the container image to validate background service enablement.
- Added `--agent` override to `vibehost` so users can select a provider per launch without changing config.
- Added `vibehost <app> shell` to open a non-agent shell session inside the container.
- Added `vibehost-container` in-container commands for snapshot/restore/list and passed container metadata/env plus Docker socket into the container for self-management.
- Added `vibehost config --host` alias for `--default-host` to match PRD/E2E usage.
- Guarded port mapping for existing containers when server state is missing and added parsing coverage for Docker port output.
- Detect TTY availability for server `docker exec` and avoid `-t` when stdin/stdout are non-interactive; added coverage for exec arg construction.
- Synced server port state from existing vibehost containers on startup so port assignments survive missing state files.
- Added server prompt tests to confirm default-yes behavior for app creation prompts.
- Added a client-side guard to require a TTY for interactive sessions so agent runs fail fast in non-terminal environments.
- Added tmux-based session wrapping for agent and shell commands to support multiple concurrent sessions.
- Forwarded terminal environment variables into interactive docker exec sessions to ensure full-screen TUIs and arrow keys behave correctly.
- Extended the local E2E script to verify systemd is PID 1 and agent CLI wrappers are present in the container.
- Extended the local E2E script to snapshot a test file, restore, and assert the file reverted.
- Added a VIBEHOST_AGENT_CHECK hook plus E2E verification that the agent command runs in a TTY-backed session.
- Added a nightly GitHub Actions workflow that runs build/test/vet via `mise` tasks.
- Added a release GitHub Actions workflow that builds Linux artifacts on version tags via `mise run release:build`.
- Added GHCR image build/push automation plus matching `mise` tasks for local parity.
- Added a host-run integration test script plus a `mise run integration` task to validate server/container wiring (requires Docker on the host).
- Fixed Dockerfile agent wrapper generation to avoid Dockerfile parsing errors and ensured systemd containers stay running with `--cgroupns=host`.
- Ran host integration tests on a Docker-capable host and marked the PRD complete.
- Added a curl | bash client install script with configurable install dir/binary name and GitHub release downloads.
- Added `vibehost bootstrap` to validate Ubuntu over SSH, install Docker, install `vibehost-server`, and warn about docker group membership for non-root users.
- Implemented client-side localhost reverse proxy via SSH local port forwarding, added server port query action, and detect local port conflicts for interactive sessions.
- Passed `VIBEHOST_APP_PORT` and `VIBEHOST_HOST_PORT` into containers on create/restore so apps can read internal and host ports from env.

### Phase 4: Local E2E Test (localhost SSH)
- Treat the VM as both client + server.
- Configure SSH to localhost and run `vibehost myapp`.
- Validate:
  - Missing app triggers prompt and creates container.
  - Agent TUI launches in full-screen mode.
  - Port mapping works (8080 -> host).
  - Snapshot/restore commands function.
 - Scripted smoke test: `bin/vibehost-e2e-local` builds binaries, installs them locally, runs a non-interactive shell session, validates port mapping, and exercises snapshot/restore.

### Phase 5: POC Completion Criteria
- From a clean VM state, `vibehost myapp` works end-to-end.
- All steps can be executed autonomously (root access).
- Tests include unit checks plus a scripted E2E run.
- Integration tests are created or run independently; they must be executed on the host and pass before the POC is considered complete.
- Each phase has a git commit with a clear message.
- The entire POC runs inside this VM sandbox (no external CI required).

### Phase 6: CI + Release Automation (prepare, but runnable locally)
- Add GitHub Actions workflows:
  - [x] Nightly build on `main`.
  - [x] Release build on version tags `v*`.
  - [x] Build and push container image to GHCR.
- [x] Ensure every CI step has a local `mise` task equivalent so it can be run end-to-end on this VM without GitHub.

## Acceptance Test Checklists
### Client CLI
- [x] `vibehost <app>` opens a TTY session via SSH to the server CLI.
- [x] `vibehost <app>@<host>` targets the configured host alias.
- [x] `vibehost config` persists default host and agent provider.
- [x] CLI exits with a clear error if SSH is unavailable or host config is missing.

### Server Daemon/CLI
- [x] Server CLI detects missing app and prompts to create (default yes on Enter).
- [x] Container create/attach works for an existing app without re-provisioning.
- [x] TTY/PTY is interactive (full-screen TUI works, arrow keys, resize).
- [x] Multiplexing supports at least one secondary session (e.g., shell + agent).

### Container Image
- [x] Container boots with systemd enabled.
- [x] Agent tooling is present and runnable (Codex default).
- [x] Skills are present and discoverable in the agent runtime.
- [x] Background service can be installed and started inside the container.

### Port Mapping
- [x] First app maps container 8080 -> host 8080.
- [x] Second app maps container 8080 -> host 8081.
- [x] Mapping is stable across restarts and stored in server state.
- [x] Container exports `VIBEHOST_APP_PORT` and `VIBEHOST_HOST_PORT` for app use.

### Localhost Reverse Proxy
- [x] When `vibehost <app>` is active, the client exposes `http://localhost:<host-port>` for the app.
- [x] Proxy shuts down cleanly when the session exits.
- [x] Port conflicts on the client are detected and reported.

### xdg-open Forwarding
- [ ] `xdg-open` inside the container triggers a client-side open.
- [ ] URLs are validated/sanitized before forwarding.

### Install + Bootstrap
- [x] `curl | bash` installs the client binary and prints next steps.
- [x] `vibehost bootstrap <host>` validates Ubuntu, installs Docker, configures server daemon.
- [x] Non-root bootstrap warns about sudo usage and docker group membership.

### Documentation
- [ ] `README.md` provides a short overview and hello-world prompt example.
- [ ] `DEVELOPMENT.md` describes local setup and the dev/test workflow.

### Snapshot/Restore
- [x] `vibehost <app> snapshot` creates a restorable snapshot.
- [x] `vibehost <app> restore <snapshot>` reverts container state.
- [x] Snapshot list shows timestamps and app name.

### E2E (Localhost SSH)
- [x] From clean VM state, `vibehost myapp` prompts to create app, defaults yes.
- [x] User lands in agent TUI and can execute a command successfully.
- [x] Web app on 8080 inside container is reachable on host mapped port.
- [x] E2E test script runs non-interactively and exits 0.

### Integration Tests
- [x] Integration tests exist (or are run independently) and are executed on the host.
- [x] POC is not marked complete until integration tests run and pass.

## Architecture Diagram (text)
```
Client Machine
  └─ vibehost (CLI)
       └─ SSH (TTY/PTY passthrough)
            └─ Server Host (VM)
                 ├─ vibehost server CLI (entrypoint)
                 ├─ vibehost daemon (container manager + mux)
                 └─ Docker
                      └─ App Container (Ubuntu + systemd + agent + skills)
```

## Architecture Diagram (Mermaid)
```mermaid
flowchart LR
  U[User] --> C[vibehost CLI]
  C -->|SSH (TTY/PTY)| S[Server CLI/Daemon]
  S --> D[(Docker)]
  D --> A[App Container<br/>Ubuntu + systemd + agent + skills]
```

## Sequence Diagram (text)
```
User          Client CLI         SSH             Server CLI/Daemon        Docker            Container
 |                |               |                      |                   |                   |
 | vibehost myapp |               |                      |                   |                   |
 |--------------->|               |                      |                   |                   |
 |                |----ssh cmd--->|--------------------->|                   |                   |
 |                |               |                      | check app exists  |                   |
 |                |               |                      |----inspect------->|                   |
 |                |               |                      |<---not found------|                   |
 |                |               |                      | prompt create     |                   |
 |                |               |<-----prompt----------|                   |                   |
 |   Enter        |               |                      |                   |                   |
 |--------------->|               |                      |                   |                   |
 |                |               |                      | create container  |----run----------->|
 |                |               |                      | attach TTY/PTY    |------------------->|
 |                |<=== TTY/PTY streaming over SSH ===========================|                   |
 |<======================== Agent TUI ========================================|                   |
```

## Sequence Diagram (Mermaid)
```mermaid
sequenceDiagram
  participant U as User
  participant C as Client CLI
  participant H as SSH
  participant S as Server CLI/Daemon
  participant D as Docker
  participant A as App Container

  U->>C: vibehost myapp
  C->>H: ssh host "vibehost-server myapp"
  H->>S: start server CLI (TTY/PTY)
  S->>D: inspect app container
  D-->>S: not found
  S-->>U: prompt create? (default yes)
  U->>C: Enter
  C->>H: forward input
  H->>S: confirm create
  S->>D: run container
  D->>A: start
  S<->>A: attach TTY/PTY
  H<->>S: stream TTY/PTY
  C<->>H: stream TTY/PTY
  U<->>C: agent TUI session
```

## E2E Test Script Outline (for LLM + automation)
### Assumptions
- VM is both client + server host (localhost SSH).
- `mise` is installed and configured; all tasks run through `mise`.
- Repo is clean; commits are allowed.

### Script Steps (pseudo-commands)
```bash
# 0) Setup tools and config
mise install
mise run setup
vibehost config --host localhost --agent codex

# 1) Build container image
mise run build:image

# 2) Start/ensure server daemon
mise run server:install
mise run server:start

# 3) First-run flow: app doesn't exist
vibehost myapp <<'EOF'

EOF

# 4) Validate agent session and TTY
# (script asserts prompt text + terminal mode switch)

# 5) Validate port mapping
curl -f http://localhost:8080/health

# 6) Snapshot + restore
vibehost myapp snapshot
vibehost myapp restore latest

# 7) Idempotent re-run
vibehost myapp <<'EOF'

EOF
```

### Script Assertions
- Server prompts to create app if missing; default yes on Enter.
- Agent TUI launches and accepts input.
- Port 8080 responds from host.
- Snapshot/restore completes with non-error exit codes.
- Second `vibehost myapp` attaches without re-provisioning.

## Open Questions
- Q1: How should users select an agent provider (flag, config, or interactive prompt)?
- Q2: What is the snapshot retention policy and naming scheme?
- Q3: Do we need app-level resource limits (CPU/mem) in the first release?
