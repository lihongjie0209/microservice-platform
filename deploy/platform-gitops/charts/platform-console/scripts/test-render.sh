#!/usr/bin/env sh
set -eu

chart_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
rendered=$(mktemp)
trap 'rm -f "$rendered"' EXIT

helm template platform-console "$chart_dir" \
  --set namespace=platform-testing \
  --set environment=testing \
  --set image.tag=test \
  --set gateway.baseDomain=example.test \
  --set gateway.environmentLabel=test \
  --set gateway.ingressClassName=apisix-test \
  --set networkPolicy.gatewayNamespace=ingress-apisix-testing > "$rendered"

grep -q 'hosts: \["console.test.example.test"\]' "$rendered"
grep -q 'PLATFORM_IDENTITY_URL: "https://identity-service.test.example.test"' "$rendered"
grep -q 'PLATFORM_DATA_EXPORT_URL: "https://data-export-service.test.example.test"' "$rendered"
grep -q 'PLATFORM_IMPORT_URL: "https://import-service.test.example.test"' "$rendered"
grep -q 'readOnlyRootFilesystem: true' "$rendered"
grep -q 'name: prepare-runtime-config' "$rendered"
grep -q 'name: limit-req' "$rendered"
grep -q 'rate: 200' "$rendered"
grep -q 'rejected_code: 429' "$rendered"
grep -q 'name: response-rewrite' "$rendered"
grep -q 'Permissions-Policy: camera=(), geolocation=(), microphone=()' "$rendered"
