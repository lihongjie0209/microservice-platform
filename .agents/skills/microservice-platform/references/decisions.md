# Architecture Decisions

## Repository model

- The workspace is a coordination directory; services, contracts, shared libraries, and delivery assets remain independently versioned repositories.
- Initial platform services are identity, tenant, and authorization.

## Interfaces

- Frontend clients use service-owned POST+JSON APIs with a uniform response envelope.
- Services use versioned gRPC contracts from `platform-protos` for synchronous calls.
- Domain events use NATS JetStream with versioned subjects and shared envelopes.

## Data

- Services never share tables.
- Online service-to-service data access uses APIs or events, never cross-schema SQL. Reporting/OLAP builds separate read models through CDC/events/exports; direct OLTP reads are exceptional, read-only, and architecture-reviewed.
- PostgreSQL is the default database. The initial services share database `platform` with per-service roles, schemas, and migration tables.
- Kingbase follows the same schema isolation model when selected.
- MySQL uses per-service database names.

## Shared code

- Stable cross-cutting Go code is extracted into `platform-go`.
- Domain code remains local even if structs look similar across services.

## Testing

- Every feature has unit coverage.
- Infrastructure and service-owned adapters use Testcontainers integration tests with the `integration` build tag.
- A service test suite never requires another service. Outbound service clients use in-process fakes/bufconn/contract stubs; multi-service journeys live in a separate platform `system-tests` project.
- Bugs found manually or in integration tests are reproduced in a unit test first whenever the root cause can be isolated.
