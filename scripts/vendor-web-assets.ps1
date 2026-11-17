# Regenerate the vendored web specimen assets.
#
# Requirements: PowerShell 7+, Node.js 20+, npm, npx/esbuild.
# Runtime fontcreator233 never runs this script; the generated files are
# committed under internal/specimen/webassets.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$assets = Join-Path $root "internal/specimen/webassets"
$geistDir = Join-Path $assets "geist"
$vendorDir = Join-Path $assets "vendor"
$licenseDir = Join-Path $assets "licenses"
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("fontcreator233-web-" + [guid]::NewGuid())

New-Item -ItemType Directory -Force $geistDir, $vendorDir, $licenseDir, $tmp | Out-Null

$geistBase = "https://raw.githubusercontent.com/vercel/geist-font/main"
Invoke-WebRequest "$geistBase/fonts/Geist/webfonts/Geist%5Bwght%5D.woff2" -OutFile (Join-Path $geistDir "Geist-Variable.woff2")
Invoke-WebRequest "$geistBase/fonts/GeistMono/webfonts/GeistMono%5Bwght%5D.woff2" -OutFile (Join-Path $geistDir "GeistMono-Variable.woff2")
Invoke-WebRequest "$geistBase/OFL.txt" -OutFile (Join-Path $licenseDir "Geist-OFL.txt")

Push-Location $tmp
try {
  npm install --no-audit --no-fund "@tanstack/virtual-core@3.19.11"
  npx --yes esbuild "node_modules/@tanstack/virtual-core/dist/esm/index.js" `
    --bundle --format=iife --global-name=TanStackVirtual --minify `
    --outfile=(Join-Path $vendorDir "tanstack-virtual.iife.js")
  Copy-Item "node_modules/@tanstack/virtual-core/LICENSE" (Join-Path $licenseDir "TanStack-Virtual-MIT.txt") -Force
} finally {
  Pop-Location
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

Write-Host "Vendored Geist + TanStack Virtual into $assets"
Get-ChildItem -Recurse $assets | Select-Object FullName, Length
