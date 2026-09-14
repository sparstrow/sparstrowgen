<# Installs an already-downloaded sparstrowgen Windows bundle for this user.
   The release package contains the two executables beside this script. #>
param([string]$Destination = "$env:LOCALAPPDATA\sparstrowgen")

$ErrorActionPreference = 'Stop'
$source = Split-Path -Parent $PSCommandPath
$launcher = Join-Path $source 'sparstrowgen-launcher.exe'
$daemon = Join-Path $source 'sparstrowgen-daemon.exe'
$config = Join-Path $source 'sparstrowgen.json'
if (-not (Test-Path -LiteralPath $launcher) -or -not (Test-Path -LiteralPath $daemon) -or -not (Test-Path -LiteralPath $config)) {
  throw 'The sparstrowgen installation bundle is incomplete.'
}
New-Item -ItemType Directory -Force -Path $Destination | Out-Null
Copy-Item -LiteralPath $launcher -Destination (Join-Path $Destination 'sparstrowgen-launcher.exe') -Force
Copy-Item -LiteralPath $daemon -Destination (Join-Path $Destination 'sparstrowgen-daemon.exe') -Force
Copy-Item -LiteralPath $config -Destination (Join-Path $Destination 'sparstrowgen.json') -Force
& (Join-Path $Destination 'sparstrowgen-launcher.exe') install
if ($LASTEXITCODE -ne 0) { throw 'sparstrowgen could not register its pairing link.' }
Write-Host 'sparstrowgen is ready to pair. Return to the browser and choose Add computer.'
