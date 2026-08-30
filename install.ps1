<#
.SYNOPSIS
    MayFly Windows Installer / Uninstaller (PowerShell)
.PARAMETER Uninstall
    Cleanly removes mayfly and mf binaries and cleans PATH
.PARAMETER Update
    Rebuilds and updates installed binaries
#>

param (
    [switch]$Uninstall,
    [switch]$Update
)

$InstallDir = "$HOME\.local\bin"
$VaultDir = "$HOME\.mayfly"

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

Write-Host "=================================================" -ForegroundColor Cyan
if ($Update) {
    Write-Host "  Updating MayFly..." -ForegroundColor Cyan
} else {
    Write-Host "  Installing MayFly (Zero-Dependency Secrets)..." -ForegroundColor Cyan
}
Write-Host "=================================================" -ForegroundColor Cyan

if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

Write-Host "Building mayfly.exe binary..."
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o "$InstallDir\mayfly.exe" .\cmd\mayfly

# Copy to mf.exe
Copy-Item -Force "$InstallDir\mayfly.exe" "$InstallDir\mf.exe"

Write-Host "✓ Installed mayfly.exe -> $InstallDir\mayfly.exe" -ForegroundColor Green
Write-Host "✓ Installed mf.exe     -> $InstallDir\mf.exe" -ForegroundColor Green

# Add to user PATH if missing
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
    Write-Host "✓ Added $InstallDir to User PATH." -ForegroundColor Green
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
Write-Host "  irm https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.ps1 | iex -args -Update"
Write-Host "                            - Update to latest version"

