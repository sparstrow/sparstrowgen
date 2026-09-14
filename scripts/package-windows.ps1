<# Builds one daemon release for one server:

   sparstrowgen-setup.exe          the self-installing Windows executable
   sparstrowgen-setup.exe.sha256
   sparstrowgen-update.json        the manifest installed computers check: version, installer URL, SHA-256

   Releases are built and published by .github/workflows/daemon-release.yml when a
   `daemon-v<Version>` tag is pushed (docs/runbooks/daemon-release.md). Run it by
   hand only to try a build. No key is involved (docs/Decisions.md D-035). The
   executable itself is unsigned for Windows, tracked in docs/KnownGaps.md G-32. #>
param(
  [Parameter(Mandatory = $true)][string]$Version,
  [string]$ServerAPI = 'https://api.sparstrow.com',
  [string]$ServerWS = 'wss://api.sparstrow.com/daemon',
  [string]$UpdateURL = 'https://github.com/sparstrow/sparstrowgen/releases/latest/download/sparstrowgen-update.json',
  [string]$Output = "$PSScriptRoot\..\dist\windows"
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw "Version must be x.y.z, got '$Version'" }
$repo = Split-Path -Parent $PSScriptRoot
New-Item -ItemType Directory -Force -Path $Output | Out-Null
# Absolute before building: go build -C resolves a relative -o from server/ (B-17).
$bundle = (Resolve-Path -LiteralPath $Output).Path
$exe = Join-Path $bundle 'sparstrowgen-setup.exe'
$tool = Join-Path $bundle 'releasetool.exe'

& go build -C "$repo\server" -o $tool ./cmd/releasetool
if ($LASTEXITCODE -ne 0) { throw 'building releasetool failed' }

$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
try {
  $flags = "-s -w -H=windowsgui -X main.version=$Version -X main.releaseAPI=$ServerAPI -X main.releaseWS=$ServerWS -X main.releaseUpdateURL=$UpdateURL"
  & go build -C "$repo\server" -trimpath -ldflags $flags -o $exe ./cmd/daemon
  if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
} finally {
  Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $exe).Hash.ToLower()
"$hash  sparstrowgen-setup.exe" | Set-Content -LiteralPath "$exe.sha256" -Encoding ascii

$installerURL = "https://github.com/sparstrow/sparstrowgen/releases/download/daemon-v$Version/sparstrowgen-setup.exe"
& $tool manifest -exe $exe -version $Version -url $installerURL -out $bundle | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'writing the update manifest failed' }
Remove-Item -LiteralPath $tool

Write-Host "Built $exe (v$Version)"
Write-Host "SHA-256 $hash"
Write-Host "Wrote $(Join-Path $bundle 'sparstrowgen-update.json') for $installerURL"
