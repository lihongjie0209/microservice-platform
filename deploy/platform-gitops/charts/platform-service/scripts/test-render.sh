#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
chart_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
deploy_dir=$(CDPATH= cd -- "$chart_dir/../../.." && pwd)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

mkdir -p "$test_root/deploy/platform-gitops/charts"
cp -R "$deploy_dir/platform-helm" "$test_root/deploy/platform-helm"
cp -R "$chart_dir" "$test_root/deploy/platform-gitops/charts/platform-service"
test_chart="$test_root/deploy/platform-gitops/charts/platform-service"

helm dependency build "$test_chart" >/dev/null
helm lint "$test_chart" \
  --set name=identity-service \
  --set namespace=platform-development \
  --set image.repository=ghcr.io/lihongjie0209/identity-service \
  --set image.tag=v0.1.0 \
  --set database.schema=identity \
  --set database.migrationTable=identity_schema_migrations \
  --set externalSecret.key=platform/development/identity-service >/dev/null

identity_output=$(helm template identity-service "$test_chart" \
  --namespace platform-development \
  --set name=identity-service \
  --set namespace=platform-development \
  --set environment=development \
  --set image.repository=ghcr.io/lihongjie0209/identity-service \
  --set image.tag=v0.1.0 \
  --set database.schema=identity \
  --set database.migrationTable=identity_schema_migrations \
  --set externalSecret.key=platform/development/identity-service \
  --set networkPolicy.gatewayNamespace=ingress-apisix-development \
  --set gateway.enabled=true \
  --set gateway.baseDomain=aaa.com \
  --set gateway.environmentLabel=dev \
  --set gateway.ingressClassName=apisix-dev)

printf '%s\n' "$identity_output" | grep -q 'kind: Deployment'
printf '%s\n' "$identity_output" | grep -q 'initContainers:'
printf '%s\n' "$identity_output" | grep -q 'name: migrate'
if printf '%s\n' "$identity_output" | grep -q 'kind: Job'; then
  echo "default init-container migration unexpectedly rendered a migration Job" >&2
  exit 1
fi
printf '%s\n' "$identity_output" | grep -q 'kind: ApisixRoute'
printf '%s\n' "$identity_output" | grep -q '"identity-service.dev.aaa.com"'
printf '%s\n' "$identity_output" | grep -q 'name: client-control'
printf '%s\n' "$identity_output" | grep -q 'name: limit-req'
printf '%s\n' "$identity_output" | grep -q 'name: response-rewrite'
printf '%s\n' "$identity_output" | grep -q 'kubernetes.io/metadata.name: ingress-apisix-development'
printf '%s\n' "$identity_output" | grep -q 'platform.swagger/enabled: "true"'
printf '%s\n' "$identity_output" | grep -q 'APP_EVENT_BUS_STREAM_NAME: "PLATFORM_EVENTS"'
printf '%s\n' "$identity_output" | grep -q 'APP_HTTP_CORS_ENABLED: "true"'
printf '%s\n' "$identity_output" | grep -q 'APP_HTTP_CORS_ALLOWED_ORIGINS: "\[https://console.dev.aaa.com\]"'
printf '%s\n' "$identity_output" | grep -q 'mountPath: /app/logs'

production_output=$(helm template identity-service "$test_chart" \
  --namespace platform-production \
  --set name=identity-service \
  --set namespace=platform-production \
  --set environment=production \
  --set image.repository=ghcr.io/lihongjie0209/identity-service \
  --set image.tag=v0.1.0 \
  --set database.schema=identity \
  --set database.migrationTable=identity_schema_migrations \
  --set externalSecret.key=platform/production/identity-service \
  --set gateway.enabled=true \
  --set gateway.baseDomain=aaa.com \
  --set gateway.production=true)
printf '%s\n' "$production_output" | grep -q 'APP_HTTP_CORS_ALLOWED_ORIGINS: "\[https://console.aaa.com\]"'

registry_output=$(helm template service-registry-service "$test_chart" \
  --namespace platform-development \
  --set name=service-registry-service \
  --set namespace=platform-development \
  --set environment=development \
  --set image.repository=ghcr.io/lihongjie0209/service-registry-service \
  --set image.tag=v0.1.0 \
  --set database.enabled=false \
  --set redis.enabled=true \
  --set eventBus.enabled=false \
  --set externalSecret.key=platform/development/service-registry-service)

if printf '%s\n' "$registry_output" | grep -q 'kind: Job'; then
  echo "database-free registry unexpectedly rendered a migration Job" >&2
  exit 1
fi
if printf '%s\n' "$registry_output" | grep -q 'initContainers:'; then
  echo "database-free registry unexpectedly rendered a migration init container" >&2
  exit 1
fi
if printf '%s\n' "$registry_output" | grep -q 'kind: ApisixRoute'; then
  echo "gateway-disabled registry unexpectedly rendered an APISIX route" >&2
  exit 1
fi
printf '%s\n' "$registry_output" | grep -q 'APP_DATABASE_ENABLED: "false"'
printf '%s\n' "$registry_output" | grep -q 'APP_REDIS_ENABLED: "true"'
printf '%s\n' "$registry_output" | grep -q 'APP_EVENT_BUS_ENABLED: "false"'

workflow_output=$(helm template workflow-service "$test_chart" \
  --namespace platform-development \
  --set name=workflow-service \
  --set namespace=platform-development \
  --set environment=development \
  --set image.repository=ghcr.io/lihongjie0209/workflow-service \
  --set image.tag=v0.1.0 \
  --set database.schema=workflow \
  --set database.migrationTable=workflow_schema_migrations \
  --set workflow.enabled=true \
  --set externalSecret.key=platform/development/workflow-service)

printf '%s\n' "$workflow_output" | grep -q 'APP_TEMPORAL_ENABLED: "true"'
printf '%s\n' "$workflow_output" | grep -q 'temporal.platform-infrastructure.svc.cluster.local:7233'
printf '%s\n' "$workflow_output" | grep -q 'dns:///authorization-service.platform-development.svc.cluster.local:9090'
printf '%s\n' "$workflow_output" | grep -q 'APP_AUTH_AUDIENCE: "workflow-service"'

search_output=$(helm template search-service "$test_chart" \
  --namespace platform-development \
  --set name=search-service \
  --set namespace=platform-development \
  --set environment=development \
  --set image.repository=ghcr.io/lihongjie0209/search-service \
  --set image.tag=v0.1.0 \
  --set database.schema=search \
  --set database.migrationTable=search_schema_migrations \
  --set search.enabled=true \
  --set externalSecret.key=platform/development/search-service)

printf '%s\n' "$search_output" | grep -q 'APP_OPENSEARCH_ENABLED: "true"'
printf '%s\n' "$search_output" | grep -q 'opensearch.platform-infrastructure.svc.cluster.local:9200'
printf '%s\n' "$search_output" | grep -q 'dns:///authorization-service.platform-development.svc.cluster.local:9090'
printf '%s\n' "$search_output" | grep -q 'APP_AUTH_AUDIENCE: "search-service"'
