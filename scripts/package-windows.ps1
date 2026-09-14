<# Builds the unsigned Windows bundle used by local and candidate verification.
   Publishing and signing the resulting archive remain release responsibilities. #>
param(
  [Parameter(Mandatory = $true)][string]$ServerAPI,
  [Parameter(Mandatory = $true)][string]$ServerWS,
  [string]$Output = "$PSScriptRoot\..\dist\windows"
)
$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
New-Item -ItemType Directory -Force -Path $Output | Out-Null
$bundle = (Resolve-Path -LiteralPath $Output).Path
& go build -C "$repo\server" -o "$bundle\sparstrowgen-daemon.exe" ./cmd/daemon
& go build -C "$repo\server" -o "$bundle\sparstrowgen-launcher.exe" ./cmd/launcher
Copy-Item -LiteralPath "$PSScriptRoot\install-windows.ps1" -Destination "$bundle\install-windows.ps1" -Force
@{ serverApi = $ServerAPI; serverWs = $ServerWS } | ConvertTo-Json | Set-Content -LiteralPath "$bundle\sparstrowgen.json" -NoNewline
Compress-Archive -Path "$bundle\sparstrowgen-daemon.exe","$bundle\sparstrowgen-launcher.exe","$bundle\sparstrowgen.json","$bundle\install-windows.ps1" -DestinationPath "$bundle\sparstrowgen-windows.zip" -Force
