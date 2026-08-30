# Reusable Lessons

## 2026-08-29: verify toolchain before generation

- Symptom: `microgen` failed while generating three services because `buf` was absent.
- Root cause: the generator intentionally requires Buf when `--generate=true` and atomically removes incomplete output.
- Prevention: run `buf --version`, `protoc-gen-go --version`, and `protoc-gen-go-grpc --version` before batch generation. Pin installation versions in workspace bootstrap scripts and CI.

## 2026-08-29: database name must affect the connection, not just metadata

- Symptom: overriding a database-name field could disagree with the database embedded in the DSN.
- Root cause: the first implementation treated the name as descriptive configuration.
- Prevention: structurally apply database name and schema through driver configuration to both application and migration connections, then cover generated DSNs and integration behavior.

## 2026-08-29: Go proxy downloads can leave a tool installation incomplete

- Symptom: Buf/platform-go installation stopped with `unexpected EOF` while downloading `klauspost/compress`; an older Buf binary remained on PATH.
- Root cause: a transient module proxy transfer failure, not a source compilation failure.
- Prevention: verify the exact installed binary version after `go install`, retry with an alternate `GOPROXY`, and never infer installation success from partial download output.

## 2026-08-29: integration images must be explicitly reproducible

- Symptom: the NATS Testcontainers test reached container creation with no usable local image and failed with `No such image`.
- Root cause: relying on a moving minor tag and implicit pull behavior made registry resolution ambiguous in the local daemon.
- Prevention: pin an existing patch tag, request an explicit image pull in Testcontainers, and validate the same image reference used by local Compose.

## 2026-08-29: valid identifier syntax does not exclude SQL keywords

- Symptom: PostgreSQL initialization failed at `CREATE SCHEMA authorization AUTHORIZATION ...` even though the schema matched the configured identifier regex.
- Root cause: `AUTHORIZATION` is SQL grammar, so using the same word as an unquoted schema name made the statement ambiguous.
- Prevention: quote infrastructure-owned SQL identifiers even after allowlist validation, and execute initialization SQL against a real PostgreSQL Testcontainer before accepting it.

## 2026-08-29: concurrent `CREATE SCHEMA IF NOT EXISTS` is not race-free

- Symptom: parallel startup migrations intermittently failed on PostgreSQL's unique namespace index even with `IF NOT EXISTS`.
- Root cause: concurrent transactions can both pass the existence check before either commits.
- Prevention: serialize schema creation with a transaction-scoped PostgreSQL advisory lock, then test parallel startup against a real PostgreSQL container.

## 2026-08-29: context-transforming middleware must preserve context on errors

- Symptom: an authentication error path returned a nil context and downstream diagnostic/test code panicked while reading the principal.
- Root cause: the API treated the transformed context as meaningful only on success.
- Prevention: context-transforming functions return the original non-nil context on failure, and shared context readers defensively handle a nil interface.

## 2026-08-29: Makefile PATH assignments must survive spaces

- Symptom: contract generation under WSL failed with `/bin/sh: VS: not found` even though every required binary was installed.
- Root cause: a temporary `PATH=...` assignment expanded a Windows path containing `Microsoft VS Code` without shell quoting.
- Prevention: quote environment assignment values in Make recipes and invoke pinned tools through an explicit workspace tools directory.

## 2026-08-29: optional foreign keys are SQL NULL, not empty strings

- Symptom: creating a membership without a primary organization unit violated its foreign key because the Go zero string was inserted as `''`.
- Root cause: transport-friendly empty strings were passed directly into nullable persistence fields.
- Prevention: explicitly map empty optional IDs to SQL `NULL` at repository writes and normalize nullable reads at the repository boundary.

## 2026-08-29: MySQL index limits are measured in bytes

- Symptom: a valid-looking `(tenant_id, path)` unique index failed under utf8mb4 with `Specified key was too long`.
- Root cause: character lengths were added without multiplying by utf8mb4's maximum four bytes per character.
- Prevention: budget composite MySQL indexes in bytes, constrain only fields whose indexability is a real invariant, and exercise migrations on the supported MySQL image.

## 2026-08-29: application errors are translated exactly once

- Symptom: a deliberate invalid-argument rejection from organization cycle detection surfaced as an internal error.
- Root cause: a generic domain error translator wrapped an already normalized application error through its default branch.
- Prevention: translators first preserve recognized platform application errors, then map repository/domain errors, and only unknown failures become internal errors.

