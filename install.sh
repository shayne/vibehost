#!/bin/sh

REPO="shayne/viberun"
BASE_URL="https://github.com/${REPO}/releases"
CHANNEL="stable"
INSTALL_DIR=""
INSTALL_BIN="viberun"

usage() {
  cat <<USAGE
viberun install script

Usage:
  curl -fsSL https://viberun.sh | sh
  curl -fsSL https://viberun.sh | sh -s -- --nightly

Options:
  --nightly           Install the nightly build
  --dir <path>        Install directory (default: /usr/local/bin, /opt/homebrew/bin on macOS)
  --bin <name>        Install binary name (default: viberun)
  -h, --help          Show this help

Env:
  VIBERUN_INSTALL_DIR Install directory override
  VIBERUN_INSTALL_BIN Install binary name override
USAGE
}

fetch() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi
}

fetch_optional() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    if curl -fsSL "$url" -o "$out"; then
      return 0
    fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -qO "$out" "$url"; then
      return 0
    fi
  fi
  return 1
}

main() {
  set -eu

  while [ $# -gt 0 ]; do
    case "$1" in
      --nightly)
        CHANNEL="nightly"
        shift
        ;;
      --dir)
        if [ $# -lt 2 ]; then
          echo "--dir requires a value" >&2
          exit 1
        fi
        INSTALL_DIR="$2"
        shift 2
        ;;
      --bin)
        if [ $# -lt 2 ]; then
          echo "--bin requires a value" >&2
          exit 1
        fi
        INSTALL_BIN="$2"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "unknown option: $1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done

  if [ -n "${VIBERUN_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$VIBERUN_INSTALL_DIR"
  fi
  if [ -n "${VIBERUN_INSTALL_BIN:-}" ]; then
    INSTALL_BIN="$VIBERUN_INSTALL_BIN"
  fi

  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    linux)
      os="linux"
      ;;
    darwin)
      os="darwin"
      ;;
    *)
      echo "unsupported OS: $os" >&2
      exit 1
      ;;
  esac

  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64)
      arch="amd64"
      ;;
    arm64|aarch64)
      arch="arm64"
      ;;
    *)
      echo "unsupported arch: $arch" >&2
      exit 1
      ;;
  esac

  asset="viberun-${os}-${arch}"
  sha="${asset}.sha256"

  if [ "$CHANNEL" = "nightly" ]; then
    asset_url="${BASE_URL}/download/nightly/${asset}"
    sha_url="${BASE_URL}/download/nightly/${sha}"
  else
    asset_url="${BASE_URL}/latest/download/${asset}"
    sha_url="${BASE_URL}/latest/download/${sha}"
  fi

  if [ -z "$INSTALL_DIR" ]; then
    if [ "$os" = "darwin" ] && [ -d "/opt/homebrew/bin" ]; then
      INSTALL_DIR="/opt/homebrew/bin"
    else
      INSTALL_DIR="/usr/local/bin"
    fi
  fi

  mkdir -p "$INSTALL_DIR" 2>/dev/null || true

  SUDO=""
  if [ ! -w "$INSTALL_DIR" ]; then
    if command -v sudo >/dev/null 2>&1; then
      SUDO="sudo"
    else
      INSTALL_DIR="$HOME/.local/bin"
      mkdir -p "$INSTALL_DIR"
    fi
  fi

  tmp_dir=""
  cleanup() {
    if [ -n "${tmp_dir:-}" ]; then
      rm -rf "$tmp_dir"
    fi
  }
  trap cleanup EXIT

  tmp_dir=$(mktemp -d)

  fetch "$asset_url" "$tmp_dir/$asset"
  if fetch_optional "$sha_url" "$tmp_dir/$sha"; then
    if command -v sha256sum >/dev/null 2>&1; then
      (cd "$tmp_dir" && sha256sum -c "$sha")
    elif command -v shasum >/dev/null 2>&1; then
      (cd "$tmp_dir" && shasum -a 256 -c "$sha")
    else
      echo "sha256sum or shasum is required for checksum verification" >&2
      exit 1
    fi
  fi

  if [ ! -f "$tmp_dir/$asset" ]; then
    echo "missing downloaded binary: $asset" >&2
    exit 1
  fi

  install_target="$INSTALL_DIR/$INSTALL_BIN"
  tmp_target="$INSTALL_DIR/.${INSTALL_BIN}.tmp.$$"
  if command -v install >/dev/null 2>&1; then
    $SUDO install -m 0755 "$tmp_dir/$asset" "$tmp_target"
  else
    $SUDO cp "$tmp_dir/$asset" "$tmp_target"
    $SUDO chmod 0755 "$tmp_target"
  fi
  $SUDO mv -f "$tmp_target" "$install_target"

  if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
    echo "Installed viberun to $install_target"
    echo "Ensure $INSTALL_DIR is in your PATH."
  else
    echo "Installed viberun to $install_target"
  fi
}

main "$@"
