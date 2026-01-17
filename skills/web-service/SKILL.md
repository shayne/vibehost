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
2) Create a systemd unit at `/etc/systemd/system/<app>.service` using port 8080 by default.
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

[Install]
WantedBy=multi-user.target
```

## Common checks
- `systemctl daemon-reload`
- `systemctl enable --now <app>.service`
- `journalctl -u <app>.service -n 200 --no-pager`