## 2026-08-29: PostgreSQL parameter types in special SQL syntax may need explicit casts

- Symptom: a parameterized `SUBSTRING(path FROM $2)` subtree move failed because pgx could not encode an integer into PostgreSQL's inferred text parameter.
- Root cause: the special `FROM` syntax did not provide stable parameter type inference through the prepared statement.
- Prevention: explicitly cast positional numeric parameters in dialect-specific SQL, isolate the dialect fragment, and verify it with both a unit contract and a real database integration test.

## 2026-08-29: validation scripts must avoid shell-reserved variable names

- Symptom: a Makefile verification command stopped with `read-only variable: status` under zsh.
- Root cause: the surrounding shell script reused zsh's reserved `status` parameter for an exit code.
- Prevention: use task-specific names such as `make_exit_code` in portable developer scripts and test commands under the workspace's declared shell.

## 2026-08-29: SQL insert columns and placeholders must be checked together

- Symptom: PostgreSQL reported `INSERT has more expressions than target columns` and MySQL reported a column/value count mismatch for the same repository method.
- Root cause: a hand-written insert listed ten columns but eleven placeholders, which compiled and passed mock-based service tests.
- Prevention: isolate non-trivial insert SQL, add a fast shape test that compares column and placeholder counts, and retain Testcontainers coverage on every supported database dialect.

## 2026-08-29: Swag model references use declared package names

- Symptom: Swagger generation could compile the handler but failed to resolve a response model referenced through a local import alias.
- Root cause: `swag` resolves annotation types using the package's declared name rather than the source file's import alias.
- Prevention: reference annotation models as `declaredpackage.Type`, then run `swagger-check` whenever frontend handlers or DTOs change.

## 2026-08-29: multi-command verification must fail fast

- Symptom: a contract generation command failed because its tool path was wrong, but later successful checks left the overall shell invocation with exit code zero.
- Root cause: the verification script did not enable fail-fast behavior and therefore reported only the final command's status.
- Prevention: start compound verification recipes with `set -e`, use the root Makefile to inject pinned tool paths, and treat every generation step as an explicit gate.

## 2026-08-29: shared Fx providers belong to infrastructure modules

- Symptom: the application dependency graph failed because two domain modules both provided the same database transactor type.
- Root cause: a shared infrastructure constructor was registered inside domain modules, making composition order and example-module removal unsafe.
- Prevention: provide database pools and transactors exactly once from the database module; domain modules provide only their repositories and services.

## 2026-08-29: generated templates need behavioral acceptance tests

- Symptom: a defect was fixed in an already generated service while newly generated services could still reproduce the same migration, principal-context, or dependency-injection bug.
- Root cause: validation covered the template repository but did not create and compile a fresh service from the local generator source.
- Prevention: after every template or generator change, generate into a temporary directory, assert critical transformed files and module paths, then run the generated project's full unit suite.

## 2026-08-29: Fx providers are lazy even inside infrastructure modules

- Symptom: an event bus provider compiled in the application graph but never connected or registered lifecycle cleanup.
- Root cause: `fx.Provide` constructors run only when another graph node consumes their result.
- Prevention: infrastructure runtimes that must start unconditionally include an explicit `fx.Invoke`, and graph plus lifecycle behavior is covered by tests.

## 2026-08-29: shared embedded persistence fields need explicit mapping tags

- Symptom: a complete SQL row existed and credential verification succeeded, but `sqlx` failed while scanning `created_at` and other audit columns into an embedded shared struct, causing login to appear as invalid credentials.
- Root cause: the shared audit struct relied on Go field-name inference, which maps `CreatedAt` differently from the SQL snake-case column.
- Prevention: shared persistence structs declare explicit `db` tags for every column, and repository tests scan a full real-shaped row rather than only testing writes.

## 2026-08-30: generated example domains must leave service ownership completely

- Symptom: tenant and authorization no longer registered example user routes, but their E2E suites and initial migrations still assumed those routes and tables existed.
- Root cause: runtime composition, migrations, generated documentation, and tests were cleaned independently instead of as one ownership boundary.
- Prevention: when replacing the template domain, remove it from Fx modules, HTTP/gRPC registration, migrations, OpenAPI annotations, and E2E journeys; assert fresh migrations do not create foreign-domain tables and non-identity services do not expose login.

## 2026-08-30: configured timezone must be enforced by the database driver

