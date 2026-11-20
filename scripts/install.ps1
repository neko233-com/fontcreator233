# fontcreator233 Windows one-click installer
# Usage (run as admin recommended):
#   powershell -ExecutionPolicy Bypass -Command "irm https://.../install.ps1 | iex"
#   powershell -File install.ps1 -Version v0.9.0 -InstallDir C:\tools\fontcreator233 -UseMSI
param(
    [string]$Version,                         # e.g. "v0.9.0"  — defaults to latest release
    [string]$InstallDir = "$env:USERPROFILE\.local\bin\fontcreator233",
    [switch]$UseMSI,                          # prefer .msi installer over .zip
    [string[]]$Mirrors = @(                   # tried in order, first reachable wins
        "https://github.com",
        "https://ghproxy.net/https://github.com",
        "https://mirror.ghproxy.com/https://github.com",
        "https://ghfast.top/https://github.com"
    )
)
$ErrorActionPreference = "Stop"
$Repo = "neko233-com/fontcreator233"

function Test-Reachable($url) {
    try {
        $r = Invoke-WebRequest -Uri $url -Method Head -TimeoutSec 10 -UseBasicParsing
        return ($r.StatusCode -eq 200) -or ($r.StatusCode -eq 302)
    } catch { return $false }
}

function Invoke-Download($url, $out) {
    $tmp = $out + ".part"
    Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    Move-Item $tmp $out -Force
}

function Invoke-JSON($url) {
    (Invoke-WebRequest -Uri $url -UseBasicParsing).Content | ConvertFrom-Json
}

Write-Host "fontcreator233 installer" -ForegroundColor Cyan

# --- resolve version ---
if (-not $Version) {
    Write-Host "  resolving latest tag ..."
    $tag = $null
    foreach ($m in $Mirrors) {
        try {
            $apiUrl = "$m/api.github.com/repos/$Repo/releases/latest"
            $rl = Invoke-JSON $apiUrl
            $tag = $rl.tag_name
            Write-Host "    latest: $tag (via $m)"
            break
        } catch { }
    }
    if (-not $tag) { throw "cannot resolve latest release; check your network or specify -Version" }
    $Version = $tag
}
Write-Host "  version:   $Version"

# --- pick asset ---
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "x86" }
$archOut = if ($arch -eq "amd64") { "windows-amd64" } else { "windows-x86" }
$archive = "fontcreator233-$Version-$archOut"

if ($UseMSI) {
    Write-Host "  mode:      MSI installer"
    $assetFile = "$archive.msi"
    $ext = ".msi"
} else {
    Write-Host "  mode:      portable zip"
    $assetFile = "$archive.zip"
    $ext = ".zip"
}

# --- find reachable mirror ---
Write-Host "  probe mirrors ..."
$base = $null
foreach ($m in $Mirrors) {
    $u = "$m/$Repo/releases/download/$Version/$assetFile"
    if (Test-Reachable $u) {
        $base = $m; break
    }
}
if (-not $base) { throw "no reachable mirror for $assetFile" }
Write-Host "    using $base"

# --- download ---
$tmp = Join-Path $env:TEMP "fc233-install-$Version"
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
$dl = Join-Path $tmp $assetFile
$url = "$base/$Repo/releases/download/$Version/$assetFile"
Write-Host "  downloading $assetFile ..."
Invoke-Download $url $dl
Write-Host "    downloaded $(Get-Item $dl).Length bytes"

# --- verify checksum ---
$shaUrl = "$url.sha256"
try {
    $shaResp = Invoke-WebRequest -Uri $shaUrl -UseBasicParsing -TimeoutSec 10
    $expected = ($shaResp.Content -split ' ')[0].Trim().ToLower()
    $actual = (Get-FileHash $dl -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) {
        Write-Host "  SHA256 mismatch! expected=$expected actual=$actual" -ForegroundColor Yellow
        Write-Host "  continuing anyway (disable with -UseMSI may help)" -ForegroundColor Yellow
    } else {
        Write-Host "  sha256 OK"
    }
} catch { Write-Host "  no checksum file available (ok)" }

# --- install ---
if ($UseMSI) {
    Write-Host "  running MSI installer ..."
    Start-Process msiexec.exe -ArgumentList "/i `"$dl`" /qn" -Wait
    Write-Host "  done. fontcreator.exe is now in Program Files; refresh PATH or open a new shell." -ForegroundColor Green
} else {
    # portable zip
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "  extracting to $InstallDir ..."
    Expand-Archive -Path $dl -DestinationPath (Join-Path $InstallDir "_new") -Force
    # move files up one level (zip wraps in fontcreator233-XXX/)
    Get-ChildItem (Join-Path $InstallDir "_new") | ForEach-Object {
        if ($_.PSIsContainer -and $_.Name -like "fontcreator233-*") {
            Get-ChildItem $_.FullName | ForEach-Object {
                Move-Item $_.FullName $InstallDir -Force
            }
        }
    }
    Remove-Item (Join-Path $InstallDir "_new") -Recurse -Force

    # add to PATH if not present
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$userPath;$InstallDir", "User")
        Write-Host "  added $InstallDir to user PATH"
    }
    Write-Host "  done. Open a new shell and run: fontcreator version" -ForegroundColor Green
}
