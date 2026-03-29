# dev-install.ps1 — contributor installer for pg-factory (Windows PowerShell)
#
# Clones (or updates) the repo, builds from source, places the binary on
# your user PATH, then launches `pg init` if not already configured.
#
# Usage:
#   .\dev-install.ps1 [-Dir <source-dir>] [-BinDir <bin-dir>]
#
# Defaults:
#   -Dir     $HOME\src\pg-factory
#   -BinDir  $HOME\.local\bin

param(
    [string]$Dir    = (Join-Path $HOME "src\pg-factory"),
    [string]$BinDir = (Join-Path $HOME ".local\bin")
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$RepoUrl = "https://github.com/Mgkusumaputra/pg-factory.git"

function Write-Info    ($msg) { Write-Host "  `u{25B8} $msg" -ForegroundColor Cyan }
function Write-Success ($msg) { Write-Host "  `u{2713} $msg" -ForegroundColor Green }
function Write-Warn    ($msg) { Write-Host "  `u{26A0} $msg" -ForegroundColor Yellow }
function Write-Fail    ($msg) { Write-Host "  `u{2717} $msg" -ForegroundColor Red; exit 1 }

# ── check Go ──────────────────────────────────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Fail "Go 1.21+ is required. Install: https://go.dev/dl/"
}
$goVersion = (go env GOVERSION) -replace '^go', ''
Write-Info "Found Go $goVersion"

# ── check git ─────────────────────────────────────────────────────────────────
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Fail "git is required. Install: https://git-scm.com/download/win"
}

# ── check Docker ──────────────────────────────────────────────────────────────
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Warn "Docker not found — needed at runtime. https://docs.docker.com/get-docker/"
}

# ── clone or update ───────────────────────────────────────────────────────────
if (Test-Path (Join-Path $Dir ".git")) {
    Write-Info "Updating repo at $Dir..."
    git -C $Dir pull --ff-only
} else {
    Write-Info "Cloning pg-factory into $Dir..."
    git clone $RepoUrl $Dir
}

# ── build ─────────────────────────────────────────────────────────────────────
Write-Info "Building from source..."
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
$BinPath = Join-Path $BinDir "pg.exe"
Push-Location $Dir
go build -ldflags="-s -w" -o $BinPath .
Pop-Location

# ── ensure BinDir is on user PATH ─────────────────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$BinDir*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$BinDir", 'User')
    $env:Path = "$env:Path;$BinDir"
    Write-Warn "Added $BinDir to your user PATH — restart your terminal for new sessions."
}

Write-Success "pg built → $BinPath"
Write-Info "Source at: $Dir"
Write-Host ""

# ── first-time setup ──────────────────────────────────────────────────────────
$PgConfig = Join-Path $HOME ".pgfactory\config.json"
if (-not (Test-Path $PgConfig)) {
    Write-Info "Launching first-time setup..."
    Write-Host ""
    & $BinPath init
} else {
    Write-Info "Config already exists ($PgConfig). Run 'pg init' to reconfigure."
}
