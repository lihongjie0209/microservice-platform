.PHONY: infra-up infra-down infra-status contracts-check sdk-test sdk-integration

infra-up:
	docker compose -f environments/local/docker-compose.yml up -d --wait

infra-down:
	docker compose -f environments/local/docker-compose.yml down

infra-status:
	docker compose -f environments/local/docker-compose.yml ps

contracts-check:
	cd contracts/platform-protos && buf lint && buf generate && git diff --exit-code -- gen/go

sdk-test:
	cd libraries/platform-go && go test -race ./... && go vet ./...

sdk-integration:
	cd libraries/platform-go && go test -tags=integration -count=1 -timeout=5m ./integration/...

