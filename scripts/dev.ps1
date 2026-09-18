<#
    Start sparstrowgen locally, with development credentials.

    The Makefile is the documented entry point, but `make` is not installed on
    the owner's machine — so this is the one that actually runs. Both set the
    same variables; if you change one, change the other.

    NOT SECRETS. The daemon token this uses is in a public repository and only
    ever unlocks a server on localhost. Production sets these in Coolify.

    The ports, the database volume, the daemon's home and that token all come
    from `devstack`, which gives every worktree its own set (WORKFLOW.md, Local
    isolation). This checkout gets the same 5433/8080/3000 it always had; a
    second worktree working at the same time gets its own, instead of the two
    quietly sharing one database and one daemon credential.

    There is no password here. Accounts live in Postgres, so the first run
    prints a setup code and the app asks you to create one — the same flow as
    production, which is the point.

    The server refuses to start without them, deliberately: there is no
    "authentication off" mode that could reach production by accident.

    Usage:
        scripts\dev.ps1            # start Postgres, server, daemon and web
        scripts\dev.ps1 -Stop      # stop them again
        scripts\dev.ps1 -NoWeb     # server and daemon only
        scripts\dev.ps1 -NoDb      # leave Postgres alone
#>
param(
    [switch]$Stop,
    [switch]$NoWeb,
    [switch]$NoDb
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot

# This worktree's own stack, reserved once and remembered. Every variable the
# server, daemon, web app, Compose file and goose need comes from here, so
# nothing below names a port.
Write-Host 'reserving this worktree''s stack...'
& go build -C "$repo\server" -o "$repo\bin\devstack.exe" ./cmd/devstack
if ($LASTEXITCODE -ne 0) { throw 'devstack build failed' }
& "$repo\bin\devstack.exe" env -dir $repo -format powershell | ForEach-Object { Invoke-Expression $_ }

$env:MAIL_TRANSPORT = 'log'

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

# Compose reads SPARSTROWGEN_DB_PORT and COMPOSE_PROJECT_NAME from the stack
# above, so this worktree's database is its own container, its own port and its
# own volume.
if (-not $NoDb) {
    Write-Host "starting postgres on :$env:SPARSTROWGEN_DB_PORT ..."
    & docker compose -f "$repo\compose.dev.yaml" up -d
    if ($LASTEXITCODE -ne 0) { throw 'postgres did not start' }
    & goose -dir "$repo\server\migrations" up
    if ($LASTEXITCODE -ne 0) { Write-Warning 'migrations did not run; is goose installed?' }
}

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

Write-Host "server  : $env:SERVER_API  (log: " -NoNewline
Write-Host "$env:TEMP\sg-server.log)"
Write-Host 'daemon  : connected to the above (log: ' -NoNewline
Write-Host "$env:TEMP\sg-daemon.log)"

if (-not $NoWeb) {
    Write-Host "web     : run ``pnpm dev`` in another terminal (it reads PORT=$env:PORT), or use the Browser pane"
}
Write-Host "slot    : $env:SPARSTROWGEN_SLOT  (database :$env:SPARSTROWGEN_DB_PORT, daemon home $env:SPARSTROWGEN_HOME)"
Write-Host ''

# The first run on an empty database has no account, and the way in is the setup
# code the server prints at startup. It goes to the log file rather than a
# console here, so without this you would have to know to go and read it.
$session = $null
foreach ($attempt in 1..20) {
    try {
        $session = Invoke-RestMethod -Uri "$env:SERVER_API/api/auth/session" -TimeoutSec 2
        break
    } catch {
        Start-Sleep -Milliseconds 500
    }
}

if ($null -eq $session) {
    Write-Host 'the server did not answer — check ' -NoNewline
    Write-Host "$env:TEMP\sg-server.log"
} elseif ($session.claimed) {
    Write-Host "sign in at $env:WEB_ORIGIN with the account you created"
} else {
    $code = Select-String -Path "$env:TEMP\sg-server.log" -Pattern 'setup_code=(\S+)' |
        Select-Object -Last 1 |
        ForEach-Object { $_.Matches[0].Groups[1].Value }
    Write-Host "no account yet. Open $env:WEB_ORIGIN and create one."
    Write-Host 'setup code: ' -NoNewline
    Write-Host $code
}
