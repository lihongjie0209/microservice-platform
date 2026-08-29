---
name: microservice-platform
description: Implement, modify, review, generate, test, or deploy any service, shared contract, SDK, event, database migration, frontend API, internal gRPC API, or infrastructure in the microservice-platform workspace. Always use this skill for work under this workspace so service boundaries, database/schema isolation, contract ownership, event reliability, and verification gates remain consistent.
compatibility: Requires Go, Git, Docker, Buf, protoc-gen-go, and protoc-gen-go-grpc. Integration tests require Docker.
---

# Microservice Platform Engineering

Use this workflow before changing any repository under this workspace.

## Start with evidence

1. Read the workspace `README.md` and `docs/platform-services.md`.
2. Inspect the target repository, its `git status`, configuration, migrations, Proto files, and tests. Do not rely on an earlier turn's description.
3. Read `references/decisions.md` and `references/lessons.md`. Update them when a new architectural decision or reusable failure lesson appears.
4. Identify whether the change belongs to a domain service, `platform-protos`, `platform-go`, deployment repository, or the generator before editing.

## Ownership rules

- `identity-service` owns users, credentials, sessions, tokens, signing keys, external identities, and service accounts.
- `tenant-service` owns tenants, memberships, invitations, organization units, groups, quotas, and tenant lifecycle.
- `authorization-service` owns permissions, roles, bindings, policies, decisions, and data scopes.
- An online service never reads another service's tables. Use versioned gRPC for synchronous facts and NATS JetStream events for asynchronous reactions.
- OLAP/reporting uses CDC, domain events, or scheduled exports to build read-only analytical models. A direct read-only connection to an owning service's OLTP data requires an explicit architecture review, separate credentials, query limits, and must never participate in an online transaction.
- Frontend REST DTOs belong to each service. They may aggregate internal gRPC responses, but must not expose database structs or Proto messages directly.
- Inter-service gRPC and event payloads originate only in `platform-protos`; never copy `.proto` files between services.
- Stable cross-cutting Go code belongs in `platform-go`. Domain models, repositories, migrations, HTTP DTOs, and service-specific policy remain local.

## Database isolation

- Every service owns an independent migration history table.
- PostgreSQL and Kingbase deployments may share a database, but each service uses a distinct schema and least-privilege role.
- MySQL isolates by database name and ignores PostgreSQL schema settings.
- Keep business DSN, migration URL, database name, schema, and migration table consistent in every profile, Compose, Kubernetes, tests, and generated projects.
- Production does not automatically create schemas unless explicitly authorized. Compose may enable schema creation for development.
- Pass `context.Context` to all database operations and use parameterized SQL.

## Contract rules

- Use versioned packages such as `platform.identity.v1`.
- Every RPC uses request and response wrapper messages.
- Map domain failures to precise gRPC status codes; never leak raw internal errors.
- Frontend endpoints use POST with JSON and the platform response envelope, except standards-required discovery endpoints such as JWKS and health probes.
- Include pagination contracts for lists and request/trace/actor/tenant context where relevant.
- Run Buf lint, breaking checks, and generation. Generated code must be reproducible and have no diff after regeneration.

## Event rules

- Use NATS JetStream, not Core NATS, for domain events that drive state changes. JetStream persistence, explicit acknowledgement, and redelivery make consumers restart-safe.
- Subjects follow `platform.<domain>.<aggregate>.<event>.v1`.
- Wrap payloads in the shared event envelope with event ID, type, aggregate, tenant, schema version, timestamp, Request ID, Trace ID, and actor.
- Publishers set a message ID for deduplication. Consumers are durable, explicitly acknowledge only after success, and tolerate duplicate delivery.
- A database state change and its event must use a transactional outbox; do not publish directly inside a database transaction.
- Consumers store processed event IDs or implement a domain idempotency boundary before applying side effects.
- Propagate trace context in NATS headers and record publish/consume metrics.

## Shared library extraction

Move code to `platform-go` only when at least two services need the same stable behavior and the behavior has no domain ownership.

Good candidates:

- NATS connection lifecycle, JetStream stream provisioning, event envelope codec, outbox dispatcher, consumer runner
- Request/Trace/actor/tenant context propagation
- gRPC client interceptors, authentication credentials, error conversion
- frontend response envelope and pagination primitives
- test fixtures for cross-cutting infrastructure

Keep these local:

- SQL queries and migrations
- domain entities and validation
- service-specific HTTP request/response DTOs
- authorization policy semantics
- orchestration that calls named business services

Version shared modules semantically. A service pins a released version; local `go.work` replacements are development-only.

## Implementation sequence

1. Change the authoritative contract first when a public or inter-service interface changes.
2. Generate and verify SDKs.
3. Implement domain model and repository with migrations.
4. Implement application service and transaction/outbox boundary.
5. Add internal gRPC transport and frontend HTTP DTO/handler separately.
6. Add event consumers and cache/session invalidation reactions.
7. Wire dependencies through Uber Fx and preserve graceful shutdown.
8. Update all environment profiles, Compose, Kubernetes, docs, and CI.

## Verification gates

Every behavior change requires unit tests. Database, Redis, NATS, migrations, and the service's own adapters require integration tests built with Testcontainers and isolated behind the `integration` build tag.

A service repository is the unit of testing and maintenance. Its unit and integration suites must never require another platform or business service to run. Exercise outbound clients against in-process fakes, gRPC `bufconn`, or contract stubs. Platform-wide journeys belong to a separate `system-tests` project and never become a prerequisite for a service repository's test suite.

When integration or manual testing exposes a defect, first add the smallest deterministic unit regression test that reproduces the root cause. Keep an integration regression only when the behavior genuinely depends on infrastructure or service boundaries. This keeps the fast suite authoritative and lowers recurrence cost.

Run checks proportional to the changed scope. Before claiming a service feature complete, require:

```bash
go test -race ./...
go vet ./...
buf lint
buf generate
git diff --exit-code -- generated-paths
go test -tags=integration -count=1 ./integration/...
```

Also generate a fresh service when changing the template/generator. Within each service verify frontend API, gRPC status behavior, outbound client behavior against stubs, event redelivery/idempotency, migration up/down, and shared-database schema isolation. Run platform `system-tests` separately for multi-service journeys.

## Maintenance rule

When an implementation attempt fails for a reusable reason, add a concise entry to `references/lessons.md` containing symptom, root cause, and prevention. When a deliberate architecture choice changes, update `references/decisions.md`, the platform plan, and affected tests in the same change.
