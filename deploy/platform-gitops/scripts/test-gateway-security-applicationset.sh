#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
gitops_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
applicationset="$gitops_dir/gateway-security-applicationset.yaml"
yq_binary=${YQ:-yq}

"$yq_binary" eval '.' "$applicationset" >/dev/null

[ "$("$yq_binary" -r '.spec.template.spec.source.path' "$applicationset")" = "deploy/platform-gitops/charts/platform-gateway-security" ]
[ "$("$yq_binary" -r '.spec.generators[0].list.elements | length' "$applicationset")" -eq 4 ]

for environment in development testing staging production; do
  [ "$("$yq_binary" -r ".spec.generators[0].list.elements[] | select(.environment == \"$environment\") | .environment" "$applicationset")" = "$environment" ]
  grep -q "platform/{{ .environment }}/zerossl-eab" "$applicationset"
  grep -q "platform/{{ .environment }}/dns" "$applicationset"
done

grep -q 'enabled: {{ ne .baseDomain "" }}' "$applicationset"
grep -q 'ingressClassName: {{ .ingressClass | quote }}' "$applicationset"