- Symptom: application and Cron configuration used `Asia/Shanghai`, while database sessions could still display timestamps in the server default timezone.
- Root cause: timezone was treated as a presentation-only YAML value rather than a connection-session invariant.
- Prevention: set PostgreSQL/Kingbase runtime parameters and MySQL DSN location in the connection factory, preserve explicit database names, and assert the effective session timezone with Testcontainers.

## 2026-08-30: identity token issuance has one owner

- Symptom: generated tenant and authorization services retained a local HMAC login endpoint, bypassing identity JWKS rotation and session controls.
- Root cause: template authentication demo code was carried into services that should only verify caller identity.
- Prevention: only identity-service issues user/service tokens; all other services use the shared EdDSA JWKS verifier, require issuer and audience in production, and test that `/auth/login` is absent without starting identity-service.

## 2026-08-30: E2E tests must exercise the owned domain

- Symptom: service integration suites stayed green until example user routes were removed, then failed on 404 responses unrelated to the service's actual domain.
- Root cause: copied scaffold journeys tested infrastructure through placeholder CRUD rather than through tenant or authorization operations.
- Prevention: each service E2E suite uses its own public and gRPC contracts, includes a negative ownership assertion, and promotes infrastructure defects such as missing idempotent replay into a focused unit regression.

## 2026-08-30: an event bus is shared only when its wire envelope is shared

- Symptom: the template's JSON event envelope and the platform SDK's protobuf envelope could use the same NATS subjects while being mutually undecodable.
- Root cause: broker compatibility was mistaken for contract compatibility.
- Prevention: platform services publish and consume `platform.common.v1.EventEnvelope` through `platform-go/eventbus`; service-local wrappers may add lifecycle wiring but must not define a second envelope format.

## 2026-08-30: high-growth partitioning must preserve uniqueness invariants

- Symptom: time partitioning proposals for audit and notification data conflicted with global event-id or tenant/idempotency uniqueness.
- Root cause: partition keys were selected from retention needs without checking PostgreSQL unique-index rules.
- Prevention: include the partition key in immutable audit identity where acceptable, use a default partition to prevent write loss, and delay/reshape partitioning when a global idempotency constraint is more important; document retention ownership in the migration.

## 2026-08-30: MySQL TEXT migrations need staged backfills

- Symptom: MySQL 8.4 rejected `ALTER TABLE ... ADD TEXT NOT NULL DEFAULT ''` while adding an ABAC condition to an existing table.
- Root cause: unlike ordinary VARCHAR columns, TEXT default support is restricted and version/mode dependent.
- Prevention: preserve TEXT for unbounded domain strings, add the column nullable, backfill existing rows, then alter it to NOT NULL; verify the upgrade migration on the supported MySQL Testcontainer.
## Generated consistency checks must compare pre/post generation, not the Git baseline

- Symptom: `contracts-check` and `swagger-check` fail for correct, newly generated files whenever the repository has intentional uncommitted work.
- Root cause: the target used `git diff --exit-code`, which compares against `HEAD` rather than checking whether generation itself changes the working tree.
- Prevention: copy the generated directory to a temporary directory, run the pinned generator, and compare the two directories. This remains deterministic in dirty worktrees and in CI.

## Linter configuration and binary major versions are one contract

- Symptom: every lint job fails before analysis with “configuration file for golangci-lint v2 with golangci-lint v1”.
- Root cause: the template emitted a v2 configuration but pinned a v1 binary in CI and had no pinned local bootstrap target.
- Prevention: pin the same v2 release in CI and the workspace Makefile, and run the template lint gate before generating services.

## Public module names must be resolved before rewriting imports

- Symptom: the desired shared SDK repository name already belonged to an unrelated private project.
- Root cause: local module naming was chosen before checking the remote namespace and repository ownership.
- Prevention: inspect the exact GitHub repository before publishing, never repurpose an unrelated repository, choose a collision-free module path, and migrate Go imports with an AST tool.

## A release tag is immutable only after remote-shaped verification

- Symptom: a shared SDK tag was created after a shell sequence continued past a failed test compilation.
- Root cause: the release command did not use fail-fast chaining and validation did not consume the published dependency as an external module.
- Prevention: release recipes use `set -e` or `&&`, run unit/race/vet before tagging, then verify a clean consumer against the published version; publish a new patch version instead of moving a bad tag.

## 2026-08-30: non-root images must provision writable runtime paths

