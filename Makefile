SHELL := /bin/sh

TOOLS_DIR := $(CURDIR)/.tools/bin
BUF := $(TOOLS_DIR)/buf
PROTOC_GEN_GO := $(TOOLS_DIR)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(TOOLS_DIR)/protoc-gen-go-grpc
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint
YQ := $(TOOLS_DIR)/yq
BUF_VERSION ?= v1.50.0
PROTOC_GEN_GO_VERSION ?= v1.36.12
PROTOC_GEN_GO_GRPC_VERSION ?= v1.5.1
GOLANGCI_LINT_VERSION ?= v2.13.1
YQ_VERSION ?= v4.50.1
SERVICES := identity-service tenant-service authorization-service audit-service config-service notification-service file-service scheduler-service swagger-service application-service dictionary-service service-registry-service
SERVICE ?=
SERVICE_DIR = services/$(SERVICE)

.DEFAULT_GOAL := help
.PHONY: help bootstrap fmt contracts contracts-check sdk-test sdk-integration \
	services-build services-test services-vet services-lint services-swagger services-swagger-check services-integration \
	service-check service-run service-build service-docker-build service-test service-test-race \
	service-test-integration service-lint service-fmt service-swagger-check service-migrate-up \
	service-migrate-down service-dev-up service-dev-down service-dev-logs \
	build test test-integration lint swagger swagger-check verify \
	delivery-check compose-check infra-up infra-down infra-logs infra-status dev-up dev-down dev-logs system-test clean clean-tools

help:
	@echo "Workspace commands:"
	@echo "  make bootstrap              Install pinned development tools"
	@echo "  make fmt                    Format shared SDK, services and Proto"
	@echo "  make contracts              Format, lint and generate shared Proto SDK"
	@echo "  make build                  Build every service"
	@echo "  make test                   Run SDK and service unit tests with race detection"
	@echo "  make test-integration       Run isolated Testcontainers suites"
	@echo "  make lint                   Run vet and configured service linters"
	@echo "  make swagger                Regenerate service OpenAPI documents"
	@echo "  make swagger-check          Regenerate and verify service OpenAPI documents"
	@echo "  make verify                 Run contracts, unit tests, vet and OpenAPI checks"
	@echo "  make delivery-check        Lint the shared Helm library chart"
	@echo "  make compose-check         Validate the local Compose environment"
	@echo "  make dev-up                Build and start infrastructure plus all P0 services"
	@echo "  make system-test           Run the multi-service identity/tenant/auth/audit journey"
	@echo ""
	@echo "Single-service commands (SERVICE=$(firstword $(SERVICES))):"
	@echo "  make service-run SERVICE=tenant-service"
	@echo "  make service-build SERVICE=tenant-service"
	@echo "  make service-test SERVICE=tenant-service"
	@echo "  make service-test-integration SERVICE=tenant-service"
	@echo "  make service-migrate-up SERVICE=tenant-service"
	@echo "  make service-dev-up SERVICE=tenant-service"
	@echo "  make service-dev-logs SERVICE=tenant-service"
	@echo ""
	@echo "Infrastructure commands:"
	@echo "  make infra-up               Start PostgreSQL, Redis and NATS"
	@echo "  make infra-status           Show local infrastructure status"
	@echo "  make infra-logs             Follow local infrastructure logs"
	@echo "  make infra-down             Stop local infrastructure"

bootstrap: $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(GOLANGCI_LINT) $(YQ)

$(BUF):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) GOTOOLCHAIN=local go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

$(PROTOC_GEN_GO):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

$(PROTOC_GEN_GO_GRPC):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

$(GOLANGCI_LINT):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(YQ):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install github.com/mikefarah/yq/v4@$(YQ_VERSION)

fmt: $(GOLANGCI_LINT)
	gofmt -w $$(find libraries services -name '*.go' -type f)
	@set -e; for service in $(SERVICES); do (cd services/$$service && $(GOLANGCI_LINT) fmt ./...); done
	@cd libraries/platform-go && $(GOLANGCI_LINT) fmt ./...
	$(MAKE) -C contracts/platform-protos fmt TOOLS_DIR=$(TOOLS_DIR)

