# install.ps1 — end-user installer for pg-factory (Windows PowerShell)
# Usage (run in PowerShell as your normal user — no Admin required):
#   irm https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.ps1 | iex
#
# Requires: Go 1.21+  |  Docker Desktop

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repo = "github.com/Mgkusumaputra/pg-factory"

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

# ── install ───────────────────────────────────────────────────────────────────
Write-Info "Installing pg-factory..."
go install "${Repo}@latest"

$GoPath = go env GOPATH
$GoBin  = Join-Path $GoPath "bin"

# go install names the binary after the module's last path segment: pg-factory.
# Rename it to the short name "pg" that the tool expects to be called as.
$builtExe  = Join-Path $GoBin "pg-factory.exe"
$targetExe = Join-Path $GoBin "pg.exe"
if (Test-Path $builtExe) {
    if (Test-Path $targetExe) { Remove-Item $targetExe -Force }
    Rename-Item -Path $builtExe -NewName "pg.exe"
}

# ── ensure GOBIN is on the user PATH ─────────────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$GoBin*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$GoBin", 'User')
    $env:Path = "$env:Path;$GoBin"
    Write-Warn "Added $GoBin to your user PATH."
    Write-Warn "Restart your terminal for the change to take effect in new sessions."
}

Write-Success "pg installed → $GoBin\pg.exe"
Write-Host ""

# ── first-time setup wizard ───────────────────────────────────────────────────
$ConfigFile = Join-Path $env:USERPROFILE ".pgfactory\config.json"
if (-not (Test-Path $ConfigFile)) {
    Write-Info "Launching first-time setup..."
    Write-Host ""
    & "$GoBin\pg.exe" init
} else {
    Write-Info "Config already exists. Run 'pg init' to reconfigure."
}
