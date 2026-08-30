#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
gitops_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
applicationset="$gitops_dir/applicationset.yaml"
yq_binary=${YQ:-yq}

"$yq_binary" eval '.' "$applicationset" >/dev/null

repository=$("$yq_binary" -r '.spec.template.spec.source.repoURL' "$applicationset")
chart_path=$("$yq_binary" -r '.spec.template.spec.source.path' "$applicationset")
service_count=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements | length' "$applicationset")
environment_count=$("$yq_binary" -r '.spec.generators[0].matrix.generators[0].list.elements | length' "$applicationset")

[ "$repository" = "https://github.com/lihongjie0209/microservice-platform.git" ]
[ "$chart_path" = "deploy/platform-gitops/charts/platform-service" ]
[ "$service_count" -eq 15 ]
[ "$environment_count" -eq 3 ]

webhook_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "webhook-service") | .schema' "$applicationset")
[ "$webhook_schema" = "webhook" ]
workflow_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "workflow-service") | .schema' "$applicationset")
[ "$workflow_schema" = "workflow" ]
search_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "search-service") | .schema' "$applicationset")
[ "$search_schema" = "search" ]

if grep -q 'github.com/lihongjie0209/platform-gitops' "$applicationset"; then
  echo "service ApplicationSet still references the removed repository" >&2
  exit 1
fi
