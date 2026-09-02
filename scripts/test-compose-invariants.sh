#!/bin/sh
set -eu

compose_file=${1:-environments/local/docker-compose.yml}

if ! grep -Fq 'pg_isready -h 127.0.0.1 -U platform_admin -d platform' "$compose_file"; then
    echo "PostgreSQL health check must require the final TCP listener" >&2
    exit 1
fi

billing_service=$(awk '
    /^  billing-service:/ { in_service = 1; next }
    in_service && /^  [a-zA-Z0-9_-]+:/ { exit }
    in_service { print }
' "$compose_file")
if ! printf '%s\n' "$billing_service" | grep -Fq 'service-registry-service: {condition: service_healthy}'; then
    echo "billing-service publishes import/export provider metadata and must wait for the service registry" >&2
    exit 1
fi
