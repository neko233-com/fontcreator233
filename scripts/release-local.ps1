# fontcreator233 local release — Windows PowerShell 5
#
# Cross-compiles every supported platform, builds the Windows MSI, creates the git tag,
# and publishes the GitHub Release. No GitHub Actions involved.
#
# Usage:
#   .\scripts\release-local.ps1 -Version v0.10.6
#   .\scripts\release-local.ps1 -Version v0.10.6 -SkipTest
#   .\scripts\release-local.ps1 -Version v0.10.6 -NoUpload     # package + tag only
#   .\scripts\release-local.ps1 -Version v0.10.6 -NoTag        # rebuild assets only

param(
    [Parameter(Mandatory)][string]$Version,
    [switch]$SkipTest,
    [switch]$NoUpload,
    [switch]$NoTag,
    [string]$NotesFile,
    [string]$OutDir = "dist"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $RepoRoot

if ($Version -notmatch '^v\d+\.\d+\.\d+$') {
    throw "Version must look like v1.2.3 (got '$Version')"
}

# ---------------------------------------------------------------- preflight
Write-Host "fontcreator233 local release — $Version" -ForegroundColor Cyan

$dirty = & git status --porcelain 2>$null
if ($dirty) {
    Write-Host "Working tree is dirty — commit or stash first:" -ForegroundColor Red
    $dirty | ForEach-Object { Write-Host "  $_" }
    exit 1
}

$existingTag = (& git tag --list $Version 2>$null)
if ($existingTag -and -not $NoTag) {
    Write-Host "Tag $Version already exists — pick a new version." -ForegroundColor Red
    exit 1
}

if (-not $NoUpload) {
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & gh auth status *> $null
    $ghOk = ($LASTEXITCODE -eq 0)
    $ErrorActionPreference = $prevEap
    if (-not $ghOk) { Write-Host "gh is not authenticated (run: gh auth login)" -ForegroundColor Red; exit 1 }
}

$ShortSha = (& git rev-parse --short HEAD).Trim()
$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$env:CGO_ENABLED = "0"

Write-Host "  commit:    $ShortSha"
Write-Host "  built:     $BuildDate"
Write-Host "  go:        $(& go version)"

# ---------------------------------------------------------------- 1 format/vet
Write-Host "`n[1/7] Format + Vet" -ForegroundColor Cyan
$needsFmt = & gofmt -l . 2>$null
if ($needsFmt) {
    Write-Host "  gofmt dirty:" -ForegroundColor Red
    $needsFmt | ForEach-Object { Write-Host "    $_" }
    exit 1
}
& go vet ./...
if ($LASTEXITCODE -ne 0) { Write-Host "  go vet failed" -ForegroundColor Red; exit 1 }
Write-Host "  ok" -ForegroundColor Green

# ---------------------------------------------------------------- 2 tests
if (-not $SkipTest) {
    Write-Host "`n[2/7] Tests" -ForegroundColor Cyan
    & go test ./... -count=1
    if ($LASTEXITCODE -ne 0) { Write-Host "  tests failed" -ForegroundColor Red; exit 1 }
    Write-Host "  ok" -ForegroundColor Green
} else {
    Write-Host "`n[2/7] Tests — SKIP" -ForegroundColor Yellow
}

# ---------------------------------------------------------------- 3 build
Write-Host "`n[3/7] Cross-compile" -ForegroundColor Cyan
$Ldflags = "-s -w " +
    "-X github.com/neko233-com/fontcreator233/internal/agentschema.Version=$Version " +
    "-X github.com/neko233-com/fontcreator233/internal/agentschema.Commit=$ShortSha " +
    "-X github.com/neko233-com/fontcreator233/internal/agentschema.BuildDate=$BuildDate"

$Targets = @(
    @{ os = "windows"; arch = "amd64"; bin = "fontcreator.exe" },
    @{ os = "darwin";  arch = "amd64"; bin = "fontcreator"     },
    @{ os = "darwin";  arch = "arm64"; bin = "fontcreator"     }
)

if (Test-Path $OutDir) { Remove-Item $OutDir -Recurse -Force }
New-Item -ItemType Directory -Path $OutDir | Out-Null

$Assets = @()
$WindowsPkgDir = $null

foreach ($t in $Targets) {
    $name = "fontcreator233-$Version-$($t.os)-$($t.arch)"
    $pkgDir = Join-Path $OutDir $name
    New-Item -ItemType Directory -Path $pkgDir | Out-Null

    $env:GOOS = $t.os
    $env:GOARCH = $t.arch
    $outBin = Join-Path $pkgDir $t.bin
    & go build -ldflags $Ldflags -trimpath -o $outBin .
    if ($LASTEXITCODE -ne 0) { throw "build failed for $($t.os)/$($t.arch)" }

    Copy-Item LICENSE   $pkgDir\ -ErrorAction SilentlyContinue
    Copy-Item NOTICE.md $pkgDir\ -ErrorAction SilentlyContinue
    Copy-Item README.md $pkgDir\ -ErrorAction SilentlyContinue

    $tgz = Join-Path $OutDir "$name.tar.gz"
    & tar.exe -czf $tgz -C $OutDir $name
    if ($LASTEXITCODE -ne 0) { throw "tar failed for $name" }

    $hash = (Get-FileHash $tgz -Algorithm SHA256).Hash.ToLower()
    "$hash  $name.tar.gz" | Out-File -Encoding ascii "$tgz.sha256"

    $Assets += $tgz
    $Assets += "$tgz.sha256"

    $mb = [math]::Round((Get-Item $tgz).Length / 1MB, 2)
    Write-Host ("  {0,-42} {1,7} MB" -f "$name.tar.gz", $mb) -ForegroundColor Green

    if ($t.os -eq "windows") { $WindowsPkgDir = $pkgDir }
}

# restore host env
$env:GOOS = ""
$env:GOARCH = ""

# ---------------------------------------------------------------- 4 MSI
Write-Host "`n[4/7] Windows MSI" -ForegroundColor Cyan
$msi = Join-Path $OutDir "fontcreator233-$Version-windows-amd64.msi"
& "$PSScriptRoot\build-msi.ps1" -Source $WindowsPkgDir -Version $Version -Arch x64 -Out $msi
if ($LASTEXITCODE -ne 0) { throw "MSI build failed" }
$Assets += $msi
$Assets += "$msi.sha256"

# ---------------------------------------------------------------- 5 summary
Write-Host "`n[5/7] Assets" -ForegroundColor Cyan
Get-ChildItem $OutDir -File | Sort-Object Name | ForEach-Object {
    Write-Host ("  {0,-52} {1,8:N2} MB" -f $_.Name, ($_.Length / 1MB))
}

# ---------------------------------------------------------------- 6 tag
if (-not $NoTag) {
    Write-Host "`n[6/7] Tag" -ForegroundColor Cyan
    & git tag -a $Version -m "fontcreator233 $Version"
    if ($LASTEXITCODE -ne 0) { throw "git tag failed" }
    & git push origin "refs/tags/$Version"
    if ($LASTEXITCODE -ne 0) { throw "git push tag failed" }
    Write-Host "  pushed $Version" -ForegroundColor Green
} else {
    Write-Host "`n[6/7] Tag — SKIP" -ForegroundColor Yellow
}

# ---------------------------------------------------------------- 7 upload
if (-not $NoUpload) {
    Write-Host "`n[7/7] Publish GitHub Release" -ForegroundColor Cyan

    $notes = $NotesFile
    $tmpNotes = $null
    if (-not $notes) {
        $tmpNotes = Join-Path $env:TEMP "fc233-notes-$Version.md"
        @"
## fontcreator233 $Version

Deterministic, offline, agent-native font creator.

### Install
``````bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/neko233-com/fontcreator233/main/scripts/install.sh | bash

# Windows (portable)
powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/neko233-com/fontcreator233/main/scripts/install.ps1 | iex"

# Windows MSI (recommended, needs admin)
#   double-click fontcreator233-$Version-windows-amd64.msi
``````

### Update
``````bash
fontcreator update --check
fontcreator update
``````

### Checksums
``````bash
sha256sum -c fontcreator233-$Version-*.tar.gz.sha256
``````
"@ | Out-File -Encoding utf8 $tmpNotes
        $notes = $tmpNotes
    }

    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & gh release view $Version *> $null
    $releaseExists = ($LASTEXITCODE -eq 0)
    $ErrorActionPreference = $prevEap

    if ($releaseExists) {
        Write-Host "  release exists — uploading with --clobber"
        & gh release upload $Version $Assets --clobber
    } else {
        & gh release create $Version $Assets --title "fontcreator233 $Version" --notes-file $notes
    }
    if ($LASTEXITCODE -ne 0) { throw "gh release publish failed" }
    if ($tmpNotes) { Remove-Item $tmpNotes -Force -ErrorAction SilentlyContinue }

    Write-Host "  published: https://github.com/neko233-com/fontcreator233/releases/tag/$Version" -ForegroundColor Green
} else {
    Write-Host "`n[7/7] Upload — SKIP" -ForegroundColor Yellow
}

Write-Host "`nRelease $Version complete." -ForegroundColor Green
