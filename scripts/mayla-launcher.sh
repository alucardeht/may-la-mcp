#!/bin/bash
set -e

REPO="alucardeht/may-la-mcp"
INSTALL_DIR="$HOME/.mayla"
BINARY="$INSTALL_DIR/mayla-daemon"
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

    echo "${os}-${arch}"
}

download_binary() {
    local version=$1
    local platform=$(get_platform)
    local url="https://github.com/$REPO/releases/download/$version/mayla-daemon-$platform"

    echo "Downloading May-la $version for $platform..." >&2
    curl -sL "$url" -o "$BINARY.tmp" && mv "$BINARY.tmp" "$BINARY" && chmod +x "$BINARY"
    echo "$version" > "$VERSION_FILE"
}

check_and_update() {
    local latest=$(get_latest_version)
    local current=""

    [ -f "$VERSION_FILE" ] && current=$(cat "$VERSION_FILE")

    if [ -z "$latest" ]; then
        [ -f "$BINARY" ] && return 0
        echo "Error: Cannot fetch latest version and no local binary found" >&2
        exit 1
    fi

    if [ "$latest" != "$current" ] || [ ! -f "$BINARY" ]; then
        download_binary "$latest"
    fi
}

check_and_update
exec "$BINARY" "$@"
