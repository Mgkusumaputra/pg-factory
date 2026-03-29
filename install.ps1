# install.ps1 — end-user installer for pg-factory (Windows PowerShell)
# Usage (run in PowerShell as your normal user — no Admin required):
#   irm https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.ps1 | iex
#
# Requires: Go 1.21+  |  Docker Desktop

$ErrorActionPreference = 'Stop'

$RepoURL = "https://github.com/Mgkusumaputra/pg-factory.git"
$RepoGo  = "github.com/Mgkusumaputra/pg-factory"

function Write-Info    ($msg) { Write-Host "  > $msg" -ForegroundColor Cyan }
function Write-Success ($msg) { Write-Host "  + $msg" -ForegroundColor Green }
function Write-Warn    ($msg) { Write-Host "  ! $msg" -ForegroundColor Yellow }
function Write-Fail    ($msg) { Write-Host "  x $msg" -ForegroundColor Red; exit 1 }

# ── check Go ──────────────────────────────────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Fail "Go is not installed. Install Go 1.21+ from https://go.dev/dl/ then re-run."
}
Write-Info "Found Go $((go env GOVERSION) -replace '^go','')"

# ── check Docker ──────────────────────────────────────────────────────────────
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Warn "Docker not found. pg-factory needs Docker at runtime."
    Write-Warn "Install: https://docs.docker.com/get-docker/"
}

# ── resolve install dir ───────────────────────────────────────────────────────
# go env GOBIN returns empty string when not set; fall back to GOPATH\bin.
$RawGobin = (go env GOBIN).Trim()
$GoBin    = if ($RawGobin -ne '') { $RawGobin } else { Join-Path (go env GOPATH).Trim() 'bin' }
New-Item -ItemType Directory -Force -Path $GoBin | Out-Null
$TargetExe = Join-Path $GoBin 'pg.exe'

# ── build ─────────────────────────────────────────────────────────────────────
# When piped via irm|iex, $PSCommandPath is empty — no script file on disk.
# Use $PSCommandPath only when it is a real, non-empty path.
$IsLocal = ($PSCommandPath -ne $null) -and
           ($PSCommandPath.Trim() -ne '') -and
           (Test-Path (Join-Path (Split-Path -Parent $PSCommandPath) 'main.go'))

if ($IsLocal) {
    # Running directly from the repo — build in place.
    Write-Info "Building from local source..."
    $RepoDir = Split-Path -Parent $PSCommandPath
    Push-Location $RepoDir
    & go build -ldflags='-s -w' -o $TargetExe .
    Pop-Location

} elseif (Get-Command git -ErrorAction SilentlyContinue) {
    # Remote install — clone a shallow copy, build, clean up.
    Write-Info "Cloning and building pg-factory..."
    $TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) 'pg-factory-install'
    if (Test-Path $TmpDir) { Remove-Item $TmpDir -Recurse -Force }
    git clone --depth 1 --quiet $RepoURL $TmpDir
    Push-Location $TmpDir
    & go build -ldflags='-s -w' -o $TargetExe .
    Pop-Location
    Remove-Item $TmpDir -Recurse -Force

} else {
    # Last resort: go install + rename (no git available).
    Write-Info "git not found — falling back to go install..."
    & go install "${RepoGo}@latest"
    $BuiltExe = Join-Path $GoBin 'pg-factory.exe'
    if (Test-Path $BuiltExe) {
        if (Test-Path $TargetExe) { Remove-Item $TargetExe -Force }
        Rename-Item -LiteralPath $BuiltExe -NewName 'pg.exe'
    }
}

if (-not (Test-Path $TargetExe)) {
    Write-Fail "Build failed — pg.exe not found at $TargetExe"
}

# ── add GOBIN to user PATH ────────────────────────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$GoBin*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$GoBin", 'User')
    $env:Path = "$env:Path;$GoBin"
    Write-Warn "Added $GoBin to your PATH — restart your terminal for new sessions."
}

Write-Success "pg installed -> $TargetExe"
Write-Host ''

# ── first-time setup wizard ───────────────────────────────────────────────────
$Config = Join-Path $env:USERPROFILE '.pgfactory\config.json'
if (-not (Test-Path $Config)) {
    Write-Info "Launching first-time setup..."
    Write-Host ''
    & $TargetExe init
} else {
    Write-Info "Config already exists. Run 'pg init' to reconfigure."
}
