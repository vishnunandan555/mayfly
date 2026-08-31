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
    [switch]$Fresh,
    [switch]$Reinstall,
    [string]$Version = "v0.0.4"
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

    $resp = "y"
    if ([Environment]::UserInteractive -and -not [Console]::IsInputRedirected) {
        $resp = Read-Host "Are you sure you want to completely uninstall MayFly? [y/N]"
    }
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
        $cleanedPath = ($userPath.Split(';') | Where-Object { $_ -ne $InstallDir -and $_ -ne "" }) -join ';'
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

# Check for existing installation
$hasPrev = (Test-Path "$InstallDir\mayfly.exe") -or (Test-Path "$InstallDir\mf.exe") -or (Test-Path $VaultDir)

if ($Fresh -or $Reinstall) {
    Write-Host "Removing previous vault and binaries..." -ForegroundColor Yellow
    Remove-Item "$InstallDir\mayfly.exe" -ErrorAction SilentlyContinue
    Remove-Item "$InstallDir\mf.exe" -ErrorAction SilentlyContinue
    if (Test-Path $VaultDir) {
        Remove-Item -Recurse -Force $VaultDir
    }
    Write-Host "✓ Wiped previous installation." -ForegroundColor Green
} elseif (-not $Update -and $hasPrev) {
    if ([Environment]::UserInteractive -and -not [Console]::IsInputRedirected) {
        Write-Host "=================================================" -ForegroundColor Yellow
        Write-Host "  Existing MayFly Installation Detected" -ForegroundColor Yellow
        Write-Host "=================================================" -ForegroundColor Yellow
        Write-Host "MayFly files were found on this machine:"
        if (Test-Path "$InstallDir\mayfly.exe") { Write-Host "  • Binary : $InstallDir\mayfly.exe" }
        if (Test-Path $VaultDir) { Write-Host "  • Vault  : $VaultDir" }
        Write-Host ""
        Write-Host "Choose an option:"
        Write-Host "  [1] Update / upgrade binaries (Keep existing vault secrets) [DEFAULT]"
        Write-Host "  [2] Remove and reinstall (Wipe ~/.mayfly vault and install fresh)"
        Write-Host "  [3] Cancel"
        Write-Host ""

        $prevChoice = "1"
        try {
            $inVal = Read-Host "Select option [1/2/3] (default: 1)"
            if (-not [string]::IsNullOrWhiteSpace($inVal)) { $prevChoice = $inVal }
        } catch {
            $prevChoice = "1"
        }

        switch ($prevChoice) {
            "1" {
                $Update = $true
            }
            "2" {
                Write-Host "`nWARNING: This will permanently delete all encrypted secrets in $VaultDir." -ForegroundColor Red
                $confirm = Read-Host "Are you sure you want to wipe ~/.mayfly and reinstall? [y/N]"
                if ($confirm -eq "y" -or $confirm -eq "Y") {
                    Remove-Item "$InstallDir\mayfly.exe" -ErrorAction SilentlyContinue
                    Remove-Item "$InstallDir\mf.exe" -ErrorAction SilentlyContinue
                    if (Test-Path $VaultDir) {
                        Remove-Item -Recurse -Force $VaultDir
                    }
                    Write-Host "✓ Wiped previous installation." -ForegroundColor Green
                } else {
                    Write-Host "Reinstallation canceled."
                    exit 0
                }
            }
            Default {
                Write-Host "Installation canceled."
                exit 0
            }
        }
    } else {
        $Update = $true
    }
}

# Accurate Windows Architecture Detection
$rawArch = $env:PROCESSOR_ARCHITECTURE
$arch = if ($rawArch -eq "ARM64") { "arm64" } else { "amd64" }


Write-Host "=================================================" -ForegroundColor Cyan
if ($Update) {
    Write-Host "  MayFly: Updating to $Version" -ForegroundColor Cyan
} else {
    Write-Host "  MayFly: Zero-Dependency Secrets Workspace" -ForegroundColor Cyan
    Write-Host "  Secure Windows Installation & Supply-Chain Verifier" -ForegroundColor Cyan
}
Write-Host "=================================================" -ForegroundColor Cyan
Write-Host "  Target Version : $Version"
Write-Host "  Platform       : Windows ($arch)"
Write-Host "  Install Path   : $InstallDir"
Write-Host "  Security Mode  : Tier-2 Cryptographic SHA-256 Verified"
Write-Host "=================================================`n" -ForegroundColor Cyan

