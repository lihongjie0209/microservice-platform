#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
chart_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

cp -R "$chart_dir" "$test_root/platform-library"
consumer_chart="$test_root/platform-library/tests/gateway-consumer"
helm dependency build "$consumer_chart" >/dev/null

development_output=$(helm template gateway-test "$consumer_chart" \
  --set name=identity-service \
  --set namespace=platform-development \
  --set gateway.enabled=true \
  --set gateway.baseDomain=aaa.com \
  --set gateway.environmentLabel=dev \
  --set gateway.ingressClassName=apisix-dev)

printf '%s\n' "$development_output" | grep -q '"identity-service.dev.aaa.com"'
printf '%s\n' "$development_output" | grep -q 'ingressClassName: "apisix-dev"'
printf '%s\n' "$development_output" | grep -q 'resolveGranularity: endpoints'

production_output=$(helm template gateway-test "$consumer_chart" \
  --set name=identity-service \
  --set namespace=platform-production \
  --set gateway.enabled=true \
  --set gateway.baseDomain=aaa.com \
  --set gateway.production=true \
  --set gateway.ingressClassName=apisix-prod)

printf '%s\n' "$production_output" | grep -q '"identity-service.aaa.com"'
if printf '%s\n' "$production_output" | grep -q 'identity-service.production.aaa.com'; then
  echo "production hostname unexpectedly contains an environment label" >&2
  exit 1
fi

disabled_output=$(helm template gateway-test "$consumer_chart" \
  --set name=identity-service \
  --set namespace=platform-development)
if [ -n "$disabled_output" ]; then
  echo "disabled gateway unexpectedly rendered a route" >&2
  exit 1
fi

if helm template gateway-test "$consumer_chart" \
  --set name=identity-service \
  --set namespace=platform-development \
  --set gateway.enabled=true >/dev/null 2>&1; then
  echo "gateway without a base domain unexpectedly rendered successfully" >&2
  exit 1
fi
