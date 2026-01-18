FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker
ENV CODEX_HOME=/root/.codex

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    dbus \
    docker.io \
    nodejs \
    npm \
    sudo \
    systemd \
    systemd-sysv \
    tmux \
    tzdata \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://starship.rs/install.sh | sh -s -- -y \
  && mkdir -p /root/.config

RUN set -eux; \
  printf '%s\n' \
    '#!/bin/sh' \
    'if [ -n "${VIBEHOST_AGENT_CHECK:-}" ]; then' \
    '  exec sh -c "$VIBEHOST_AGENT_CHECK"' \
    'fi' \
    'exec npx -y @openai/codex@latest --dangerously-bypass-approvals-and-sandbox "$@"' \
    > /usr/local/bin/codex; \
  printf '%s\n' \
    '#!/bin/sh' \
    'if [ -n "${VIBEHOST_AGENT_CHECK:-}" ]; then' \
    '  exec sh -c "$VIBEHOST_AGENT_CHECK"' \
    'fi' \
    'exec npx -y @anthropic-ai/claude-code@latest --dangerously-skip-permissions "$@"' \
    > /usr/local/bin/claude; \
  printf '%s\n' \
    '#!/bin/sh' \
    'if [ -n "${VIBEHOST_AGENT_CHECK:-}" ]; then' \
    '  exec sh -c "$VIBEHOST_AGENT_CHECK"' \
    'fi' \
    'exec npx -y @google/gemini-cli@latest --approval-mode=yolo "$@"' \
    > /usr/local/bin/gemini; \
  printf '%s\n' \
    '#!/bin/sh' \
    'set -e' \
    'url="${1:-}"' \
    'if [ -z "$url" ]; then' \
    '  echo "xdg-open: missing url" >&2' \
    '  exit 2' \
    'fi' \
    'socket="${VIBEHOST_XDG_OPEN_SOCKET:-/tmp/vibehost-open.sock}"' \
    'if [ -S "$socket" ]; then' \
    '  exec curl -sS --unix-socket "$socket" -X POST --data-urlencode "url=$url" http://localhost/open' \
    'fi' \
    'if [ -x /usr/bin/xdg-open ]; then' \
    '  exec /usr/bin/xdg-open "$url"' \
    'fi' \
    'echo "xdg-open forwarding unavailable; missing socket $socket" >&2' \
    'exit 1' \
    > /usr/local/bin/xdg-open; \
  printf '%s\n' \
    '#!/bin/sh' \
    "printf 'vibehost-agent-check ok\\\\n'" \
    > /usr/local/bin/vibehost-agent-check; \
  chmod +x /usr/local/bin/codex /usr/local/bin/claude /usr/local/bin/gemini /usr/local/bin/xdg-open /usr/local/bin/vibehost-agent-check

COPY bin/vibehost-tmux-status /usr/local/bin/vibehost-tmux-status
COPY config/tmux.conf /etc/tmux.conf
COPY config/starship.toml /root/.config/starship.toml
COPY config/bashrc-vibehost.sh /etc/profile.d/vibehost.sh
RUN chmod +x /usr/local/bin/vibehost-tmux-status \
  && cat /etc/profile.d/vibehost.sh >> /etc/bash.bashrc

RUN mkdir -p ${CODEX_HOME}/skills
COPY skills/ ${CODEX_HOME}/skills/
COPY bin/vibehost-demo /usr/local/bin/vibehost-demo
COPY bin/vibehost-container /usr/local/bin/vibehost-container
COPY systemd/vibehost-demo.service /etc/systemd/system/vibehost-demo.service
RUN systemctl enable vibehost-demo.service

VOLUME ["/sys/fs/cgroup"]
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
