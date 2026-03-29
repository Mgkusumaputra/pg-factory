#!/usr/bin/env bash
# dev-install.sh — contributor installer for pg-factory (macOS / Linux / WSL)
#
# Clones (or updates) the repo, builds from source, links the binary to
# a directory on PATH, then launches `pg init` if not already configured.
#
# Usage:
#   bash dev-install.sh [--dir <install-dir>] [--bin-dir <bin-dir>]
#
# Defaults:
#   --dir      $HOME/src/pg-factory
#   --bin-dir  $HOME/.local/bin   (falls back to /usr/local/bin if writable)
set -euo pipefail

REPO_URL="https://github.com/Mgkusumaputra/pg-factory.git"
INSTALL_DIR="${PGFACTORY_SRC:-$HOME/src/pg-factory}"
BIN_DIR="${PGFACTORY_BIN:-}"

# ── parse args ────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)     INSTALL_DIR="$2"; shift 2 ;;
    --bin-dir) BIN_DIR="$2";     shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

# ── choose bin dir ────────────────────────────────────────────────────────────
if [[ -z "$BIN_DIR" ]]; then
  if [[ -w /usr/local/bin ]]; then
    BIN_DIR="/usr/local/bin"
  else
    BIN_DIR="$HOME/.local/bin"
    mkdir -p "$BIN_DIR"
  fi
fi

info()    { printf '\033[0;36m  ▸ %s\033[0m\n' "$*"; }
success() { printf '\033[0;32m  ✓ %s\033[0m\n' "$*"; }
warn()    { printf '\033[0;33m  ⚠ %s\033[0m\n' "$*"; }
error()   { printf '\033[0;31m  ✗ %s\033[0m\n' "$*" >&2; exit 1; }

# ── check Go ──────────────────────────────────────────────────────────────────
command -v go &>/dev/null || error "Go 1.21+ is required. Install: https://go.dev/dl/"
info "Found Go $(go env GOVERSION | sed 's/go//')"

# ── check Docker ──────────────────────────────────────────────────────────────
command -v docker &>/dev/null || warn "Docker not found — needed at runtime. https://docs.docker.com/get-docker/"

# ── clone or update ───────────────────────────────────────────────────────────
if [[ -d "$INSTALL_DIR/.git" ]]; then
  info "Updating repo at $INSTALL_DIR…"
  git -C "$INSTALL_DIR" pull --ff-only
else
  info "Cloning pg-factory into $INSTALL_DIR…"
  git clone "$REPO_URL" "$INSTALL_DIR"
fi

# ── build ─────────────────────────────────────────────────────────────────────
info "Building from source…"
(cd "$INSTALL_DIR" && go build -ldflags="-s -w" -o "$BIN_DIR/pg" .)

# ── ensure BIN_DIR is on PATH ─────────────────────────────────────────────────
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$BIN_DIR"; then
  SHELL_RC="$HOME/.bashrc"
  [ -n "${ZSH_VERSION:-}" ] && SHELL_RC="$HOME/.zshrc"
  echo "" >> "$SHELL_RC"
  echo "# pg-factory dev" >> "$SHELL_RC"
  echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$SHELL_RC"
  export PATH="$PATH:$BIN_DIR"
  warn "Added $BIN_DIR to PATH in $SHELL_RC — restart your shell or: source $SHELL_RC"
fi

success "pg built → $BIN_DIR/pg"
info "Source at: $INSTALL_DIR"
echo ""

# ── first-time setup ──────────────────────────────────────────────────────────
PGFACTORY_CONFIG="${XDG_CONFIG_HOME:-$HOME}/.pgfactory/config.json"
if [[ ! -f "$PGFACTORY_CONFIG" ]]; then
  info "Launching first-time setup…"
  echo ""
  "$BIN_DIR/pg" init
else
  info "Config already exists ($PGFACTORY_CONFIG). Run 'pg init' to reconfigure."
fi
