#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
gitops_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
applicationset="$gitops_dir/console-applicationset.yaml"
yq_binary=${YQ:-yq}

"$yq_binary" eval '.' "$applicationset" >/dev/null

repository=$("$yq_binary" -r '.spec.template.spec.source.repoURL' "$applicationset")
chart_path=$("$yq_binary" -r '.spec.template.spec.source.path' "$applicationset")
environment_count=$("$yq_binary" -r '.spec.generators[0].list.elements | length' "$applicationset")
image_repository=$("$yq_binary" -r '.spec.template.spec.source.helm.parameters[] | select(.name == "image.tag") | .value' "$applicationset")

[ "$repository" = "https://github.com/lihongjie0209/microservice-platform.git" ]
[ "$chart_path" = "deploy/platform-gitops/charts/platform-console" ]
[ "$environment_count" -eq 4 ]
[ "$image_repository" = "{{ .imageTag }}" ]

for environment in development testing staging production; do
  count=$("$yq_binary" -r ".spec.generators[0].list.elements[] | select(.environment == \"$environment\") | .environment" "$applicationset" | wc -l)
  [ "$count" -eq 1 ]
done