- Symptom: services passed unit tests and built successfully but restarted in Compose because the non-root user could not create `/app/logs`.
- Root cause: the image copied read-only application assets but did not create and transfer ownership of configured writable directories before switching users.
- Prevention: create every local runtime path during image build, assign it to the runtime user, and include a real container startup journey in platform CI.

## 2026-08-30: database init directories do not reconcile existing volumes

- Symptom: early services connected to a shared PostgreSQL volume while later services failed password authentication even though their roles were present in the current init SQL.
- Root cause: the official PostgreSQL entrypoint runs `/docker-entrypoint-initdb.d` only when initializing an empty data directory.
- Prevention: keep first-boot SQL idempotent and run a non-destructive bootstrap job on every development platform start to reconcile roles, passwords, schemas, ownership, grants, timezone, and search paths.

## 2026-08-30: JetStream subject ownership is exclusive across streams

- Symptom: independently healthy services failed startup with NATS error 10065 because each tried to provision a differently named stream covering `platform.>`.
- Root cause: JetStream does not allow overlapping subjects in separate streams; shared broker subjects were configured without a single stream owner/name.
- Prevention: assign all platform domain subjects to one consistently named stream, let provisioning remain idempotent, and isolate consumers with durable names rather than overlapping streams.

## 2026-08-30: skipped authentication does not create an audit actor

- Symptom: an unauthenticated bootstrap registration reached the handler but failed with “authenticated actor is required”.
- Root cause: bypassing authentication intentionally leaves principal context empty while the service correctly requires actor attribution for persisted mutations.
- Prevention: bootstrap mutations use a narrowly scoped development PSK or a controlled administrative identity; never weaken the global audit invariant to make a public route work.

## 2026-08-30: Swagger generation warnings can hide incomplete schemas

- Symptom: `swag` exited successfully while reducing a request containing `json.RawMessage` to an empty object definition.
- Root cause: the generator could not infer the standard-library alias as an OpenAPI model but treated the parse error as a warning.
- Prevention: annotate opaque JSON fields with `swaggertype:"object"`, inspect generator output, and retain generated-document consistency checks.

## 2026-08-30: platform services generated from a standalone template must drop local token issuance

- Symptom: a newly generated platform service compiled with the template's local HMAC issuer but could not authenticate Identity's EdDSA multi-audience tokens.
- Root cause: the standalone template intentionally supports self-contained development, while platform services have a stricter single-issuer architecture.
- Prevention: after generation, non-identity platform services replace local issuance at runtime with the shared JWKS verifier, add their audience to Identity, and test a real cross-service token before publication.

## 2026-08-30: dynamic gRPC reflection requires stream authentication propagation

- Symptom: unary business calls carried PSK/JWT metadata, but descriptor discovery failed authentication because Server Reflection is a bidirectional streaming RPC.
- Root cause: the outbound client configured only unary metadata interceptors.
- Prevention: authentication metadata is attached to unary and stream client calls; Reflection is exposed only on an internal protected endpoint and included explicitly in its auth policy.

## 2026-08-30: build metadata must handle a repository without a first commit

- Symptom: the first build of a generated service passed `HEAD unknown` as one linker value and failed before the initial commit existed.
- Root cause: plain `git rev-parse HEAD` can print `HEAD` before returning failure, after which the shell fallback appends `unknown`.
- Prevention: resolve build commits with `git rev-parse --verify HEAD` so an unborn repository produces exactly the fallback value.

## 2026-08-30: Kubernetes discovery and document access are separate security decisions

- Symptom: an annotated Service appeared in the Swagger catalog but its production document endpoint was disabled or blocked by cross-namespace NetworkPolicy.
- Root cause: Service list/watch proves discoverability only; it does not grant workload network access or application authentication.
- Prevention: enable the protected internal document endpoint, forward a valid multi-audience caller token, allow only swagger-service ingress on the document port, and test discovery plus document retrieval together.

## 2026-08-30: template-only markers must be removed as complete lines

- Symptom: a generated GitHub Actions step was indented into the preceding multiline shell command and the workflow became invalid YAML.
- Root cause: the generator removed marker text starting at the marker token, but preserved whitespace before an indented marker line.
- Prevention: when a marker occupies a whitespace-only line, remove the whole start/end marker lines and enclosed block; retain an exact regression for the generated YAML boundary.

## 2026-08-30: configuration wrappers must mirror coupled runtime invariants

