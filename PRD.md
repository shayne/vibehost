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

### Out of Scope
- TLS termination, domains, or ingress configuration.
- Web UI for management and observability.

## Functional Requirements (ordered)
- [ ] R1: CLI command `vibehost <app>` opens an agent session in the app container on the default host.
  - Acceptance: The client connects via SSH to the host, executes the server program, and the user is placed into a TTY/PTY-backed agent session.
- [ ] R2: CLI supports host targeting via `vibehost <app>@<host>`.
  - Acceptance: Command routes to specified host from XDG config without manual flags.
- [ ] R3: Server daemon/CLI manages container lifecycle and port mappings per app.
  - Acceptance: First app maps container port 8080 to host 8080; subsequent apps map to 8081, 8082, etc.
- [x] R4: Containers include systemd support and can run background services.
  - Acceptance: A basic systemd service can be installed and started inside the container.
- [x] R5: Agent tooling is preinstalled and selectable (Codex default; Claude Code, Gemini alternatives).
  - Acceptance: User can choose provider at launch time or via config.
- [x] R6: Built-in skills guide agents on web services, cron jobs, and background services.
  - Acceptance: Skills are available in the agent runtime and are discoverable.
- [x] R7: Snapshot and restore are supported via CLI and in-container commands.
  - Acceptance: `vibehost <app> snapshot` creates a restorable image; `restore` can roll back state.

## UX and CLI Design
- `vibehost <app>`: enter agent session on default host.
- `vibehost <app>@<host>`: enter agent session on a named host.
- `vibehost <app> snapshot`: create snapshot of app container.
- `vibehost <app> snapshots`: list snapshots for the app.
- `vibehost <app> restore <snapshot>`: restore app state.
- `vibehost <app> shell`: open a shell without agent (optional).
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
5) User interacts in a full TTY session backed by the server (multiplexed if needed).

## Configuration
- XDG config includes default host and named hosts.
- App naming is `<app>` with optional `@<host>` suffix.

## Assumptions
- Remote VM is reachable from the client.
- Docker is installed on the VM.
- Agent tools can be installed via `npx -y ...@latest`.
- Port 8080 is the default internal port for web apps.
- The host VM is a Linux system with root access for setup.
- The repo uses `mise` for all tooling and task execution.

## Risks and Open Issues
- Port collision and mapping strategy for many apps on the same host.
- Snapshot size and performance for large app states.
- Security model for "full approvals" in agent sessions.

## Success Metrics
- Time to first agent session from fresh install (< 2 minutes).
- Successful web app launch on mapped port with no manual wiring.
- Snapshot/restore success rate > 95% in basic scenarios.

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
- Expose a server CLI entrypoint for SSH execution.

### Phase 3: Client CLI
- Implement `vibehost` client:
  - Resolves host via config (`vibehost config`).
  - SSHs to host and runs server CLI with TTY passthrough.
  - Supports `vibehost <app>` and snapshot/restore commands.

Progress Notes:
- Implemented initial client-side target parsing + config-based host/alias resolution in Go (no SSH yet).
- Added SSH invocation with TTY passthrough in the client CLI; errors clearly if SSH is missing.
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

### Phase 4: Local E2E Test (localhost SSH)
- Treat the VM as both client + server.
- Configure SSH to localhost and run `vibehost myapp`.
- Validate:
  - Missing app triggers prompt and creates container.
  - Agent TUI launches in full-screen mode.
  - Port mapping works (8080 -> host).
  - Snapshot/restore commands function.

### Phase 5: POC Completion Criteria
- From a clean VM state, `vibehost myapp` works end-to-end.
- All steps can be executed autonomously (root access).
- Tests include unit checks plus a scripted E2E run.
- Each phase has a git commit with a clear message.
- The entire POC runs inside this VM sandbox (no external CI required).

### Phase 6: CI + Release Automation (prepare, but runnable locally)
- Add GitHub Actions workflows:
  - Nightly build on `main`.
  - Release build on version tags `v*`.
  - Build and push container image to GHCR.
- Ensure every CI step has a local `mise` task equivalent so it can be run end-to-end on this VM without GitHub.

## Acceptance Test Checklists
### Client CLI
- [ ] `vibehost <app>` opens a TTY session via SSH to the server CLI.
- [ ] `vibehost <app>@<host>` targets the configured host alias.
- [ ] `vibehost config` persists default host and agent provider.
- [ ] CLI exits with a clear error if SSH is unavailable or host config is missing.

### Server Daemon/CLI
- [ ] Server CLI detects missing app and prompts to create (default yes on Enter).
- [ ] Container create/attach works for an existing app without re-provisioning.
- [ ] TTY/PTY is interactive (full-screen TUI works, arrow keys, resize).
- [ ] Multiplexing supports at least one secondary session (e.g., shell + agent).

### Container Image
- [ ] Container boots with systemd enabled.
- [ ] Agent tooling is present and runnable (Codex default).
- [x] Skills are present and discoverable in the agent runtime.
- [x] Background service can be installed and started inside the container.

### Port Mapping
- [ ] First app maps container 8080 -> host 8080.
- [ ] Second app maps container 8080 -> host 8081.
- [ ] Mapping is stable across restarts and stored in server state.

### Snapshot/Restore
- [ ] `vibehost <app> snapshot` creates a restorable snapshot.
- [ ] `vibehost <app> restore <snapshot>` reverts container state.
- [x] Snapshot list shows timestamps and app name.

### E2E (Localhost SSH)
- [ ] From clean VM state, `vibehost myapp` prompts to create app, defaults yes.
- [ ] User lands in agent TUI and can execute a command successfully.
- [ ] Web app on 8080 inside container is reachable on host mapped port.
- [ ] E2E test script runs non-interactively and exits 0.

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
