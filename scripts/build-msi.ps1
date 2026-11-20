# build-msi.ps1 — WiX v7 MSI builder
# Builds an MSI installer from a prepared fontcreator233 package directory.
# Requires: WiX 5.0.2 (last freely-licensed release; v6+/v7 require the paid OSMF EULA)
#   dotnet tool install --global wix --version 5.0.2
# Usage:
#   .\scripts\build-msi.ps1 -Source dist\fontcreator233-v0.9.0-windows-amd64 -Version v0.9.0 -Arch x64 -Out dist\fontcreator233-v0.9.0-x64.msi

param(
    [Parameter(Mandatory)][string]$Source,     # dir containing fontcreator.exe + LICENSE + README
    [Parameter(Mandatory)][string]$Version,     # e.g. "v0.9.0" (leading v stripped for MSI)
    [Parameter(Mandatory)][ValidateSet("x64","arm64")][string]$Arch,
    [string]$Out                                 # output MSI path (default = "$Source\..\fontcreator233-$Version-$Arch.msi")
)

$ErrorActionPreference = "Stop"

# strip leading "v" for MSI version (must be x.y.z format)
$MsiVersion = $Version -replace '^v', ''

if (-not $Out) {
    $Out = Join-Path (Split-Path -Parent $Source) "fontcreator233-$Version-$Arch.msi"
}
$Out = [System.IO.Path]::GetFullPath($Out)

$SourceDir = [System.IO.Path]::GetFullPath($Source)
$WxsPath = [System.IO.Path]::GetFullPath("$PSScriptRoot\installer\fontcreator.wxs")

# sanity checks
if (-not (Test-Path $WxsPath)) { throw "fontcreator.wxs not found at $WxsPath" }
$exe = Join-Path $SourceDir "fontcreator.exe"
if (-not (Test-Path $exe))     { throw "fontcreator.exe not found in $SourceDir" }

Write-Host "fontcreator233 MSI build" -ForegroundColor Cyan
Write-Host "  wxs:      $WxsPath"
Write-Host "  source:   $SourceDir"
Write-Host "  version:  $MsiVersion  (msi)"
Write-Host "  arch:     $Arch"
Write-Host "  output:   $Out"

# WiX architecture mapping: x64 -> x64, arm64 -> arm64
$wixArch = if ($Arch -eq "arm64") { "arm64" } else { "x64" }

# Resolve a WiX 5 binary. WiX 6/7 refuse to build until the paid OSMF EULA is accepted,
# so fall back to a pinned 5.0.2 install under %TEMP% when needed.
$WixExe = "wix"
$verOut = (& wix --version 2>&1) -join " "
if ($LASTEXITCODE -ne 0) {
    Write-Host "  wix not found — installing 5.0.2 ..." -ForegroundColor Yellow
    dotnet tool install --global wix --version 5.0.2 | Out-Null
} elseif ($verOut -match 'version\s+([6-9])\.') {
    Write-Host "  detected WiX v$($Matches[1]) (paid OSMF EULA) — using pinned 5.0.2 instead" -ForegroundColor Yellow
    $pinned = Join-Path $env:TEMP "wix-5.0.2"
    if (-not (Test-Path (Join-Path $pinned "wix.exe"))) {
        dotnet tool install wix --version 5.0.2 --tool-path $pinned | Out-Null
    }
    $WixExe = Join-Path $pinned "wix.exe"
}

& $WixExe build $WxsPath `
    -d "Version=$MsiVersion" `
    -d "Source=$SourceDir" `
    -d "Out=$Out" `
    -arch $wixArch `
    -out $Out `
    -pdbtype none

if ($LASTEXITCODE -ne 0) { throw "wix build failed (exit $LASTEXITCODE)" }

# WiX 5 may emit a .wixpdb next to the output; it is not a release artifact.
Remove-Item ([System.IO.Path]::ChangeExtension($Out, ".wixpdb")) -Force -ErrorAction SilentlyContinue

$size = [math]::Round((Get-Item $Out).Length / 1MB, 2)
$hash = (Get-FileHash $Out -Algorithm SHA256).Hash.ToLower()

Write-Host "`nMSI ready." -ForegroundColor Green
Write-Host "  $Out — $size MB"
Write-Host "  SHA256: $hash"

# emit checksum
"$hash  $(Split-Path $Out -Leaf)" | Out-File -Encoding ascii "$Out.sha256"
