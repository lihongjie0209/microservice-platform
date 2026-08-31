#!/bin/sh
set -eu

compose_file=${1:-environments/local/docker-compose.yml}

if ! grep -Fq 'pg_isready -h 127.0.0.1 -U platform_admin -d platform' "$compose_file"; then
    echo "PostgreSQL health check must require the final TCP listener" >&2
    exit 1
fi
