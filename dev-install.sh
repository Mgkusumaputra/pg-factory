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
MIN_GO="1.25"

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

version_ge() {
  local have="$1"
  local need="$2"
  local have_major have_minor need_major need_minor
  IFS='.' read -r have_major have_minor _ <<< "$have"
  IFS='.' read -r need_major need_minor _ <<< "$need"
  if (( have_major > need_major )); then return 0; fi
  if (( have_major < need_major )); then return 1; fi
  (( have_minor >= need_minor ))
}

# ── check Go ──────────────────────────────────────────────────────────────────
command -v go &>/dev/null || error "Go $MIN_GO+ is required. Install: https://go.dev/dl/"
GO_VERSION_RAW="$(go env GOVERSION | sed 's/^go//')"
GO_VERSION="$(printf '%s' "$GO_VERSION_RAW" | sed -E 's/^([0-9]+\.[0-9]+).*/\1/')"
if [[ ! "$GO_VERSION" =~ ^[0-9]+\.[0-9]+$ ]]; then
  error "Could not parse Go version from '$GO_VERSION_RAW'. Install Go $MIN_GO+."
fi
if ! version_ge "$GO_VERSION" "$MIN_GO"; then
  error "Go $MIN_GO+ is required. Found Go $GO_VERSION_RAW."
fi
info "Found Go $GO_VERSION_RAW"

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
PGFACTORY_CONFIG="$HOME/.pgfactory/config.json"
if [[ ! -f "$PGFACTORY_CONFIG" ]]; then
  info "Launching first-time setup…"
  echo ""
  "$BIN_DIR/pg" init
else
  info "Config already exists ($PGFACTORY_CONFIG). Run 'pg init' to reconfigure."
fi
