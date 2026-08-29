# Architecture Decisions

## Repository model

- The workspace is a coordination directory; services, contracts, shared libraries, and delivery assets remain independently versioned repositories.
- Initial platform services are identity, tenant, and authorization.

## Interfaces

- Frontend clients use service-owned POST+JSON APIs with a uniform response envelope.
- Services use versioned gRPC contracts from `platform-protos` for synchronous calls.
- Domain events use NATS JetStream with versioned subjects and shared envelopes.
- All `platform.>` domain subjects belong to the single `PLATFORM_EVENTS` JetStream stream. Services may idempotently provision the same stream; independent processing is represented by durable consumer names, never overlapping streams.
- The only platform event wire format is protobuf `platform.common.v1.EventEnvelope`; JSON broker envelopes are not allowed on `platform.>` subjects.
- `identity-service` is the only platform token issuer. Other services verify EdDSA access tokens from its JWKS endpoint through `platform-go`, validating issuer and service-specific audience; they never expose credential login endpoints.

## Data

- Services never share tables.
- Online service-to-service data access uses APIs or events, never cross-schema SQL. Reporting/OLAP builds separate read models through CDC/events/exports; direct OLTP reads are exceptional, read-only, and architecture-reviewed.
- PostgreSQL is the default database. The initial services share database `platform` with per-service roles, schemas, and migration tables.
- Kingbase follows the same schema isolation model when selected.
- MySQL uses per-service database names.
- All mutable persistent tables use `version BIGINT NOT NULL DEFAULT 1` and the four platform audit columns: `created_at`, `updated_at`, `created_by`, and `updated_by`. Actor attribution is passed explicitly from shared authenticated context through application and repository layers.
- Updates and soft deletes use expected-version optimistic concurrency. Distributed locks are reserved for cross-resource/cross-system invariants and use the narrowest stable resource key.
- PostgreSQL strings default to `TEXT`; bounded types are reserved for genuine domain constraints. Time instants use `TIMESTAMPTZ`, with database/connection presentation set to `Asia/Shanghai` (UTC+08:00).
- High-growth tables have explicit partition, retention, and archive policies. Service migrations own declarative partitioning; deployment/DBA automation may use `pg_partman` without making it mandatory for service startup.

## Shared code

- Stable cross-cutting Go code is extracted into `platform-go`.
- Domain code remains local even if structs look similar across services.
- Shared authentication/authorization interceptors, principal/audit context, and Redis distributed locking live in `platform-go`; services provide policy/verifier implementations and domain authorization requirements.
- Global error codes and transport mappings live in `platform-go`; domain services may define messages/details but must not allocate duplicate numeric codes locally.

## Testing

- Every feature has unit coverage.
- Infrastructure and service-owned adapters use Testcontainers integration tests with the `integration` build tag.
- A service test suite never requires another service. Outbound service clients use in-process fakes/bufconn/contract stubs; multi-service journeys live in a separate platform `system-tests` project.
- Bugs found manually or in integration tests are reproduced in a unit test first whenever the root cause can be isolated.
