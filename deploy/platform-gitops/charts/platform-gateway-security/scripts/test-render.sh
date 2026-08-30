#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
chart_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)

development_output=$(helm template gateway-security "$chart_dir" \
  --set enabled=true \
  --set baseDomain=aaa.com \
  --set environmentLabel=dev \
  --set ingressClassName=apisix-dev \
  --set issuer.email=platform@example.com \
  --set issuer.eabKeyID=test-key-id \
  --set 'issuer.solvers[0].dns01.cloudflare.apiTokenSecretRef.name=zerossl-dns' \
  --set 'issuer.solvers[0].dns01.cloudflare.apiTokenSecretRef.key=api-token')

printf '%s\n' "$development_output" | grep -q '"dev.aaa.com"'
printf '%s\n' "$development_output" | grep -q '"\*.dev.aaa.com"'
printf '%s\n' "$development_output" | grep -q 'https://acme.zerossl.com/v2/DV90'
printf '%s\n' "$development_output" | grep -q 'keyAlgorithm: HS256'
printf '%s\n' "$development_output" | grep -q 'kind: ApisixTls'
printf '%s\n' "$development_output" | grep -q 'ingressClassName: "apisix-dev"'
printf '%s\n' "$development_output" | grep -q 'namespace: default'

production_output=$(helm template gateway-security "$chart_dir" \
  --set enabled=true \
  --set baseDomain=aaa.com \
  --set production=true \
  --set issuer.email=platform@example.com \
  --set issuer.eabKeyID=test-key-id \
  --set 'issuer.solvers[0].dns01.cloudflare.apiTokenSecretRef.name=zerossl-dns' \
  --set 'issuer.solvers[0].dns01.cloudflare.apiTokenSecretRef.key=api-token')

printf '%s\n' "$production_output" | grep -q '"aaa.com"'
printf '%s\n' "$production_output" | grep -q '"\*.aaa.com"'
if printf '%s\n' "$production_output" | grep -q '\.production\.aaa\.com'; then
  echo "production certificate unexpectedly contains an environment label" >&2
  exit 1
fi

if helm template gateway-security "$chart_dir" \
  --set enabled=true \
  --set baseDomain=aaa.com \
  --set environmentLabel=dev >/dev/null 2>&1; then
  echo "enabled ZeroSSL issuer unexpectedly accepted missing EAB configuration" >&2
  exit 1
fi
