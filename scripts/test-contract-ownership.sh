#!/bin/sh
set -eu

if find services -type f -name '*.proto' -print -quit | grep -q .; then
	echo "service-local Proto found; business contracts belong in platform-protos" >&2
	find services -type f -name '*.proto' -print >&2
	exit 1
fi

for service in services/*-service; do
	if ! grep -q 'github.com/lihongjie0209/platform-protos ' "$service/go.mod"; then
		echo "$service: released platform-protos dependency is missing" >&2
		exit 1
	fi
	if grep -R -E -q 'gen/hello|RegisterHelloService|UnimplementedHelloService' "$service" --exclude-dir=.git --exclude-dir=bin; then
		echo "$service: scaffold Hello contract remains" >&2
		exit 1
	fi
done
