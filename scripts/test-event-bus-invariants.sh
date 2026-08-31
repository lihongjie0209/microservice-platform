#!/bin/sh
set -eu

for service in services/*-service; do
	config="$service/config/config.yaml"
	if ! grep -q 'stream_name: PLATFORM_EVENTS' "$config"; then
		echo "$service: canonical PLATFORM_EVENTS stream is missing" >&2
		exit 1
	fi
	if grep -R -E -q 'SERVICE_EVENTS|subjects:.*service\.>|"service\.>"' "$service/config" "$service/internal/config"; then
		echo "$service: legacy private event stream configuration remains" >&2
		exit 1
	fi
done
