$ErrorActionPreference = "Stop"

$REPO = "alucardeht/may-la-mcp"
$INSTALL_DIR = "$env:USERPROFILE\.mayla"
$MAYLA_CLI = "$INSTALL_DIR\mayla.exe"
$MAYLA_DAEMON = "$INSTALL_DIR\mayla-daemon.exe"
$VERSION_FILE = "$INSTALL_DIR\version"

if (-not (Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -ErrorAction Stop
        return $response.tag_name
    } catch {
        return $null
    }
}

function Get-Platform {
    $arch = if ([Environment]::Is64BitOperatingSystem) {
        if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
    } else { "amd64" }
    return "windows_$arch"
}

function Download-Binaries {
    param([string]$Version)

    $platform = Get-Platform

    $cliUrl = "https://github.com/$REPO/releases/download/$Version/mayla-$platform.exe"
    Write-Host "Downloading mayla CLI $Version for $platform..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $cliUrl -OutFile "$MAYLA_CLI.tmp" -ErrorAction Stop
    Move-Item -Path "$MAYLA_CLI.tmp" -Destination $MAYLA_CLI -Force

    $daemonUrl = "https://github.com/$REPO/releases/download/$Version/mayla-daemon-$platform.exe"
    Write-Host "Downloading mayla-daemon $Version for $platform..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $daemonUrl -OutFile "$MAYLA_DAEMON.tmp" -ErrorAction Stop
    Move-Item -Path "$MAYLA_DAEMON.tmp" -Destination $MAYLA_DAEMON -Force

    Set-Content -Path $VERSION_FILE -Value $Version
}

function Update-IfNeeded {
    $latest = Get-LatestVersion
    $current = if (Test-Path $VERSION_FILE) { Get-Content $VERSION_FILE } else { $null }

    if (-not $latest) {
        if ((Test-Path $MAYLA_CLI) -and (Test-Path $MAYLA_DAEMON)) { return }
        Write-Error "Cannot fetch latest version and no local binaries found"
        exit 1
    }

    if ($latest -ne $current -or -not (Test-Path $MAYLA_CLI) -or -not (Test-Path $MAYLA_DAEMON)) {
        Download-Binaries -Version $latest
    }
}

Update-IfNeeded
& $MAYLA_CLI @args
