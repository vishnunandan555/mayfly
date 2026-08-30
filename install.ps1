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
    Write-Host "Uninstalling MayFly..." -ForegroundColor Yellow
    Remove-Item "$InstallDir\mayfly.exe" -ErrorAction SilentlyContinue
    Remove-Item "$InstallDir\mf.exe" -ErrorAction SilentlyContinue
    Write-Host "✓ Removed mayfly.exe and mf.exe binaries from $InstallDir" -ForegroundColor Green

    # Clean User PATH if it was added by MayFly
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -like "*$InstallDir*") {
        $cleanedPath = ($userPath.Split(';') | Where-Object { $_ -ne $InstallDir }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $cleanedPath, "User")
        Write-Host "✓ Cleaned $InstallDir from User PATH." -ForegroundColor Green
    }

    if (Test-Path $VaultDir) {
        $resp = Read-Host "Do you want to permanently delete encrypted secrets in $VaultDir? (y/N)"
        if ($resp -eq "y" -or $resp -eq "Y") {
            Remove-Item -Recurse -Force $VaultDir
            Write-Host "✓ Removed $VaultDir directory." -ForegroundColor Green
        }
    }
    Write-Host "MayFly has been completely and cleanly uninstalled from your system." -ForegroundColor Green
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

Write-Host "`nInstallation complete! Run 'mayfly' or 'mf' to start." -ForegroundColor Green
