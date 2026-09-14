<# Builds sparstrowgen-setup.exe: one self-installing Windows executable that
   knows one server. Opening it installs sparstrowgen for the current Windows
   user; the browser's sparstrowgen:// link then pairs it.

   Unsigned. Signing is tracked in docs/KnownGaps.md G-32. #>
param(
  [string]$ServerAPI = 'https://api.sparstrow.com',
  [string]$ServerWS = 'wss://api.sparstrow.com/daemon',
  [string]$Output = "$PSScriptRoot\..\dist\windows"
)
$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
New-Item -ItemType Directory -Force -Path $Output | Out-Null
# Absolute before building: go build -C resolves a relative -o from server/ (B-17).
$bundle = (Resolve-Path -LiteralPath $Output).Path
$exe = Join-Path $bundle 'sparstrowgen-setup.exe'
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
$flags = "-s -w -H=windowsgui -X main.releaseAPI=$ServerAPI -X main.releaseWS=$ServerWS"
& go build -C "$repo\server" -trimpath -ldflags $flags -o $exe ./cmd/daemon
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $exe).Hash.ToLower()
"$hash  sparstrowgen-setup.exe" | Set-Content -LiteralPath "$exe.sha256" -Encoding ascii
Write-Host "Built $exe"
Write-Host "SHA-256 $hash"
