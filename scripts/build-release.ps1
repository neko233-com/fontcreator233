# fontcreator233 local release builder — Windows PowerShell 5
# Go build (ldflags) → test → sha256 → zip → report
# Usage: .\scripts\build-release.ps1 [-Version v0.9.0] [-SkipTest] [-OutDir dist]

param(
    [string]$Version,
    [switch]$SkipTest,
    [switch]$WithMSI,
    [string]$OutDir = "dist"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $RepoRoot

# ---- resolve git metadata ----
if (-not $Version) {
    $tag = & git describe --tags --exact-match 2>$null
    if ($LASTEXITCODE -eq 0) { $Version = $tag } else {
        $Version = "v0.9.0-dev"
    }
}
$ShortSha = (& git rev-parse --short HEAD) 2>$null; if (-not $ShortSha) { $ShortSha = "nogit" }
$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

Write-Host "fontcreator233 release build" -ForegroundColor Cyan
Write-Host "  version:   $Version"
Write-Host "  commit:    $ShortSha"
Write-Host "  built:     $BuildDate"
Write-Host "  go:        $(& go version)"

# ---- format + vet ----
Write-Host "`n[1/5] Format + Vet" -ForegroundColor Cyan
$needsFmt = & gofmt -l . 2>$null
if ($needsFmt) {
    Write-Host "  gofmt dirty:" -ForegroundColor Red; $needsFmt | ForEach-Object { Write-Host "    $_" }
    exit 1
}
& go vet ./...
if ($LASTEXITCODE -ne 0) { Write-Host "  go vet failed" -ForegroundColor Red; exit 1 }
Write-Host "  ok" -ForegroundColor Green

# ---- test ----
if (-not $SkipTest) {
    Write-Host "`n[2/5] Tests" -ForegroundColor Cyan
    & go test ./... -count=1
    if ($LASTEXITCODE -ne 0) { Write-Host "  tests failed" -ForegroundColor Red; exit 1 }
    Write-Host "  ok" -ForegroundColor Green
} else {
    Write-Host "`n[2/5] Tests — SKIP" -ForegroundColor Yellow
}

# ---- build ----
Write-Host "`n[3/5] Build" -ForegroundColor Cyan
$env:CGO_ENABLED = "0"
$Ldflags = "-s -w -X github.com/neko233-com/fontcreator233/internal/agentschema.Version=$Version -X github.com/neko233-com/fontcreator233/internal/agentschema.Commit=$ShortSha -X github.com/neko233-com/fontcreator233/internal/agentschema.BuildDate=$BuildDate"
& go build -ldflags $Ldflags -trimpath -o "fontcreator.exe" .
if ($LASTEXITCODE -ne 0) { Write-Host "  build failed" -ForegroundColor Red; exit 1 }
Write-Host "  ok — $((Get-Item fontcreator.exe).Length) bytes" -ForegroundColor Green

# ---- smoke ----
Write-Host "`n[4/5] Smoke" -ForegroundColor Cyan
& .\fontcreator.exe version
$v = & .\fontcreator.exe version --json | ConvertFrom-Json
if ($v.ok -ne $true) { Write-Host "  version smoke failed" -ForegroundColor Red; exit 1 }
& .\fontcreator.exe describe --json > $null
& .\fontcreator.exe init --dir "$env:TEMP\fc233-smoke" --template geometric --family "Smoke" --force > $null
& .\fontcreator.exe validate "$env:TEMP\fc233-smoke\font.recipe.yaml" --json > $null
Write-Host "  ok" -ForegroundColor Green

# ---- package ----
Write-Host "`n[5/5] Package → $OutDir" -ForegroundColor Cyan
$PkgName = "fontcreator233-$Version-windows-amd64"
$PkgDir = Join-Path $OutDir $PkgName
if (Test-Path $PkgDir) { Remove-Item $PkgDir -Recurse -Force }
New-Item -ItemType Directory -Path $PkgDir | Out-Null
Copy-Item fontcreator.exe $PkgDir\fontcreator.exe
Copy-Item LICENSE $PkgDir\ 2>$null
Copy-Item NOTICE.md $PkgDir\ 2>$null
Copy-Item README.md $PkgDir\ 2>$null

$ZipPath = Join-Path $OutDir "$PkgName.zip"
if (Test-Path $ZipPath) { Remove-Item $ZipPath -Force }
Compress-Archive -Path "$PkgDir\*" -DestinationPath $ZipPath

$Hash = (Get-FileHash $ZipPath -Algorithm SHA256).Hash.ToLower()
"$Hash  $PkgName.zip" | Out-File -Encoding ascii (Join-Path $OutDir "$PkgName.zip.sha256")

Write-Host "Done." -ForegroundColor Green
Write-Host "  $PkgName.zip — $([math]::Round((Get-Item $ZipPath).Length / 1MB, 2)) MB"
Write-Host "  SHA256: $Hash"
Write-Host ""
Write-Host "  Verify: Get-FileHash $PkgName.zip -Algorithm SHA256"

# ---- optional MSI ----
if ($WithMSI) {
    Write-Host "`n[6/6] MSI installer" -ForegroundColor Cyan
    $msi = Join-Path $OutDir "$PkgName.msi"
    & "$PSScriptRoot\build-msi.ps1" -Source $PkgDir -Version $Version -Arch x64 -Out $msi
    if ($LASTEXITCODE -ne 0) { exit 1 }
}
