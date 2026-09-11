<#
    Start sparstrowgen locally, with development credentials.

    The Makefile is the documented entry point, but `make` is not installed on
    the owner's machine — so this is the one that actually runs. Both set the
    same variables; if you change one, change the other.

    NOT SECRETS. The hash below is of the password "sparstrowgen-dev", it is in
    a public repository, and it only ever unlocks a server on localhost.
    Production sets these in Coolify, with a real hash from `server -hashpw`.

    The server refuses to start without them, deliberately: there is no
    "authentication off" mode that could reach production by accident.

    Usage:
        scripts\dev.ps1            # start server, daemon and web
        scripts\dev.ps1 -Stop      # stop them again
        scripts\dev.ps1 -NoWeb     # server and daemon only
#>
param(
    [switch]$Stop,
    [switch]$NoWeb
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot

$env:OWNER_PASSWORD_HASH = '$argon2id$v=19$m=19456,t=2,p=1$Fajya1dgZWWINWrLW54w6Q$sxOqelkhkixUwTt6CR21pfVYbczrOiM1C6WdNlheoIU'
$env:DAEMON_TOKEN        = 'dev-daemon-token-not-a-secret-0123456789'
$env:WEB_ORIGIN          = 'http://localhost:3000'
$env:SESSION_SECURE      = 'false'
$env:DATABASE_URL        = 'postgres://sparstrowgen:sparstrowgen@localhost:5433/sparstrowgen?sslmode=disable'

function Stop-Sparstrowgen {
    Get-Process server, daemon -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -and $_.Path.StartsWith($repo) } |
        ForEach-Object { Write-Host "stopping $($_.ProcessName) (pid $($_.Id))"; Stop-Process -Id $_.Id -Force }
}

if ($Stop) {
    Stop-Sparstrowgen
    return
}

Stop-Sparstrowgen
Start-Sleep -Milliseconds 500

# Rebuilt rather than `go run`, so the running process has a path this script can
# find again to stop it.
Write-Host 'building...'
& go build -C "$repo\server" -o "$repo\bin\server.exe" ./cmd/server
if ($LASTEXITCODE -ne 0) { throw 'server build failed' }
& go build -C "$repo\server" -o "$repo\bin\daemon.exe" ./cmd/daemon
if ($LASTEXITCODE -ne 0) { throw 'daemon build failed' }

# The daemon passes this through to the claude CLI it spawns. Without it a
# nested `claude -p` meets "OAuth access token has expired" and cannot recover
# (docs/Capabilities.md).
$token = [Environment]::GetEnvironmentVariable('CLAUDE_CODE_OAUTH_TOKEN', 'User')
if ($token) {
    $env:CLAUDE_CODE_OAUTH_TOKEN = $token
} else {
    Write-Warning 'CLAUDE_CODE_OAUTH_TOKEN is not set — claude turns will fail. See docs/runbooks/claude-headless-auth.md'
}

Start-Process -FilePath "$repo\bin\server.exe" -WorkingDirectory $repo `
    -RedirectStandardOutput "$env:TEMP\sg-server.log" -RedirectStandardError "$env:TEMP\sg-server.err" `
    -WindowStyle Hidden
Start-Process -FilePath "$repo\bin\daemon.exe" -WorkingDirectory $repo `
    -RedirectStandardOutput "$env:TEMP\sg-daemon.log" -RedirectStandardError "$env:TEMP\sg-daemon.err" `
    -WindowStyle Hidden

Write-Host 'server  : http://localhost:8080  (log: ' -NoNewline
Write-Host "$env:TEMP\sg-server.log)"
Write-Host 'daemon  : connected to the above (log: ' -NoNewline
Write-Host "$env:TEMP\sg-daemon.log)"

if (-not $NoWeb) {
    Write-Host 'web     : run `pnpm dev` in another terminal, or use the Browser pane'
}
Write-Host ''
Write-Host 'sign in with the development password: sparstrowgen-dev'
