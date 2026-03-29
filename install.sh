#!/usr/bin/env bash
# install.sh — end-user installer for pg-factory (macOS / Linux / WSL)
# Usage:  curl -fsSL https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.sh | bash
set -euo pipefail

REPO="github.com/Mgkusumaputra/pg-factory"
MIN_GO="1.21"

# ── helpers ──────────────────────────────────────────────────────────────────
info()    { printf '\033[0;36m  ▸ %s\033[0m\n' "$*"; }
success() { printf '\033[0;32m  ✓ %s\033[0m\n' "$*"; }
warn()    { printf '\033[0;33m  ⚠ %s\033[0m\n' "$*"; }
error()   { printf '\033[0;31m  ✗ %s\033[0m\n' "$*" >&2; exit 1; }

# ── check Go ─────────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
  error "Go is not installed. Install Go $MIN_GO+ from https://go.dev/dl/ then re-run."
fi

GO_VERSION=$(go env GOVERSION | sed 's/go//')
info "Found Go $GO_VERSION"

# ── check Docker (warning only — Docker needed at runtime, not install time) ──
if ! command -v docker &>/dev/null; then
  warn "Docker not found. pg-factory needs Docker at runtime."
  warn "Install: https://docs.docker.com/get-docker/"
fi

# ── install ───────────────────────────────────────────────────────────────────
info "Installing pg-factory…"
go install "${REPO}@latest"

GOBIN="$(go env GOPATH)/bin"

# go install names the binary after the module's last path segment: pg-factory.
# Rename it to the short name "pg" that the tool expects to be called as.
if [[ -f "$GOBIN/pg-factory" ]]; then
  mv "$GOBIN/pg-factory" "$GOBIN/pg"
fi

# ── ensure GOBIN is on PATH ───────────────────────────────────────────────────
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$GOBIN"; then
  if [ -n "${ZSH_VERSION:-}" ]; then
    SHELL_RC="$HOME/.zshrc"
  elif [ -n "${BASH_VERSION:-}" ]; then
    SHELL_RC="$HOME/.bashrc"
  else
    SHELL_RC="$HOME/.profile"
  fi
  echo "" >> "$SHELL_RC"
  echo "# pg-factory" >> "$SHELL_RC"
  echo "export PATH=\"\$PATH:$GOBIN\"" >> "$SHELL_RC"
  export PATH="$PATH:$GOBIN"
  warn "Added $GOBIN to PATH in $SHELL_RC"
  warn "Restart your shell or run: source $SHELL_RC"
fi

success "pg installed → $GOBIN/pg"
echo ""

# ── first-time setup wizard ───────────────────────────────────────────────────
PGFACTORY_CONFIG="${XDG_CONFIG_HOME:-$HOME}/.pgfactory/config.json"
if [[ ! -f "$PGFACTORY_CONFIG" ]]; then
  info "Launching first-time setup…"
  echo ""
  "$GOBIN/pg" init
else
  info "Config already exists. Run 'pg init' to reconfigure."
fi
