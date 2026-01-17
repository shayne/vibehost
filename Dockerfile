FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker
ENV CODEX_HOME=/root/.codex

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    dbus \
    nodejs \
    npm \
    sudo \
    systemd \
    systemd-sysv \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
  cat > /usr/local/bin/codex <<'SCRIPT'\
#!/bin/sh\
exec npx -y @openai/codex@latest "$@"\
SCRIPT\
  ; \
  cat > /usr/local/bin/claude <<'SCRIPT'\
#!/bin/sh\
exec npx -y @anthropic-ai/claude-code@latest "$@"\
SCRIPT\
  ; \
  cat > /usr/local/bin/gemini <<'SCRIPT'\
#!/bin/sh\
exec npx -y @google/gemini-cli@latest "$@"\
SCRIPT\
  ; \
  chmod +x /usr/local/bin/codex /usr/local/bin/claude /usr/local/bin/gemini

RUN mkdir -p ${CODEX_HOME}/skills
COPY skills/ ${CODEX_HOME}/skills/

VOLUME ["/sys/fs/cgroup"]
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
