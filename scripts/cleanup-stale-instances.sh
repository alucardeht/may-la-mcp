#!/bin/bash

MAYLA_DIR="$HOME/.mayla/instances"

if [[ ! -d "$MAYLA_DIR" ]]; then
	exit 0
fi

find "$MAYLA_DIR" -type d -mindepth 1 -maxdepth 1 -mmin +60 | while read -r instance_dir; do
	instance_id=$(basename "$instance_dir")

	workspace_file="$instance_dir/workspace.path"
	if [[ -f "$workspace_file" ]]; then
		workspace=$(cat "$workspace_file" 2>/dev/null)
		if [[ ! -d "$workspace" ]]; then
			echo "Workspace deleted, cleaning up instance: $instance_id"
			rm -rf "$instance_dir"
			continue
		fi
	fi

	daemon_sock="$instance_dir/daemon.sock"
	if [[ ! -f "$daemon_sock" ]]; then
		echo "Socket missing, cleaning up instance: $instance_id"
		rm -rf "$instance_dir"
		continue
	fi

	if ! timeout 2 bash -c "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\",\"params\":{}}' | nc -U '$daemon_sock' > /dev/null 2>&1" 2>/dev/null; then
		echo "Daemon unhealthy, cleaning up instance: $instance_id"
		rm -rf "$instance_dir"
	fi
done
