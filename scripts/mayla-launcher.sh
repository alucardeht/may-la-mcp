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

backup_instance() {
	local instance_dir=$1
	local instance_id=$(basename "$instance_dir")
	local backup_dir="$INSTALL_DIR/backups/$instance_id-$(date +%s)"

	mkdir -p "$INSTALL_DIR/backups"
	mv "$instance_dir" "$backup_dir" 2>/dev/null || return 1
	echo "Backed up instance to: $backup_dir" >&2
	return 0
}

cleanup_stale_instances() {
	find "$INSTALL_DIR/instances" -type d -mindepth 1 -maxdepth 1 -mmin +60 2>/dev/null | while read -r instance_dir; do
		instance_id=$(basename "$instance_dir")

		if [[ -f "$instance_dir/memory.db" ]] || [[ -f "$instance_dir/index.db" ]]; then
			echo "WARNING: Preserving instance $instance_id (contains databases)" >&2
			continue
		fi

		workspace_file="$instance_dir/workspace.path"
		if [[ -f "$workspace_file" ]]; then
			workspace=$(cat "$workspace_file" 2>/dev/null)
			if [[ ! -d "$workspace" ]]; then
				backup_instance "$instance_dir" || true
				continue
			fi
		fi

		daemon_sock="$instance_dir/daemon.sock"
		if [[ ! -f "$daemon_sock" ]]; then
			backup_instance "$instance_dir" || true
			continue
		fi

		if ! timeout 2 bash -c "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\",\"params\":{}}' | nc -U '$daemon_sock' > /dev/null 2>&1" 2>/dev/null; then
			backup_instance "$instance_dir" || true
		fi
	done
}

check_and_update
cleanup_stale_instances

exec "$MAYLA_CLI" "$@"
