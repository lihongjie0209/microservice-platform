#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
gitops_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
applicationset="$gitops_dir/gateway-applicationset.yaml"
yq_binary=${YQ:-yq}

"$yq_binary" eval '.' "$applicationset" >/dev/null

render_development() {
  "$yq_binary" -r \
    '.spec.template.spec.source.helm.values
     | sub("{{replicas}}", "1")
     | sub("{{etcdReplicas}}", "1")
     | sub("{{ingressClass}}", "apisix-dev")
     | sub("{{namespace}}", "ingress-apisix-development")
     | sub("{{environment}}", "development")' \
    "$applicationset" |
    helm template apisix-development apisix/apisix \
      --version 2.17.0 \
      --namespace ingress-apisix-development \
      -f - >/dev/null
}

render_attempt=1
while ! render_development; do
  if [ "$render_attempt" -ge 3 ]; then
    echo "failed to render the pinned APISIX chart after $render_attempt attempts" >&2
    exit 1
  fi
  render_attempt=$((render_attempt + 1))
done
