#!/bin/sh
set -eu

if grep -R -q 'github.com/nats-io/nats.go' services/*-service/internal --include='*.go'; then
	echo "services must use the shared platform-go eventbus instead of direct NATS clients" >&2
	exit 1
fi

producers="identity tenant authorization application dictionary workflow metering billing rule data-export import config notification file"
for name in $producers; do
	service="services/$name-service"
	if ! grep -R -E -q 'platformoutbox|AddOutbox' "$service/internal" --include='*.go'; then
		echo "$service: transactional Outbox integration is missing" >&2
		exit 1
	fi
	for dialect in postgres kingbase mysql; do
		if ! grep -R -q 'outbox' "$service/migrations/$dialect" --include='*.sql'; then
			echo "$service: $dialect Outbox migration is missing" >&2
			exit 1
		fi
	done
done

for name in search webhook; do
	service="services/$name-service"
	if ! grep -R -q 'platforminbox' "$service/internal" --include='*.go'; then
		echo "$service: transactional Inbox integration is missing" >&2
		exit 1
	fi
	for dialect in postgres kingbase mysql; do
		if ! grep -R -q 'event_inbox' "$service/migrations/$dialect" --include='*.sql'; then
			echo "$service: $dialect Inbox migration is missing" >&2
			exit 1
		fi
	done
done

if ! grep -q 'ErrDuplicate' services/audit-service/internal/audit/service.go ||
	! grep -q 'ConcurrentDuplicateEvent' services/audit-service/internal/audit/service_test.go; then
	echo "audit-service: duplicate event acknowledgement regression is missing" >&2
	exit 1
fi

for name in workflow data-export import; do
	if ! grep -R -q 'Claim' "services/$name-service/internal" --include='*.go'; then
		echo "services/$name-service: durable command claim is missing" >&2
		exit 1
	fi
done
