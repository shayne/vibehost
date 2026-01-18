---
name: web-service
description: Help run a web service under systemd, including ports and health checks.
metadata:
  short-description: Web service
---

# web-service

Purpose: help set up a web app inside the container with a stable systemd service and port 8080 mapping.

## Workflow
1) Identify the app command and working directory.
2) If the user asks for a simple web app without specifics, pick a lightweight server and build a small, tasteful single-page HTML UI.
3) Ensure the web server binds to `0.0.0.0` (not `127.0.0.1`) so host port mapping works.
3) Create a systemd unit at `/etc/systemd/system/<app>.service` using port 8080 by default.
3) Reload, enable, and start the service.
4) Verify with `systemctl status` and `journalctl -u`.

## systemd unit template
```
[Unit]
Description=Vibehost web service (<app>)
After=network.target

[Service]
Type=simple
WorkingDirectory=/workspace/<app>
ExecStart=/usr/bin/env bash -lc '<start command>'
Restart=on-failure
Environment=PORT=8080
Environment=HOST=0.0.0.0

[Install]
WantedBy=multi-user.target
```

## Common checks
- `systemctl daemon-reload`
- `systemctl enable --now <app>.service`
- `journalctl -u <app>.service -n 200 --no-pager`

## User-facing notes
- Treat the host port as the only user-facing port; do not mention 8080 unless the user explicitly asks.
- Always include the concrete local URL derived from the environment.
- Use `printenv VIBEHOST_HOST_PORT` (or `echo "$VIBEHOST_HOST_PORT"`) to read it, then say:
  - `Open http://localhost:<port> in your laptop browser while this session is active.`

## Default hello-world behavior
- Prefer a minimal single-file app with an attractive HTML + CSS landing page.
- Keep dependencies light (Python stdlib or Node + minimal script).
- Use port 8080 and bind `0.0.0.0` automatically; do not ask the user to specify these or mention them unless asked.
