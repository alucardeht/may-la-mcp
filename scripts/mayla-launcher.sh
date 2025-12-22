#!/bin/bash
set -e

REPO="alucardeht/may-la-mcp"
INSTALL_DIR="$HOME/.mayla"
MAYLA_CLI="$INSTALL_DIR/mayla"
MAYLA_DAEMON="$INSTALL_DIR/mayla-daemon"
VERSION_FILE="$INSTALL_DIR/version"

mkdir -p "$INSTALL_DIR"

get_latest_version() {
    curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' 2>/dev/null || echo ""
}

get_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
    esac

    echo "${os}_${arch}"
}

download_binary() {
    local version=$1
    local platform=$(get_platform)

    local mayla_url="https://github.com/$REPO/releases/download/$version/mayla-$platform"
    echo "Downloading mayla $version for $platform..." >&2
    curl -sL "$mayla_url" -o "$MAYLA_CLI.tmp" && mv "$MAYLA_CLI.tmp" "$MAYLA_CLI" && chmod +x "$MAYLA_CLI"

    local daemon_url="https://github.com/$REPO/releases/download/$version/mayla-daemon-$platform"
    echo "Downloading mayla-daemon $version for $platform..." >&2
    curl -sL "$daemon_url" -o "$MAYLA_DAEMON.tmp" && mv "$MAYLA_DAEMON.tmp" "$MAYLA_DAEMON" && chmod +x "$MAYLA_DAEMON"

    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "Removing macOS quarantine attributes..." >&2
        xattr -d com.apple.quarantine "$MAYLA_CLI" 2>/dev/null || true
        xattr -d com.apple.quarantine "$MAYLA_DAEMON" 2>/dev/null || true
    fi

    echo "$version" > "$VERSION_FILE"
}

check_and_update() {
    local latest=$(get_latest_version)
    local current=""

    [ -f "$VERSION_FILE" ] && current=$(cat "$VERSION_FILE")

    if [ -z "$latest" ]; then
        [ -f "$MAYLA_CLI" ] && [ -f "$MAYLA_DAEMON" ] && return 0
        echo "Error: Cannot fetch latest version and binaries not found" >&2
        exit 1
    fi

    if [ "$latest" != "$current" ] || [ ! -f "$MAYLA_CLI" ] || [ ! -f "$MAYLA_DAEMON" ]; then
        download_binary "$latest"
    fi
}

check_and_update

if ! pgrep -f "mayla-daemon" > /dev/null 2>&1; then
    "$MAYLA_DAEMON" &
    sleep 1
fi

exec "$MAYLA_CLI" "$@"
