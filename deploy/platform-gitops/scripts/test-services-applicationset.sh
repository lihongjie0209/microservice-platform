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
[ "$service_count" -eq 20 ]
[ "$environment_count" -eq 4 ]

for environment in development testing staging production; do
  count=$("$yq_binary" -r ".spec.generators[0].matrix.generators[0].list.elements[] | select(.environment == \"$environment\") | .environment" "$applicationset" | wc -l)
  [ "$count" -eq 1 ]
done

staging_profile=$("$yq_binary" -r '.spec.generators[0].matrix.generators[0].list.elements[] | select(.environment == "staging") | .profile' "$applicationset")
[ "$staging_profile" = "production" ]

webhook_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "webhook-service") | .schema' "$applicationset")
[ "$webhook_schema" = "webhook" ]
workflow_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "workflow-service") | .schema' "$applicationset")
[ "$workflow_schema" = "workflow" ]
search_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "search-service") | .schema' "$applicationset")
[ "$search_schema" = "search" ]
metering_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "metering-service") | .schema' "$applicationset")
[ "$metering_schema" = "metering" ]
billing_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "billing-service") | .schema' "$applicationset")
[ "$billing_schema" = "billing" ]
rule_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "rule-service") | .schema' "$applicationset")
[ "$rule_schema" = "rule_service" ]
export_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "data-export-service") | .schema' "$applicationset")
[ "$export_schema" = "data_export" ]
import_schema=$("$yq_binary" -r '.spec.generators[0].matrix.generators[1].list.elements[] | select(.service == "import-service") | .schema' "$applicationset")
[ "$import_schema" = "data_import" ]

if grep -q 'github.com/lihongjie0209/platform-gitops' "$applicationset"; then
  echo "service ApplicationSet still references the removed repository" >&2
  exit 1
fi
