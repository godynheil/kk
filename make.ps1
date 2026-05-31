#!/usr/bin/env pwsh
# make.ps1 - Windows PowerShell equivalent of the Makefile
# Usage: .\make.ps1 [target]
# Targets: build, build-all, test, smoke, fmt, lint, clean, portable, portable-windows, vuln
# powershell -NoProfile -ExecutionPolicy Bypass -File ./make.ps1 build

param(
    [Parameter(Position=0)]
    [string]$Target = "build"
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

# Compute BUILD_VERSION from current git branch (mirrors Makefile logic)
function Get-BuildVersion {
    try {
        $branch = & git -c "safe.directory=$PSScriptRoot" rev-parse --abbrev-ref HEAD 2>$null
        if ($LASTEXITCODE -eq 0 -and $branch) {
            return $branch -replace '/', '-'
        }
    } catch {}
    return "dev"
}

$BUILD_VERSION    = Get-BuildVersion
$BUILD_DATE       = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$BUILD_DATE_LOCAL = (Get-Date).ToString("yyyy-MM-ddTHH:mm:sszzz")
Write-Host "Build version: $BUILD_VERSION"    -ForegroundColor Cyan
Write-Host "Build date   : $BUILD_DATE | $BUILD_DATE_LOCAL" -ForegroundColor Cyan

# Load credentials from .env (key=value lines, comments ignored)
function Get-DotEnvValue([string]$Key) {
    $envFile = Join-Path $PSScriptRoot ".env"
    if (-not (Test-Path $envFile)) { return "" }
    foreach ($line in Get-Content $envFile) {
        if ($line -match "^\s*$Key\s*=\s*(.+)$") { return $Matches[1].Trim() }
    }
    return ""
}

function Get-BuildSetting([string]$Key) {
    $value = [Environment]::GetEnvironmentVariable($Key)
    if ($value) { return $value }
    return Get-DotEnvValue $Key
}

$GOOGLE_CLIENT_ID     = Get-BuildSetting "KK_GOOGLE_CLIENT_ID"
$GOOGLE_CLIENT_SECRET = Get-BuildSetting "KK_GOOGLE_CLIENT_SECRET"
if (-not $GOOGLE_CLIENT_ID)     { Write-Warning ".env: KK_GOOGLE_CLIENT_ID not set — OAuth will need env vars at runtime" }
if (-not $GOOGLE_CLIENT_SECRET) { Write-Warning ".env: KK_GOOGLE_CLIENT_SECRET not set — OAuth will need env vars at runtime" }

$PKG = "github.com/godynheil/kk/internal/app"
$GOPATH = & go env GOPATH
function Resolve-GoBin([string]$name) {
    if (Get-Command $name -ErrorAction SilentlyContinue) { return $name }
    $p = Join-Path $GOPATH "bin\$name.exe"
    if (Test-Path $p) { return $p }
    return $null
}

function Get-LDFlags {
    return "-X $PKG.BuildVersion=$BUILD_VERSION -X `"$PKG.BuildDate=$BUILD_DATE`" -X `"$PKG.BuildDateLocal=$BUILD_DATE_LOCAL`" -X `"$PKG.DefaultGoogleOAuthClientID=$GOOGLE_CLIENT_ID`" -X `"$PKG.DefaultGoogleOAuthClientSecret=$GOOGLE_CLIENT_SECRET`""
}

function Resolve-SignTool {
    $configured = Get-BuildSetting "KK_SIGNTOOL_PATH"
    if ($configured) {
        if (Test-Path $configured) { return $configured }
        throw "KK_SIGNTOOL_PATH does not exist: $configured"
    }
    $cmd = Get-Command "signtool.exe" -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $cmd = Get-Command "signtool" -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

function Invoke-SignExe([string]$Path) {
    $certPath = Get-BuildSetting "KK_SIGN_CERT_PATH"
    $certPassword = Get-BuildSetting "KK_SIGN_CERT_PASSWORD"
    $certSha1 = Get-BuildSetting "KK_SIGN_CERT_SHA1"
    if (-not $certPath -and -not $certSha1) {
        Write-Host "Skipping code signing; set KK_SIGN_CERT_PATH or KK_SIGN_CERT_SHA1 to sign." -ForegroundColor DarkYellow
        return
    }
    if (-not (Test-Path $Path)) { throw "Cannot sign missing executable: $Path" }

    $signtool = Resolve-SignTool
    if (-not $signtool) { throw "signtool was not found. Install the Windows SDK or set KK_SIGNTOOL_PATH." }

    $digestAlg = Get-BuildSetting "KK_SIGN_DIGEST_ALG"
    if (-not $digestAlg) { $digestAlg = "SHA256" }
    $timestampUrl = Get-BuildSetting "KK_SIGN_TIMESTAMP_URL"
    if (-not $timestampUrl) { $timestampUrl = "http://timestamp.digicert.com" }

    $signtoolArgs = @("sign", "/fd", $digestAlg, "/tr", $timestampUrl, "/td", $digestAlg)
    if ($certPath) {
        if (-not (Test-Path $certPath)) { throw "KK_SIGN_CERT_PATH does not exist: $certPath" }
        $signtoolArgs += @("/f", $certPath)
        if ($certPassword) { $signtoolArgs += @("/p", $certPassword) }
    } else {
        $certSha1 = $certSha1 -replace "\s", ""
        $signtoolArgs += @("/sha1", $certSha1)
        if ((Get-BuildSetting "KK_SIGN_MACHINE_STORE") -match "^(1|true|yes)$") { $signtoolArgs += "/sm" }
    }
    $signtoolArgs += $Path

    Write-Host "Signing $Path" -ForegroundColor Yellow
    & $signtool @signtoolArgs
    if ($LASTEXITCODE -ne 0) { throw "signtool failed" }
}

function Invoke-Lint {
    Write-Host "==> lint" -ForegroundColor Yellow
    $bin = Resolve-GoBin "golangci-lint"
    if (-not $bin) {
        Write-Warning "golangci-lint not found in PATH or GOPATH/bin. Skipping lint."
        return
    }
    & $bin run ./...
    if ($LASTEXITCODE -ne 0) { throw "lint failed" }
}

function Invoke-Build {
    Invoke-Lint
    Write-Host "==> build" -ForegroundColor Yellow
    Push-Location "cmd/kk"
    $govi = Resolve-GoBin "goversioninfo"
    if (-not $govi) {
        Write-Warning "goversioninfo not found in PATH or GOPATH/bin. Skipping versioninfo resource generation."
    } else {
        & $govi -o resource.syso versioninfo.json
        if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed" }
    }
    Pop-Location
    & go build -buildvcs=false -ldflags (Get-LDFlags) ./cmd/kk
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    Invoke-SignExe "kk.exe"
}

function Invoke-Portable {
    Invoke-Lint
    Write-Host "==> portable (Windows)" -ForegroundColor Yellow
    Push-Location "cmd/kk"
    $govi = Resolve-GoBin "goversioninfo"
    if (-not $govi) {
        Write-Warning "goversioninfo not found in PATH or GOPATH/bin. Skipping versioninfo resource generation."
    } else {
        & $govi -o resource.syso versioninfo.json
        if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed" }
    }
    Pop-Location
    New-Item -ItemType Directory -Force "dist/kk-portable" | Out-Null
    $env:GOOS = "windows"; $env:GOARCH = "amd64"
    & go build -buildvcs=false -o "dist/kk-portable/kk-portable.exe" -ldflags (Get-LDFlags) ./cmd/kk
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    Invoke-SignExe "dist/kk-portable/kk-portable.exe"
    if (Test-Path "thirdparty/PortableGit") {
        Copy-Item -Recurse -Force "thirdparty/PortableGit" "dist/kk-portable/"
    } else {
        Write-Host "Note: thirdparty/PortableGit not found; place PortableGit under thirdparty/ to include it."
    }
    if (Test-Path "thirdparty/rclone") {
        Copy-Item -Recurse -Force "thirdparty/rclone" "dist/kk-portable/"
    } else {
        Write-Host "Note: thirdparty/rclone not found; place rclone under thirdparty/ to include it."
    }
    Copy-Item -Force "README.md" "dist/kk-portable/" -ErrorAction SilentlyContinue
    Copy-Item -Recurse -Force "docs" "dist/kk-portable/" -ErrorAction SilentlyContinue
    # Create zip if Compress-Archive is available
    try {
        Compress-Archive -Path "dist/kk-portable" -DestinationPath "dist/kk-portable.zip" -Force
        Write-Host "Created dist/kk-portable.zip"
    } catch {
        Write-Host "Note: could not create zip archive: $_"
    }
}

function Invoke-PortableWindows {
    Invoke-Lint
    Write-Host "==> portable-windows" -ForegroundColor Yellow
    & powershell -NoProfile -ExecutionPolicy Bypass -File "scripts/portable-windows.ps1"
    if ($LASTEXITCODE -ne 0) { throw "portable-windows script failed" }
}

function New-Sha256Sums([string[]]$Paths, [string]$OutputPath) {
    Write-Host "==> checksums" -ForegroundColor Yellow
    $lines = foreach ($path in $Paths) {
        if (-not (Test-Path $path)) { throw "Cannot checksum missing file: $path" }
        $hash = (Get-FileHash -Algorithm SHA256 $path).Hash.ToLowerInvariant()
        "$hash  $(Split-Path -Leaf $path)"
    }
    $lines | Set-Content -Encoding ASCII $OutputPath
    Write-Host "Created $OutputPath"
}

function New-GitHubReleaseMarkdown([string[]]$AssetPaths, [string]$ChecksumPath, [string]$OutputPath) {
    Write-Host "==> GitHub release markdown" -ForegroundColor Yellow
    $assetList = ($AssetPaths | ForEach-Object { "- ``$(Split-Path -Leaf $_)``" }) -join "`r`n"
    $checksumBlock = (Get-Content $ChecksumPath) -join "`r`n"
    $content = @"
## KK $BUILD_VERSION

### Downloads

$assetList
- ``$(Split-Path -Leaf $ChecksumPath)``

### Windows

Download ``kk.zip`` (or ``kk.exe``) for the standalone CLI, or download ``kk-portable.zip`` for the portable bundle with bundled optional tools when available.

This release may be unsigned unless a code-signing certificate was configured during the build. Windows may show an "Unknown Publisher" warning for unsigned builds.

### Verify SHA256 checksums

PowerShell:

~~~powershell
Get-FileHash -Algorithm SHA256 .\kk.exe
Get-FileHash -Algorithm SHA256 .\kk.zip
Get-FileHash -Algorithm SHA256 .\kk-portable.zip
Get-Content .\SHA256SUMS.txt
~~~

Expected checksums:

~~~text
$checksumBlock
~~~

### Notes

- Checksums help users verify that downloaded files match this release.
- Checksums are not a replacement for code signing.
- Windows will still not show a trusted publisher unless the executable is signed with a trusted code-signing certificate.
"@
    $content | Set-Content -Encoding UTF8 $OutputPath
    Write-Host "Created $OutputPath"
}

function Invoke-BuildAll {
    Write-Host "==> build-all" -ForegroundColor Yellow
    Invoke-Build
    Invoke-Portable

    New-Item -ItemType Directory -Force "dist" | Out-Null
    Copy-Item -Force "kk.exe" "dist/kk.exe"

    try {
        Compress-Archive -Path "dist/kk.exe" -DestinationPath "dist/kk.zip" -Force
        Write-Host "Created dist/kk.zip"
    } catch {
        Write-Host "Note: could not create zip archive: $_"
    }

    $assets = @("dist/kk.exe", "dist/kk.zip", "dist/kk-portable.zip")
    New-Sha256Sums $assets "dist/SHA256SUMS.txt"
    New-GitHubReleaseMarkdown $assets "dist/SHA256SUMS.txt" "dist/GITHUB_RELEASE.md"

    Write-Host "Release assets:" -ForegroundColor Cyan
    Write-Host "  dist/kk.exe"
    Write-Host "  dist/kk.zip"
    Write-Host "  dist/kk-portable.zip"
    Write-Host "  dist/SHA256SUMS.txt"
    Write-Host "  dist/GITHUB_RELEASE.md"
}

function Invoke-Test {
    Write-Host "==> test" -ForegroundColor Yellow
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "tests failed" }
}

function Invoke-Smoke {
    Write-Host "==> smoke" -ForegroundColor Yellow
    & bash "./scripts/smoke-test.sh"
    if ($LASTEXITCODE -ne 0) { throw "smoke tests failed" }
}

function Invoke-Fmt {
    Write-Host "==> fmt" -ForegroundColor Yellow
    & gofmt -w ./cmd ./internal
    if ($LASTEXITCODE -ne 0) { throw "gofmt failed" }
}

function Invoke-Clean {
    Write-Host "==> clean" -ForegroundColor Yellow
    Remove-Item -Force -ErrorAction SilentlyContinue "kk", "kk.exe", "cmd/kk/resource.syso"
}

function Invoke-Vuln {
    Write-Host "==> vuln" -ForegroundColor Yellow

    Write-Host "Installing govulncheck..." -ForegroundColor Cyan
    & go install golang.org/x/vuln/cmd/govulncheck@latest
    if ($LASTEXITCODE -ne 0) { throw "go install govulncheck failed" }

    Write-Host "Installing gosec..." -ForegroundColor Cyan
    & go install github.com/securego/gosec/v2/cmd/gosec@latest
    if ($LASTEXITCODE -ne 0) { throw "go install gosec failed" }

    Write-Host "Running govulncheck..." -ForegroundColor Cyan
    $govuln = Resolve-GoBin "govulncheck"
    if (-not $govuln) { $govuln = "govulncheck" }
    & $govuln ./...
    if ($LASTEXITCODE -ne 0) { throw "govulncheck found vulnerabilities" }

    Write-Host "Running gosec..." -ForegroundColor Cyan
    $gosec = Resolve-GoBin "gosec"
    if (-not $gosec) { $gosec = "gosec" }
    & $gosec ./...
    if ($LASTEXITCODE -ne 0) { throw "gosec found security issues" }
}

switch ($Target) {
    "build"            { Invoke-Build }
    "build-all"        { Invoke-BuildAll }
    "portable"         { Invoke-Portable }
    "portable-windows" { Invoke-PortableWindows }
    "test"             { Invoke-Test }
    "smoke"            { Invoke-Smoke }
    "fmt"              { Invoke-Fmt }
    "lint"             { Invoke-Lint }
    "clean"            { Invoke-Clean }
    "vuln"             { Invoke-Vuln }
    default {
        Write-Error "Unknown target: $Target`nAvailable targets: build, build-all, portable, portable-windows, test, smoke, fmt, lint, clean, vuln"
        exit 1
    }
}