# Configure command aliases (preserve on update, prompt on fresh install)
$aliasChoice = "1"
if ($Update) {
    if ((Test-Path "$InstallDir\mayfly.exe") -and (Test-Path "$InstallDir\mf.exe")) {
        $aliasChoice = "1"
    } elseif (Test-Path "$InstallDir\mayfly.exe") {
        $aliasChoice = "2"
    } elseif (Test-Path "$InstallDir\mf.exe") {
        $aliasChoice = "3"
    } else {
        $aliasChoice = "1"
    }
} elseif ([Environment]::UserInteractive -and -not [Console]::IsInputRedirected) {
    Write-Host "Choose command alias to install:"
    Write-Host "  [1] Both 'mayfly' and 'mf' (Recommended: press Enter)"
    Write-Host "  [2] Only 'mayfly'"
    Write-Host "  [3] Only 'mf'`n"
    try {
        $inputVal = Read-Host "Select option [1/2/3]"
        if (-not [string]::IsNullOrWhiteSpace($inputVal)) { $aliasChoice = $inputVal }
    } catch {
        $aliasChoice = "1"
    }
}


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

$installed = $false

try {
    Write-Host "`nDownloading $targetBin from GitHub Releases..."
    $binPath = Join-Path $tempDir $targetBin
    $checksumPath = Join-Path $tempDir "checksums.txt"

    try {
        Invoke-WebRequest -Uri "$baseUrl/$targetBin" -OutFile $binPath -UseBasicParsing
        Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksumPath -UseBasicParsing

        Write-Host "Verifying cryptographic SHA-256 checksum..."
        $computedHash = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash.ToLower()
        $expectedLine = Get-Content $checksumPath | Where-Object { $_ -match $targetBin }

        if (-not $expectedLine -or -not ($expectedLine.ToLower().StartsWith($computedHash))) {
            Write-Host "`n[SECURITY ALERT]: Cryptographic checksum verification failed!" -ForegroundColor Red
            Write-Host "The downloaded binary does not match the published release hash." -ForegroundColor Red
            Write-Host "Installation aborted to protect your system." -ForegroundColor Red
            exit 1
        }

        Write-Host "[OK] Cryptographic SHA-256 Checksum Verified: Authentic & Untampered." -ForegroundColor Green
        Copy-Item -Force $binPath "$InstallDir\mayfly.exe"
        $installed = $true
    } catch {
        # Fallback to local source build if repository is present
        $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
        if (Test-Path "$scriptDir\cmd\mayfly\main.go") {
            Write-Host "Note: Remote release download unavailable; building locally from Go source..." -ForegroundColor Yellow
            Push-Location $scriptDir
            try {
                go build -o "$InstallDir\mayfly.exe" .\cmd\mayfly
                $installed = $true
            } finally {
                Pop-Location
            }
        } else {
            Write-Host "`nError: Failed to download release binary or checksums from GitHub Releases ($baseUrl)." -ForegroundColor Red
            Write-Host "Please verify that release '$Version' exists at https://github.com/$Repo/releases" -ForegroundColor Red
            exit 1
        }
    }
} finally {
    Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
}

if ($installed) {
    switch ($aliasChoice) {
        "2" {
            Remove-Item "$InstallDir\mf.exe" -ErrorAction SilentlyContinue
            Write-Host "[OK] Installed mayfly.exe -> $InstallDir\mayfly.exe" -ForegroundColor Green
        }
        "3" {
            Move-Item -Force "$InstallDir\mayfly.exe" "$InstallDir\mf.exe"
            Write-Host "[OK] Installed mf.exe -> $InstallDir\mf.exe" -ForegroundColor Green
        }
        Default {
            Copy-Item -Force "$InstallDir\mayfly.exe" "$InstallDir\mf.exe"
            Write-Host "[OK] Installed mayfly.exe -> $InstallDir\mayfly.exe" -ForegroundColor Green
            Write-Host "[OK] Installed mf.exe     -> $InstallDir\mf.exe" -ForegroundColor Green
        }
    }

    # Automatically ensure User PATH is configured
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path += ";$InstallDir"
        Write-Host "[OK] Added $InstallDir to User PATH." -ForegroundColor Green
    }
}

Write-Host "`n=================================================" -ForegroundColor Cyan
Write-Host "  MayFly Installation Complete" -ForegroundColor Cyan
Write-Host "=================================================" -ForegroundColor Cyan
Write-Host "`nGetting Started:"
Write-Host "  mayfly (or mf)            - Launch Global TUI Dashboard"
Write-Host "  mf c                      - Open TUI for current project"
Write-Host "  mf run <command>          - Run app with in-memory secrets"
Write-Host "  mf --help (or mf help)    - View all available commands"
Write-Host "`nManagement & Updates:"
Write-Host "  mf uninstall              - Cleanly uninstall MayFly & remove binaries"
