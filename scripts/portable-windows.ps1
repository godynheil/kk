<#
PowerShell script to build a Windows portable bundle (kk-portable.exe)
Usage: run from repository root:
  powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\portable-windows.ps1
Or just: .\scripts\portable-windows.ps1 (if execution policy allows)

This script mirrors the Makefile 'portable' target but uses PowerShell-native commands so it works on plain Windows.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Determine repository root (script is located at scripts/portable-windows.ps1)
$scriptPath = $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent (Split-Path -Parent $scriptPath)
Push-Location $repoRoot

try {
    Write-Host "Repository root: $repoRoot"

    # Compute BUILD_VERSION like the Makefile
    $rawBranch = & git -c "safe.directory=$PWD" rev-parse --abbrev-ref HEAD 2>$null
    if ($LASTEXITCODE -ne 0 -or -not $rawBranch) { $buildVersion = 'dev' } else { $buildVersion = $rawBranch -replace '/', '-' }
    Write-Host "Using BuildVersion: $buildVersion"

    # Run goversioninfo if available
    if (Get-Command goversioninfo -ErrorAction SilentlyContinue) {
        Write-Host "Running goversioninfo..."
        Push-Location cmd/kk
        goversioninfo -o resource.syso versioninfo.json
        Pop-Location
    } else {
        Write-Host "goversioninfo not found; skipping resource.syso generation"
    }

    # Load credentials from .env (key=value lines, comments ignored)
    function Get-DotEnvValue([string]$Key) {
        $envFile = Join-Path $repoRoot ".env"
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
            Write-Host "Skipping code signing; set KK_SIGN_CERT_PATH or KK_SIGN_CERT_SHA1 to sign."
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

        Write-Host "Signing $Path"
        & $signtool @signtoolArgs
        if ($LASTEXITCODE -ne 0) { throw "signtool failed" }
    }

    $googleClientId     = Get-BuildSetting "KK_GOOGLE_CLIENT_ID"
    $googleClientSecret = Get-BuildSetting "KK_GOOGLE_CLIENT_SECRET"
    if (-not $googleClientId)     { Write-Warning ".env: KK_GOOGLE_CLIENT_ID not set — OAuth will need env vars at runtime" }
    if (-not $googleClientSecret) { Write-Warning ".env: KK_GOOGLE_CLIENT_SECRET not set — OAuth will need env vars at runtime" }

    $distDir = Join-Path $repoRoot 'dist/kk-portable'
    if (-Not (Test-Path $distDir)) { New-Item -ItemType Directory -Path $distDir | Out-Null }

    # Build Windows executable
    Write-Host "Building kk-portable.exe (windows/amd64)..."
    $PKG = "github.com/godynheil/kk/internal/app"
    $ldflags = "-X $PKG.BuildVersion=$buildVersion -X `"$PKG.DefaultGoogleOAuthClientID=$googleClientId`" -X `"$PKG.DefaultGoogleOAuthClientSecret=$googleClientSecret`""
    # Set environment vars only for the build process
    & {
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        go build -o "$distDir\kk-portable.exe" -ldflags $ldflags ./cmd/kk
    }
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    Invoke-SignExe "$distDir\kk-portable.exe"

    # Copy optional thirdparty assets
    $portableGitSrc = Join-Path $repoRoot 'thirdparty/PortableGit'
    if (Test-Path $portableGitSrc) {
        Write-Host "Copying PortableGit..."
        $dest = Join-Path $distDir 'PortableGit'
        Remove-Item -Recurse -Force $dest -ErrorAction SilentlyContinue
        Copy-Item -Recurse -Force $portableGitSrc $dest
    } else {
        Write-Host "Note: thirdparty/PortableGit not found; place PortableGit under thirdparty/ to include it."
    }

    $rcloneSrc = Join-Path $repoRoot 'thirdparty/rclone'
    if (Test-Path $rcloneSrc) {
        Write-Host "Copying rclone..."
        $dest = Join-Path $distDir 'rclone'
        Remove-Item -Recurse -Force $dest -ErrorAction SilentlyContinue
        Copy-Item -Recurse -Force $rcloneSrc $dest
    } else {
        Write-Host "Note: thirdparty/rclone not found; place rclone under thirdparty/ to include it."
    }

    # Copy README and docs if present (best-effort)
    if (Test-Path (Join-Path $repoRoot 'README.md')) {
        Copy-Item -Force README.md $distDir
    }
    if (Test-Path (Join-Path $repoRoot 'docs')) {
        $docsDest = Join-Path $distDir 'docs'
        Remove-Item -Recurse -Force $docsDest -ErrorAction SilentlyContinue
        Copy-Item -Recurse -Force docs $docsDest
    }

    # Create zip archive using Compress-Archive
    $zipPath = Join-Path $repoRoot 'dist/kk-portable.zip'
    if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
    Write-Host "Creating $zipPath..."
    Compress-Archive -Path $distDir -DestinationPath $zipPath -Force

    Write-Host "Portable build complete. Output: $distDir and $zipPath"
} finally {
    Pop-Location
}

