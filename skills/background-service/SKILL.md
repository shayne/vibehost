---
name: background-service
description: Help run a long-lived background process under systemd.
metadata:
  short-description: Systemd background service
---

# background-service

Purpose: help run a long-lived background process under systemd.

## Workflow
1) Define the command and working directory.
2) Create a systemd unit.
3) Enable and start it.
4) Check logs and restart policy.

## systemd unit template
```
[Unit]
Description=Vibehost background service (<name>)
After=network.target

[Service]
Type=simple
WorkingDirectory=/workspace/<app>
ExecStart=/usr/bin/env bash -lc '<command>'
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
```

## Common checks
- `systemctl daemon-reload`
- `systemctl enable --now <name>.service`
- `journalctl -u <name>.service -n 200 --no-pager`
