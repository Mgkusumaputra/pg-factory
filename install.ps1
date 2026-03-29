# install.ps1 — end-user installer for pg-factory (Windows PowerShell)
# Usage (run in PowerShell as your normal user — no Admin required):
#   irm https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.ps1 | iex
#
# Requires: Go 1.21+  |  Docker Desktop

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repo    = "github.com/Mgkusumaputra/pg-factory"
$RepoURL = "https://github.com/Mgkusumaputra/pg-factory.git"

function Write-Info    ($msg) { Write-Host "  `u{25B8} $msg" -ForegroundColor Cyan }
function Write-Success ($msg) { Write-Host "  `u{2713} $msg" -ForegroundColor Green }
function Write-Warn    ($msg) { Write-Host "  `u{26A0} $msg" -ForegroundColor Yellow }
function Write-Fail    ($msg) { Write-Host "  `u{2717} $msg" -ForegroundColor Red; exit 1 }

# ── check Go ──────────────────────────────────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Fail "Go is not installed. Install Go 1.21+ from https://go.dev/dl/ then re-run."
}
$goVersion = (go env GOVERSION) -replace '^go', ''
Write-Info "Found Go $goVersion"

# ── check Docker ──────────────────────────────────────────────────────────────
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Warn "Docker not found. pg-factory needs Docker at runtime."
    Write-Warn "Install: https://docs.docker.com/get-docker/"
}

# ── resolve GOBIN (respects $env:GOBIN override) ─────────────────────────────
$GoBin = go env GOBIN
if (-not $GoBin) { $GoBin = Join-Path (go env GOPATH) "bin" }
New-Item -ItemType Directory -Force -Path $GoBin | Out-Null
$TargetExe = Join-Path $GoBin "pg.exe"

# ── build ─────────────────────────────────────────────────────────────────────
# Strategy: build from local source if we're in the repo root (dev workflow),
# otherwise clone from GitHub and build — avoids the go install binary-rename
# problem where go install always names the binary after the module path
# ("pg-factory") rather than the intended command name ("pg").

# $PSCommandPath is empty when the script is piped via irm | iex (no file on disk).
# In that case treat it as a remote install and go straight to the clone path.
$ScriptDir = if ($PSCommandPath) { Split-Path -Parent $PSCommandPath } else { $null }
$IsInRepo  = $ScriptDir -and
             (Test-Path (Join-Path $ScriptDir "main.go")) -and
             (Test-Path (Join-Path $ScriptDir "go.mod"))

if ($IsInRepo) {
    # ── local / dev path ──────────────────────────────────────────────────────
    Write-Info "Building from local source..."
    Push-Location $ScriptDir
    go build -ldflags="-s -w" -o $TargetExe .
    Pop-Location
} else {
    # ── remote / end-user path ────────────────────────────────────────────────
    Write-Info "Cloning and building pg-factory..."
    $TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "pg-factory-install"
    if (Test-Path $TmpDir) { Remove-Item $TmpDir -Recurse -Force }

    if (Get-Command git -ErrorAction SilentlyContinue) {
        git clone --depth 1 $RepoURL $TmpDir 2>&1 | Out-Null
    } else {
        # Fallback: go install + rename (git not available)
        Write-Info "git not found, falling back to go install..."
        go install "${Repo}@latest"
        $BuiltExe = Join-Path $GoBin "pg-factory.exe"
        if (Test-Path $BuiltExe) {
            if (Test-Path $TargetExe) { Remove-Item $TargetExe -Force }
            Rename-Item -Path $BuiltExe -NewName "pg.exe"
        }
        if (-not (Test-Path $TargetExe)) {
            Write-Fail "Installation failed: pg.exe not found at $TargetExe"
        }
        # Skip the build step below
        $TmpDir = $null
    }

    if ($TmpDir -and (Test-Path $TmpDir)) {
        Push-Location $TmpDir
        go build -ldflags="-s -w" -o $TargetExe .
        Pop-Location
        Remove-Item $TmpDir -Recurse -Force
    }
}

if (-not (Test-Path $TargetExe)) {
    Write-Fail "Build failed: pg.exe not found at $TargetExe"
}

# ── ensure GOBIN is on the user PATH ─────────────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$GoBin*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$GoBin", 'User')
    $env:Path = "$env:Path;$GoBin"
    Write-Warn "Added $GoBin to your user PATH."
    Write-Warn "Restart your terminal for the change to take effect in new sessions."
}

Write-Success "pg installed → $TargetExe"
Write-Host ""

# ── first-time setup wizard ───────────────────────────────────────────────────
$ConfigFile = Join-Path $env:USERPROFILE ".pgfactory\config.json"
if (-not (Test-Path $ConfigFile)) {
    Write-Info "Launching first-time setup..."
    Write-Host ""
    & $TargetExe init
} else {
    Write-Info "Config already exists. Run 'pg init' to reconfigure."
}
