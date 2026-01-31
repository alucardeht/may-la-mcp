#!/bin/bash

MAYLA_DIR="$HOME/.mayla/instances"

if [[ ! -d "$MAYLA_DIR" ]]; then
	exit 0
fi

backup_instance() {
	local instance_dir=$1
	local instance_id=$(basename "$instance_dir")
	local backup_dir="$HOME/.mayla/backups/$instance_id-$(date +%s)"

	mkdir -p "$HOME/.mayla/backups"
	mv "$instance_dir" "$backup_dir" 2>/dev/null || return 1
	echo "Backed up instance to: $backup_dir" >&2
	return 0
}

find "$MAYLA_DIR" -type d -mindepth 1 -maxdepth 1 -mmin +60 | while read -r instance_dir; do
	instance_id=$(basename "$instance_dir")

	if [[ -f "$instance_dir/memory.db" ]] || [[ -f "$instance_dir/index.db" ]]; then
		echo "WARNING: Preserving instance $instance_id (contains databases)" >&2
		continue
	fi

	workspace_file="$instance_dir/workspace.path"
	if [[ -f "$workspace_file" ]]; then
		workspace=$(cat "$workspace_file" 2>/dev/null)
		if [[ ! -d "$workspace" ]]; then
			echo "Workspace deleted, backing up instance: $instance_id"
			backup_instance "$instance_dir"
			continue
		fi
	fi

	daemon_sock="$instance_dir/daemon.sock"
	if [[ ! -f "$daemon_sock" ]]; then
		echo "Socket missing, backing up instance: $instance_id"
		backup_instance "$instance_dir"
		continue
	fi

	if ! timeout 2 bash -c "echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\",\"params\":{}}' | nc -U '$daemon_sock' > /dev/null 2>&1" 2>/dev/null; then
		echo "Daemon unhealthy, backing up instance: $instance_id"
		backup_instance "$instance_dir"
	fi
done