contracts: bootstrap
	$(MAKE) -C contracts/platform-protos generate TOOLS_DIR=$(TOOLS_DIR)

contracts-check: bootstrap
	$(MAKE) -C contracts/platform-protos check TOOLS_DIR=$(TOOLS_DIR)

sdk-test:
	$(MAKE) -C libraries/platform-go test

sdk-integration:
	$(MAKE) -C libraries/platform-go test-integration

services-build:
	@set -e; for service in $(SERVICES); do $(MAKE) -C services/$$service build; done

services-test:
	@set -e; for service in $(SERVICES); do $(MAKE) -C services/$$service test-race; done

services-vet:
	@set -e; for service in $(SERVICES); do (cd services/$$service && go vet ./...); done

services-lint: $(GOLANGCI_LINT)
	@set -e; for service in $(SERVICES); do PATH="$(TOOLS_DIR):$$PATH" $(MAKE) -C services/$$service lint; done

services-swagger-check:
	@set -e; for service in $(SERVICES); do $(MAKE) -C services/$$service swagger-check; done

services-swagger:
	@set -e; for service in $(SERVICES); do $(MAKE) -C services/$$service swagger; done

services-integration:
	@set -e; for service in $(SERVICES); do $(MAKE) -C services/$$service test-integration; done

service-check:
	@if [ -z "$(SERVICE)" ]; then echo "SERVICE is required; choose one of: $(SERVICES)" >&2; exit 2; fi
	@case " $(SERVICES) " in *" $(SERVICE) "*) ;; *) echo "unknown SERVICE=$(SERVICE); choose one of: $(SERVICES)" >&2; exit 2;; esac

service-run service-build service-docker-build service-test service-test-race service-test-integration service-lint service-fmt service-swagger-check service-migrate-up service-migrate-down service-dev-up service-dev-down service-dev-logs: service-check
	@target=$@; PATH="$(TOOLS_DIR):$$PATH" $(MAKE) -C $(SERVICE_DIR) $${target#service-}

build: services-build

test: sdk-test services-test

test-integration: sdk-integration services-integration

lint: services-vet services-lint

swagger: services-swagger

swagger-check: services-swagger-check

verify: contracts-check sdk-test services-test services-vet services-swagger-check delivery-check compose-check

delivery-check: $(YQ)
	helm lint deploy/platform-helm
	sh deploy/platform-helm/scripts/test-gateway.sh
	helm lint deploy/platform-gitops/charts/platform-gateway-security
	sh deploy/platform-gitops/charts/platform-gateway-security/scripts/test-render.sh
	YQ=$(YQ) sh deploy/platform-gitops/scripts/test-apisix-applicationset.sh
	sh deploy/platform-gitops/charts/platform-service/scripts/test-render.sh
	YQ=$(YQ) sh deploy/platform-gitops/scripts/test-services-applicationset.sh

compose-check:
	docker compose -f environments/local/docker-compose.yml config -q
	docker compose --profile platform -f environments/local/docker-compose.yml config -q

infra-up:
	docker compose -f environments/local/docker-compose.yml up -d --wait

infra-down:
	docker compose -f environments/local/docker-compose.yml down

infra-logs:
	docker compose -f environments/local/docker-compose.yml logs -f

infra-status:
	docker compose -f environments/local/docker-compose.yml ps

dev-up:
	docker compose --profile platform -f environments/local/docker-compose.yml up --build -d --wait

dev-down:
	docker compose --profile platform -f environments/local/docker-compose.yml down --remove-orphans

dev-logs:
	docker compose --profile platform -f environments/local/docker-compose.yml logs -f

system-test: dev-up
	cd system-tests && go test -tags=system -count=1 -timeout=5m ./...

clean:
	@set -e; for service in $(SERVICES); do $(MAKE) -C services/$$service clean; done

clean-tools:
	rm -rf $(CURDIR)/.tools
