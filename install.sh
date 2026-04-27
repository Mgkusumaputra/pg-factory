#!/usr/bin/env bash
# install.sh — end-user installer for pg-factory (macOS / Linux / WSL)
# Usage:  curl -fsSL https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.sh | bash
set -euo pipefail

REPO_URL="https://github.com/Mgkusumaputra/pg-factory.git"
REPO_GO="github.com/Mgkusumaputra/pg-factory"
MIN_GO="1.25"

# ── helpers ──────────────────────────────────────────────────────────────────
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

# ── check Go ─────────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
  error "Go is not installed. Install Go $MIN_GO+ from https://go.dev/dl/ then re-run."
fi

GO_VERSION_RAW="$(go env GOVERSION | sed 's/^go//')"
GO_VERSION="$(printf '%s' "$GO_VERSION_RAW" | sed -E 's/^([0-9]+\.[0-9]+).*/\1/')"
if [[ ! "$GO_VERSION" =~ ^[0-9]+\.[0-9]+$ ]]; then
  error "Could not parse Go version from '$GO_VERSION_RAW'. Install Go $MIN_GO+ from https://go.dev/dl/."
fi
if ! version_ge "$GO_VERSION" "$MIN_GO"; then
  error "Go $MIN_GO+ is required. Found Go $GO_VERSION_RAW."
fi
info "Found Go $GO_VERSION_RAW"

# ── check Docker (warning only — needed at runtime, not install time) ─────────
if ! command -v docker &>/dev/null; then
  warn "Docker not found. pg-factory needs Docker at runtime."
  warn "Install: https://docs.docker.com/get-docker/"
fi

# ── resolve GOBIN (respects $GOBIN override) ──────────────────────────────────
GOBIN_DIR="$(go env GOBIN)"
if [[ -z "$GOBIN_DIR" ]]; then
  GOBIN_DIR="$(go env GOPATH)/bin"
fi
mkdir -p "$GOBIN_DIR"
TARGET="$GOBIN_DIR/pg"

# ── build ─────────────────────────────────────────────────────────────────────
# Strategy: build from local source if we're in the repo root (dev workflow),
# otherwise clone from GitHub and build — avoids the go install binary-rename
# problem where go install always names the binary after the module path
# ("pg-factory") not the intended command name ("pg").

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -f "$SCRIPT_DIR/main.go" && -f "$SCRIPT_DIR/go.mod" ]]; then
  # ── local / dev path ────────────────────────────────────────────────────────
  info "Building from local source…"
  (cd "$SCRIPT_DIR" && go build -ldflags="-s -w" -o "$TARGET" .)
elif command -v git &>/dev/null; then
  # ── remote / end-user path (git available) ──────────────────────────────────
  info "Cloning and building pg-factory…"
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "$TMP_DIR"' EXIT
  git clone --depth 1 "$REPO_URL" "$TMP_DIR" &>/dev/null
  (cd "$TMP_DIR" && go build -ldflags="-s -w" -o "$TARGET" .)
else
  # ── fallback: go install + rename (git not available) ────────────────────────
  info "git not found, falling back to go install…"
  go install "${REPO_GO}@latest"
  BUILT="$GOBIN_DIR/pg-factory"
  if [[ -f "$BUILT" ]]; then
    mv "$BUILT" "$TARGET"
  fi
fi

if [[ ! -f "$TARGET" ]]; then
  error "Build failed: pg binary not found at $TARGET"
fi

# ── ensure GOBIN is on PATH ───────────────────────────────────────────────────
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$GOBIN_DIR"; then
  if [ -n "${ZSH_VERSION:-}" ]; then
    SHELL_RC="$HOME/.zshrc"
  elif [ -n "${BASH_VERSION:-}" ]; then
    SHELL_RC="$HOME/.bashrc"
  else
    SHELL_RC="$HOME/.profile"
  fi
  echo "" >> "$SHELL_RC"
  echo "# pg-factory" >> "$SHELL_RC"
  echo "export PATH=\"\$PATH:$GOBIN_DIR\"" >> "$SHELL_RC"
  export PATH="$PATH:$GOBIN_DIR"
  warn "Added $GOBIN_DIR to PATH in $SHELL_RC"
  warn "Restart your shell or run: source $SHELL_RC"
fi

success "pg installed → $TARGET"
echo ""

# ── first-time setup wizard ───────────────────────────────────────────────────
PGFACTORY_CONFIG="$HOME/.pgfactory/config.json"
if [[ ! -f "$PGFACTORY_CONFIG" ]]; then
  info "Launching first-time setup…"
  echo ""
  "$TARGET" init
else
  info "Config already exists. Run 'pg init' to reconfigure."
fi
