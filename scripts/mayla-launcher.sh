#!/bin/bash
set -e

REPO="alucardeht/may-la-mcp"
INSTALL_DIR="$HOME/.mayla"
MAYLA_CLI="$INSTALL_DIR/mayla"
MAYLA_DAEMON="$INSTALL_DIR/mayla-daemon"
VERSION_FILE="$INSTALL_DIR/version"

mkdir -p "$INSTALL_DIR"

get_latest_version() {
	latest=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' 2>/dev/null)
	echo "$latest"
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
	latest=$(get_latest_version)
	current=""

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

cleanup_stale_instances() {
	find "$INSTALL_DIR/instances" -type d -mindepth 1 -maxdepth 1 -mmin +60 2>/dev/null | while read -r instance_dir; do
		pid_file="$instance_dir/daemon.pid"

		if [[ -f "$pid_file" ]]; then
			pid=$(cat "$pid_file" 2>/dev/null)
			if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
				continue
			fi
		fi

		rm -rf "$instance_dir" 2>/dev/null || true
	done
}

check_and_update
cleanup_stale_instances

exec "$MAYLA_CLI" "$@"
