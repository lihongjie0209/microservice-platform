#!/bin/sh
set -eu

for service in services/*-service; do
	name=$(basename "$service")
	if find "$service/internal/principal" -type f -name '*.go' 2>/dev/null | grep -q .; then
		echo "$service: service-local principal contexts are forbidden; use platform-go/principal" >&2
		exit 1
	fi
	if ! grep -R -q 'microservice-platform-go/principal' "$service/internal" --include='*.go'; then
		echo "$service: shared platform principal context is required" >&2
		exit 1
	fi
	if [ "$name" != identity-service ] && ! grep -R -q 'microservice-platform-go/authn' "$service/internal" --include='*.go'; then
		echo "$service: non-identity services must verify identity-service tokens through platform-go/authn" >&2
		exit 1
	fi
	spec="$service/docs/swagger.json"
	if [ ! -s "$spec" ] || ! grep -q '"post"[[:space:]]*:' "$spec"; then
		echo "$service: generated OpenAPI POST operations are missing" >&2
		exit 1
	fi
	if grep -E -q '"(put|patch|delete)"[[:space:]]*:' "$spec"; then
		echo "$service: business OpenAPI operations must use POST" >&2
		exit 1
	fi
	get_count=$(grep -c '"get"[[:space:]]*:' "$spec" || true)
	if [ "$name" = identity-service ]; then
		if [ "$get_count" -ne 1 ] || ! grep -q '"/.well-known/jwks.json"' "$spec"; then
			echo "$service: only the standards-required JWKS GET is allowed" >&2
			exit 1
		fi
	elif [ "$get_count" -ne 0 ]; then
		echo "$service: documented business GET operation found" >&2
		exit 1
	fi
	if ! grep -q '"Bearer"' "$spec" || ! grep -q '"PSK"' "$spec"; then
		echo "$service: JWT and PSK OpenAPI security definitions are required" >&2
		exit 1
	fi
	if ! grep -R -E -q 'HTTPPaths:.*\*' "$service/internal/transport/http" --include='*_test.go'; then
		echo "$service: HTTP PSK wildcard regression is missing" >&2
		exit 1
	fi
	if ! grep -R -E -q 'GRPCMethods:.*\*' "$service/internal/transport/grpc" --include='*_test.go'; then
		echo "$service: gRPC PSK wildcard regression is missing" >&2
		exit 1
	fi
done
