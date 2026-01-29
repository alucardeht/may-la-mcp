#!/bin/bash

MAYLA_DIR="$HOME/.mayla/instances"

if [[ ! -d "$MAYLA_DIR" ]]; then
	exit 0
fi

find "$MAYLA_DIR" -type d -mindepth 1 -maxdepth 1 -mmin +60 | while read -r instance_dir; do
	instance_id=$(basename "$instance_dir")

	pid_file="$instance_dir/daemon.pid"
	if [[ -f "$pid_file" ]]; then
		pid=$(cat "$pid_file" 2>/dev/null)
		if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
			continue
		fi
	fi

	echo "Cleaning up stale instance: $instance_id"
	rm -rf "$instance_dir"
done