- Symptom: the service configuration accepted a JetStream ack wait shorter than the platform consumer handler timeout, then failed only while starting the consumer.
- Root cause: the service-local wrapper exposed only part of the shared SDK options and therefore could not validate their relationship.
- Prevention: mirror every coupled SDK option at the configuration boundary, validate cross-field invariants before Fx startup, and cover the invalid combination with a unit test.

## 2026-08-30: PostgreSQL text preference is not a portable MySQL indexing rule

- Symptom: an application-service migration passed PostgreSQL but MySQL rejected `TEXT DEFAULT ''`; indexed and unique `TEXT` columns would also require prefix lengths.
- Root cause: PostgreSQL-oriented type guidance was copied verbatim into a MySQL physical schema.
- Prevention: keep PostgreSQL/Kingbase domain strings as `TEXT`; in MySQL use bounded `VARCHAR` for identifiers, foreign keys, unique keys, and indexed values, leave payload/display text as `TEXT` without defaults, and run both database migration suites before publishing.

## 2026-08-30: logical provider registration needs replica coordination

- Symptom: multiple replicas of one provider would repeatedly rotate the same service-level lease token, causing otherwise healthy replicas to invalidate each other's heartbeats.
- Root cause: the registry row identifies a logical service and stable Kubernetes DNS target, while lifecycle hooks execute independently in every Pod.
- Prevention: elect one registration coordinator per logical service with the shared ownership-safe Redis lock, renew that leadership separately from the provider lease, and stop/unregister immediately after leadership is lost.

## 2026-08-30: protobuf responses must not be copied by value

- Symptom: provider client tests passed but `go vet` rejected assignments such as `*response = *value` because generated messages contain an internal mutex.
- Root cause: an outbound wrapper treated generated Protobuf messages as ordinary value structs.
- Prevention: keep Protobuf messages behind pointers and transfer content with `proto.Reset` plus `proto.Merge` (or return the allocated pointer directly); retain `go vet` as a required service gate.

## 2026-08-30: metadata discovery must support unknown service names

- Symptom: a dictionary gateway could discover instances only after it already knew every provider service name.
- Root cause: the first registry list/watch contract required an exact service name, which made cross-domain capability discovery circular.
- Prevention: allow an empty service name only when a non-empty metadata selector is supplied, test cross-service selection, and keep domain-specific metadata interpretation in the consuming service.

## 2026-08-30: Helm library defaults do not become consumer root values

- Symptom: a consumer that intentionally omitted an optional gateway block failed while rendering a library named template with a nil-value access.
- Root cause: a Helm library chart's `values.yaml` is not merged into the parent chart's root `.Values` passed to an included named template.
- Prevention: named library templates default the whole optional map with `default dict`, default individual fields locally, and test both enabled and completely omitted configurations from a real consumer chart.

## 2026-08-30: pre-install migration Jobs cannot consume same-release configuration

- Symptom: a migration Job looked ordered before Deployment but could fail on first install because its referenced ConfigMap or ExternalSecret target did not exist yet.
- Root cause: Helm pre-install hooks run before ordinary resources in the same release, and creation of an ExternalSecret does not synchronously create its target Secret.
- Prevention: default Kubernetes startup migration to an init container that runs after Pod configuration resolves and before the API container; retain a hook Job only when configuration and secrets are independently pre-provisioned.

## 2026-08-30: a new durable consumer may replay matching stream history

- Symptom: a dead-letter integration scenario consumed an earlier event before the event published specifically for that scenario.
- Root cause: the JetStream consumer retained the default all-available delivery policy, so a newly created durable correctly started from matching stream history rather than only future messages.
- Prevention: preserve replay-friendly production defaults; isolate independent tests with distinct subjects or explicitly configure a start policy only when that policy is part of the behavior under test.

## 2026-08-30: Compose validation does not prove image tags are runnable

- Symptom: `docker compose config` passed, but the development stack stopped immediately because a syntactically valid Temporal image tag did not exist.
- Root cause: Compose schema validation does not resolve image manifests or exercise container health checks.
- Prevention: pin a published image tag, then require an actual `docker compose up --wait` smoke test for newly added infrastructure; treat registry transport failures separately from an invalid tag.

## 2026-08-31: shell loop failure propagation must be explicit

- Symptom: the workspace `make verify` returned success even though early per-service Swagger commands failed, because later loop iterations succeeded.
- Root cause: relying on `set -e` inside a POSIX shell loop did not reliably terminate the recipe for a failed command in the loop body.
- Prevention: append `|| exit $?` to every per-repository loop body and retain a regression command that substitutes one guaranteed-failing child target.
