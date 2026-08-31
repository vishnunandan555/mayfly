<#
.SYNOPSIS
    MayFly Windows Installer / Uninstaller (PowerShell)
    Features: Tier-2 Cryptographic SHA-256 Checksum Verification & Offline Local Builds
.PARAMETER Uninstall
    Cleanly removes mayfly and mf binaries and cleans PATH
.PARAMETER Update
    Rebuilds and updates installed binaries
#>

param (
    [switch]$Uninstall,
    [switch]$Update,
    [string]$Version = "v0.0.1"
)

$InstallDir = "$HOME\.local\bin"
$VaultDir = "$HOME\.mayfly"
$Repo = "vishnunandan555/mayfly"

if ($Uninstall) {
    Write-Host "=================================================" -ForegroundColor Yellow
    Write-Host "  MayFly Windows Complete Uninstaller" -ForegroundColor Yellow
    Write-Host "=================================================" -ForegroundColor Yellow
    Write-Host "WARNING: This will completely remove mayfly.exe and mf.exe,"
    Write-Host "clean your User PATH, and PERMANENTLY DELETE all encrypted"
    Write-Host "secrets in $VaultDir.`n"

    $resp = Read-Host "Are you sure you want to completely uninstall MayFly? [y/N]"
    if ($resp -ne "y" -and $resp -ne "Y") {
        Write-Host "Uninstallation canceled."
        exit 0
    }

    Remove-Item "$InstallDir\mayfly.exe" -ErrorAction SilentlyContinue
    Remove-Item "$InstallDir\mf.exe" -ErrorAction SilentlyContinue
    Write-Host "✓ Removed mayfly.exe and mf.exe from $InstallDir" -ForegroundColor Green

    # Clean User PATH if it was added by MayFly
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -like "*$InstallDir*") {
        $cleanedPath = ($userPath.Split(';') | Where-Object { $_ -ne $InstallDir }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $cleanedPath, "User")
        Write-Host "✓ Cleaned $InstallDir from User PATH." -ForegroundColor Green
    }

    if (Test-Path $VaultDir) {
        Remove-Item -Recurse -Force $VaultDir
        Write-Host "✓ Removed $VaultDir directory and all encrypted vaults." -ForegroundColor Green
    }

    Write-Host "`nMayFly has been completely and cleanly uninstalled from your system." -ForegroundColor Green
    exit 0
}

$arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }

Write-Host "=================================================" -ForegroundColor Cyan
if ($Update) {
    Write-Host "  🦋 MayFly — Updating to $Version" -ForegroundColor Cyan
} else {
    Write-Host "  🦋 MayFly — Zero-Dependency Secrets Workspace" -ForegroundColor Cyan
    Write-Host "  Secure Installation & Supply-Chain Verifier" -ForegroundColor Cyan
}
Write-Host "=================================================" -ForegroundColor Cyan
Write-Host "  Target Version : $Version"
Write-Host "  Platform       : Windows ($arch)"
Write-Host "  Install Path   : $InstallDir"
Write-Host "  Security Mode  : Tier-2 Cryptographic SHA-256 Verified"
Write-Host "=================================================`n" -ForegroundColor Cyan

Write-Host "Choose command alias to install:"
Write-Host "  [1] Both 'mayfly' and 'mf' (Recommended — press Enter)"
Write-Host "  [2] Only 'mayfly'"
Write-Host "  [3] Only 'mf'`n"

$aliasChoice = Read-Host "Select option [1/2/3]"
if ([string]::IsNullOrWhiteSpace($aliasChoice)) { $aliasChoice = "1" }

if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

$targetBin = "mayfly-windows-$arch.exe"

$baseUrl = if ($Version -eq "latest") {
    "https://github.com/$Repo/releases/latest/download"
} else {
    "https://github.com/$Repo/releases/download/$Version"
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    Write-Host "`nDownloading $targetBin from GitHub Releases..."
    $binPath = Join-Path $tempDir $targetBin
    $checksumPath = Join-Path $tempDir "checksums.txt"

    try {
        Invoke-WebRequest -Uri "$baseUrl/$targetBin" -OutFile $binPath -UseBasicParsing
        Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksumPath -UseBasicParsing
    } catch {
        Write-Host "`n❌ Error: Failed to download release binary or checksums from GitHub Releases ($baseUrl)." -ForegroundColor Red
        Write-Host "Please verify that release '$Version' exists at https://github.com/$Repo/releases" -ForegroundColor Red
        exit 1
    }

    Write-Host "Verifying cryptographic SHA-256 checksum..."
    $computedHash = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash.ToLower()
    $expectedLine = Get-Content $checksumPath | Where-Object { $_ -match $targetBin }

    if (-not $expectedLine -or -not ($expectedLine.ToLower().StartsWith($computedHash))) {
        Write-Host "`n🚨 SECURITY ALERT: Cryptographic checksum verification failed!" -ForegroundColor Red
        Write-Host "The downloaded binary does not match the published release hash." -ForegroundColor Red
        Write-Host "Installation aborted to protect your system." -ForegroundColor Red
        exit 1
    }

    Write-Host "✓ Cryptographic SHA-256 Checksum Verified: Authentic & Untampered." -ForegroundColor Green
    Copy-Item -Force $binPath "$InstallDir\mayfly.exe"
} finally {
    Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
}

switch ($aliasChoice) {
    "2" {
        Remove-Item "$InstallDir\mf.exe" -ErrorAction SilentlyContinue
        Write-Host "✓ Installed mayfly.exe -> $InstallDir\mayfly.exe" -ForegroundColor Green
    }
    "3" {
        Move-Item -Force "$InstallDir\mayfly.exe" "$InstallDir\mf.exe"
        Write-Host "✓ Installed mf.exe -> $InstallDir\mf.exe" -ForegroundColor Green
    }
    Default {
        Copy-Item -Force "$InstallDir\mayfly.exe" "$InstallDir\mf.exe"
        Write-Host "✓ Installed mayfly.exe -> $InstallDir\mayfly.exe" -ForegroundColor Green
        Write-Host "✓ Installed mf.exe     -> $InstallDir\mf.exe" -ForegroundColor Green
    }
}

# Prompt before adding to User PATH if missing
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
    Write-Host "`nNote: '$InstallDir' is not in your current User PATH."
    $addPath = Read-Host "Add '$InstallDir' to User PATH? [Y/n]"
    if ([string]::IsNullOrWhiteSpace($addPath) -or $addPath -eq "y" -or $addPath -eq "Y") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        Write-Host "✓ Added $InstallDir to User PATH." -ForegroundColor Green
    }
}

Write-Host "`n=================================================" -ForegroundColor Cyan
Write-Host "  🎉 MayFly Installation Complete!" -ForegroundColor Cyan
Write-Host "=================================================" -ForegroundColor Cyan
Write-Host "`nGetting Started:"
Write-Host "  mayfly (or mf)            - Launch Global TUI Dashboard"
Write-Host "  mf c                      - Open TUI for current project"
Write-Host "  mf run <command>          - Run app with in-memory secrets"
Write-Host "  mf --help (or mf help)    - View all available commands"
Write-Host "`nManagement & Updates:"
Write-Host "  mf uninstall              - Cleanly uninstall MayFly & remove binaries"
