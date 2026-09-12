<#
    Start sparstrowgen locally, with development credentials.

    The Makefile is the documented entry point, but `make` is not installed on
    the owner's machine — so this is the one that actually runs. Both set the
    same variables; if you change one, change the other.

    NOT SECRETS. The daemon token below is in a public repository and only ever
    unlocks a server on localhost. Production sets these in Coolify.

    There is no password here. Accounts live in Postgres, so the first run
    prints a setup code and the app asks you to create one — the same flow as
    production, which is the point.

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

# The first run on an empty database has no account, and the way in is the setup
# code the server prints at startup. It goes to the log file rather than a
# console here, so without this you would have to know to go and read it.
$session = $null
foreach ($attempt in 1..20) {
    try {
        $session = Invoke-RestMethod -Uri 'http://localhost:8080/api/auth/session' -TimeoutSec 2
        break
    } catch {
        Start-Sleep -Milliseconds 500
    }
}

if ($null -eq $session) {
    Write-Host 'the server did not answer — check ' -NoNewline
    Write-Host "$env:TEMP\sg-server.log"
} elseif ($session.claimed) {
    Write-Host 'sign in at http://localhost:3000 with the account you created'
} else {
    $code = Select-String -Path "$env:TEMP\sg-server.log" -Pattern 'setup_code=(\S+)' |
        Select-Object -Last 1 |
        ForEach-Object { $_.Matches[0].Groups[1].Value }
    Write-Host 'no account yet. Open http://localhost:3000 and create one.'
    Write-Host 'setup code: ' -NoNewline
    Write-Host $code
}
