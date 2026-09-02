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

application_service=$(awk '
    /^  application-service:/ { in_service = 1; next }
    in_service && /^  [a-zA-Z0-9_-]+:/ { exit }
    in_service { print }
' "$compose_file")
import_service=$(awk '
    /^  import-service:/ { in_service = 1; next }
    in_service && /^  [a-zA-Z0-9_-]+:/ { exit }
    in_service { print }
' "$compose_file")
grant_method=/platform.application.v1.ApplicationService/BatchCheckTenantApplications
if ! printf '%s\n' "$application_service" | grep -Fq "APP_AUTH_PSK_GRPC_METHODS: $grant_method"; then
    echo "application-service must protect the tenant-application grant check with PSK" >&2
    exit 1
fi
if grep -Eq 'APP_AUTH_PSK_GRPC_METHODS:.*\[[^]]*\]' "$compose_file"; then
    echo "PSK gRPC method environment variables must use Viper comma-separated syntax without brackets" >&2
    exit 1
fi
if ! printf '%s\n' "$import_service" | grep -Fq 'application-service: {condition: service_healthy}'; then
    echo "import-service must wait for application-service before checking application grants" >&2
    exit 1
fi
if ! printf '%s\n' "$import_service" | grep -Fq 'APP_OUTBOUND_GRPC_APPLICATION_AUTH_TYPE: psk' ||
    ! printf '%s\n' "$import_service" | grep -Fq 'APP_OUTBOUND_GRPC_APPLICATION_TLS_ALLOW_INSECURE: "true"'; then
    echo "import-service must explicitly authenticate development grant checks" >&2
    exit 1
fi
application_psk=$(printf '%s\n' "$application_service" | sed -n 's/^[[:space:]]*APP_AUTH_PSK_KEY:[[:space:]]*//p')
import_psk=$(printf '%s\n' "$import_service" | sed -n 's/^[[:space:]]*APP_OUTBOUND_GRPC_APPLICATION_AUTH_TOKEN:[[:space:]]*//p')
if [ -z "$application_psk" ] || [ "$application_psk" != "$import_psk" ]; then
    echo "application-service and import-service development PSKs must match" >&2
    exit 1
fi

for consumer_name in audit-service billing-service dictionary-service scheduler-service search-service data-export-service; do
    consumer_service=$(awk -v service="$consumer_name" '
        $0 == "  " service ":" { in_service = 1; next }
        in_service && /^  [a-zA-Z0-9_-]+:/ { exit }
        in_service { print }
    ' "$compose_file")
    if ! printf '%s\n' "$consumer_service" | grep -Fq 'application-service: {condition: service_healthy}' ||
        ! printf '%s\n' "$consumer_service" | grep -Fq 'APP_OUTBOUND_GRPC_APPLICATION_AUTH_TYPE: psk' ||
        ! printf '%s\n' "$consumer_service" | grep -Fq 'APP_OUTBOUND_GRPC_APPLICATION_TLS_ALLOW_INSECURE: "true"'; then
        echo "$consumer_name must wait for and authenticate application grant checks" >&2
        exit 1
    fi
    consumer_psk=$(printf '%s\n' "$consumer_service" | sed -n 's/^[[:space:]]*APP_OUTBOUND_GRPC_APPLICATION_AUTH_TOKEN:[[:space:]]*//p')
    if [ "$application_psk" != "$consumer_psk" ]; then
        echo "application-service and $consumer_name development PSKs must match" >&2
        exit 1
    fi
done

data_export_service=$(awk '
    /^  data-export-service:/ { in_service = 1; next }
    in_service && /^  [a-zA-Z0-9_-]+:/ { exit }
    in_service { print }
' "$compose_file")
if ! printf '%s\n' "$data_export_service" | grep -Fq 'APP_OBJECT_STORAGE_PRESIGN_ENDPOINT: 127.0.0.1:9000' ||
    ! printf '%s\n' "$data_export_service" | grep -Fq 'APP_OBJECT_STORAGE_REGION: us-east-1'; then
    echo "data-export-service must sign development download URLs for the host-visible MinIO endpoint" >&2
    exit 1
fi
