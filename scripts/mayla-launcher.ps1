$ErrorActionPreference = "Stop"

$REPO = "alucardeht/may-la-mcp"
$INSTALL_DIR = "$env:USERPROFILE\.mayla"
$BINARY = "$INSTALL_DIR\mayla-daemon.exe"
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
    return "windows-$arch"
}

function Download-Binary {
    param([string]$Version)

    $platform = Get-Platform
    $url = "https://github.com/$REPO/releases/download/$Version/mayla-daemon-$platform.exe"

    Write-Host "Downloading May-la $Version for $platform..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $url -OutFile "$BINARY.tmp" -ErrorAction Stop
    Move-Item -Path "$BINARY.tmp" -Destination $BINARY -Force
    Set-Content -Path $VERSION_FILE -Value $Version
}

function Update-IfNeeded {
    $latest = Get-LatestVersion
    $current = if (Test-Path $VERSION_FILE) { Get-Content $VERSION_FILE } else { $null }

    if (-not $latest) {
        if (Test-Path $BINARY) { return }
        Write-Error "Cannot fetch latest version and no local binary found"
        exit 1
    }

    if ($latest -ne $current -or -not (Test-Path $BINARY)) {
        Download-Binary -Version $latest
    }
}

Update-IfNeeded
& $BINARY @args
