# Reusable Lessons

## 2026-09-02: POST-only business APIs still require gateway GET and OPTIONS methods

- Symptom: service handlers correctly used POST+JSON, but APISIX allowed only POST, so browsers could not complete CORS preflight and standard JWKS, health, Swagger document, and Swagger asset GET requests never reached their handlers.
- Root cause: the business API verb convention was applied to the entire external protocol surface, including standards-mandated discovery and browser transport requests.
- Prevention: gateway method allowlists include GET, POST, and OPTIONS; business routers still expose only POST for domain APIs, while GET remains limited to explicit standards endpoints and OPTIONS is handled by the service CORS middleware. Assert all three verbs in Helm render tests.

## 2026-09-02: console development endpoints must track the authoritative Compose ports

- Symptom: the application-split console compiled and its API adapters were correct, but the default development runtime configuration sent Identity, Tenant, Authorization, and Application requests to obsolete ports and omitted Swagger entirely.
- Root cause: the frontend runtime template checked environment-variable coverage, while the concrete development configuration was neither checked for every service key nor compared when Compose ports changed.
- Prevention: treat Compose as the local endpoint authority, update `public/platform-config.js` with port changes, require every typed platform service key in both the development config and container template, and run `pnpm check:runtime-config` in frontend CI. Platform CI also checks out the independently versioned console and runs `make console-endpoint-invariants-check` to compare all 20 concrete development ports with Compose.

## 2026-09-02: service control-plane authentication must fail closed when PSK routing is missing

- Symptom: dictionary Provider lifecycle RPCs were excluded from member RBAC because they were designed for PSK callers, but a missing PSK route pattern caused generic authentication to accept an ordinary user JWT instead.
- Root cause: authorization exclusions expressed the expected credential class only as an operational configuration assumption; the transport had a permissive authentication fallback.
- Prevention: mark service-only control-plane methods explicitly, require that their configured PSK wildcard matches before credential verification, reject rather than fall back when it does not, and retain HTTP/gRPC unit regressions using an otherwise valid user JWT. Keep human-readable global catalogs separately protected with `ScopePlatform`.

## 2026-09-02: authorization scope zero values must never decide ownership implicitly

- Symptom: global application catalog, menu, and tenant-grant management requirements omitted `Scope`, so Go's zero value silently evaluated them in the selected tenant namespace and a tenant wildcard could expose platform operations.
- Root cause: resource/action coverage tests asserted only that a requirement existed, not that its authorization namespace matched domain ownership; one list endpoint was also reused for tenant consumption and platform administration.
- Prevention: set `ScopePlatform`, `ScopePrincipal`, or an intentional `ScopeTenant` explicitly for every protected route, test the scope classification, and split tenant self-service reads from platform management APIs whenever they require different subjects or target-tenant rules. Run the AST-based `make authorization-invariants-check` in platform CI so inferred map-element literals cannot silently omit the field.

## 2026-09-02: a tenant-scoped RBAC decision does not validate request resource ownership

- Symptom: tenant management endpoints authorized the caller's selected membership but then accepted another `tenant_id` or an ID belonging to another tenant in the request, allowing the decision and mutation target to diverge.
- Root cause: transport authorization and domain ownership were treated as the same check; ID-based methods did not load the persisted tenant before mutating.
- Prevention: tenant domain methods compare explicit tenant IDs with the trusted JWT claim, load ID-based resources and validate their persisted tenant, allow cross-tenant orchestration only for separately authorized service/system or platform-marked calls, and keep deterministic cross-tenant unit regressions.

## 2026-09-01: frontend JSON summaries need transport-specific DTOs

- Symptom: audit before/after JSON appeared as base64 strings in the frontend even though the stored payload was valid JSON.
- Root cause: the HTTP response exposed the domain model's `[]byte`; Go JSON encoding correctly treats byte slices as base64 rather than embedded JSON.
- Prevention: define explicit HTTP response DTOs, expose structured JSON as `json.RawMessage` with an OpenAPI object annotation, and assert the marshaled response shape in a unit test.

## 2026-09-01: authenticated requests still require server-side tenant binding

- Symptom: an authenticated audit caller could submit another `tenant_id` in a query body and read that tenant's records.
- Root cause: an early local principal model retained only subject and authentication method, discarding the trusted tenant claim returned by the shared JWKS verifier.
- Prevention: use the shared platform principal end to end, compare user-scoped requests with its trusted `tenant_id` in the application layer for every transport, and reserve tenant-independent access for explicitly authorized service/system principals.

## 2026-09-01: revoking refresh state does not invalidate a signed access token by itself

- Symptom: an administrator revoked a user session, while the already issued JWT could still call Identity APIs until its expiration time.
- Root cause: bearer authentication verified only the token signature and claims; it never checked the session ID against revocation state.
- Prevention: Identity validates user-token session state after signature verification and tests a revoked-token request end to end. Other services use short-lived access tokens plus a bounded session-status cache invalidated by Identity revocation events; service-account tokens bypass user-session lookup.

## 2026-08-31: environment list syntax must be normalized before wildcard validation

- Symptom: a Compose service repeatedly failed startup because a PSK gRPC method was validated as `"[/package.Service/*"` instead of `"/package.Service/*"`.
- Root cause: Viper split a bracketed environment list on commas but preserved the leading and trailing brackets as data.
- Prevention: configuration decoding normalizes both comma-separated and bracketed string lists before validation; retain a unit regression using the exact Compose representation and verify a freshly generated service inherits it.

## 2026-08-31: GitHub CLI repository inference is ambiguous with multiple remotes

- Symptom: `gh run list` reported no workflows for a repository whose pushed commit already had a successful CI run.
- Root cause: the checkout had both `origin` and `upstream`, and GitHub CLI inferred the upstream repository instead of the push target.
- Prevention: CI inspection commands always pass `--repo <owner/repository>` when a checkout has multiple GitHub remotes; verify the selected repository with `gh repo view <owner/repository>` before concluding Actions is absent.

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
## Migration version numbers are repository-global within a dialect

- Symptom: Testcontainers and startup migration failed before executing SQL with `duplicate migration file`, while unit tests and integration compile-only checks passed.
- Root cause: a new migration reused an existing numeric prefix under a different descriptive filename; `golang-migrate` keys files by version and direction, not by the full name.
- Prevention: inspect the complete target migration directory before choosing a version and keep a fast unit test that asserts exactly one up/down pair per version for every supported dialect.

## Generated consistency checks must compare pre/post generation, not the Git baseline

- Symptom: `contracts-check` and `swagger-check` fail for correct, newly generated files whenever the repository has intentional uncommitted work.
- Root cause: the target used `git diff --exit-code`, which compares against `HEAD` rather than checking whether generation itself changes the working tree.
- Prevention: copy the generated directory to a temporary directory, run the pinned generator, and compare the two directories. This remains deterministic in dirty worktrees and in CI.

## Linter configuration and binary major versions are one contract

- Symptom: every lint job fails before analysis with “configuration file for golangci-lint v2 with golangci-lint v1”.
- Root cause: the template emitted a v2 configuration but pinned a v1 binary in CI and had no pinned local bootstrap target.
- Prevention: pin the same v2 release in CI and the workspace Makefile, and run the template lint gate before generating services.

## A repository move does not necessarily change its Go module path

- Symptom: installing CEL from its new `cel-expr/cel-go` repository path failed because release `v0.30.0` still declares `github.com/google/cel-go` in `go.mod`.
- Root cause: the source repository moved before the published Go module path changed, and repository URLs were treated as module identifiers.
- Prevention: resolve dependencies from their declared `go.mod` path and verify a release with `go mod download`/`go list -m`; use `github.com/google/cel-go` until upstream publishes a module-path migration.

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

## 2026-08-31: do not use zsh special path parameters as loop variables

- Symptom: inspection commands unexpectedly lost utilities such as `rg`, `sed`, and `base64` after a loop assigned to `path`.
- Root cause: zsh exposes `path` as the tied array form of `PATH`, so a harmless-looking loop variable mutates executable lookup for the rest of that shell.
- Prevention: use task-specific names such as `service_dir` or `target_file` in shell loops; avoid `path`, `status`, and other shell-special parameter names.

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

## 2026-09-01: branch-based raw contract URLs can remain stale after a push

- Symptom: the frontend contract generator completed successfully immediately after a backend Swagger push but omitted the newly added path.
- Root cause: the raw GitHub branch URL returned a cached prior representation even though the branch ref already pointed at the new commit; a timestamp query alone did not reliably select the new representation.
- Prevention: when consuming a contract immediately after publication, pin the source URL to the verified backend commit revision (or an immutable release artifact), regenerate, and assert the expected path exists before implementing the client.

## 2026-09-01: adding a regression test must preserve the existing test file

- Symptom: a new transport regression test replaced an existing global-error-code regression in the same `_test.go` path.
- Root cause: the target test path was created without first reading the repository's tracked file at that exact path.
- Prevention: resolve and inspect existing test files before `Add File`; extend the existing suite or choose a new descriptive filename, then review the complete diff to ensure prior assertions remain present.

## 2026-09-01: JSON transport migrations need an explicit compatibility window

- Symptom: changing a request field from a JSON-encoded string to structured JSON made existing E2E clients receive an invalid-request response even though the new DTO compiled and unit tests passed.
- Root cause: `json.RawMessage` also accepts a JSON string token, which was forwarded with its quotes to domain validation instead of being normalized from the legacy representation.
- Prevention: when replacing JSON-in-string fields, accept and unwrap the legacy string form at the transport boundary, always emit structured JSON, add a focused legacy-request unit regression, and remove compatibility only after clients have migrated.

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

## 2026-08-31: verification commands must not silently format authoritative contracts

- Symptom: `make verify` changed a committed Proto file while reporting a successful contract check.
- Root cause: the check target depended on the mutating `buf format -w` target and verified generated Go only.
- Prevention: generation may format explicitly, but checks use `buf format --diff --exit-code`; CI validates both Proto formatting and generated-code drift.

## 2026-08-31: replay-safe search projections must treat stale external versions as success

- Symptom: a delayed index event received OpenSearch `409 Conflict`, was retried until MaxDeliver, and could be dead-lettered even though a newer projection was already correct.
- Root cause: external version conflict was handled as infrastructure unavailability instead of the expected result of monotonic projection ordering.
- Prevention: use `source_version` with `external_gte`, acknowledge stale/equal conflicts as idempotent success, and retain a unit regression for the 409 path.

## 2026-08-31: index bootstrap is a concurrent replica operation

- Symptom: two fresh replicas both observed a missing index; one created it and the other failed startup on the create response.
- Root cause: index existence check followed by creation is inherently racy.
- Prevention: after a create conflict/bad-request, re-check index existence and succeed only when the expected index now exists; never assume startup hooks are singletons.

## 2026-08-31: shared JetStream provisioning requires identical subjects

- Symptom: transactional Outbox publishers repeatedly received `nats: no response from stream` after another service started.
- Root cause: every service idempotently reconciled the shared `PLATFORM_EVENTS` stream, but a generated default still used `service.>` and replaced the authoritative `platform.>` subject set.
- Prevention: every participant in the shared stream configures the identical `PLATFORM_EVENTS` / `platform.>` pair in both file and code defaults; keep it in the generator, enforce a workspace static invariant, and verify a multi-service restart against a persistent JetStream volume.

## 2026-08-31: inbound authentication metadata is not outbound metadata

- Symptom: search requests passed local JWT authentication but authorization-service rejected role lookup with `missing bearer token`.
- Root cause: gRPC does not automatically copy an incoming credential to an outbound RPC, and HTTP request headers are not gRPC metadata.
- Prevention: explicitly forward the already verified caller credential only at the authorization boundary; never make generic outbound clients forward credentials implicitly, and retain HTTP and gRPC context regressions.

## 2026-08-31: outbound pagination must honor the provider contract

- Symptom: authenticated search queries still failed because authorization-service rejected `page_size=200` while its maximum is 100.
- Root cause: the consumer chose a private batch size instead of the provider's published pagination invariant.
- Prevention: bind outbound pagination to the shared/provider contract limit and assert the exact request page size in the client unit test.

## 2026-08-31: platform CI must check out every Compose build context

- Symptom: local `make system-test` passed, but GitHub CI failed before building because `services/search-service` did not exist.
- Root cause: the coordination repository excludes independent service checkouts and the CI clone allow-list was not extended with the new Compose service.
- Prevention: add every new service to the CI checkout list in the same platform integration commit and keep Compose validation after checkout.

## 2026-08-31: generated service examples must not survive domain conversion

- Symptom: a newly generated domain service compiled while still documenting and testing local Hello/User Proto, principal, event-bus, and configuration examples alongside the real central contract and shared SDK.
- Root cause: scaffold generation establishes a runnable baseline, but successful compilation does not prove template-only artifacts were replaced or that the service follows platform ownership boundaries.
- Prevention: before the first service commit, audit imports, routes, generated code, docs, CI targets, configuration sections, and tests for template domain names; remove local Proto and cross-cutting duplicates, depend on the pinned central contract/shared SDK, enforce the workspace contract-ownership check, then run vet, lint, Swagger drift, and service integration tests.

## 2026-08-31: bound Compose build concurrency on hosted runners

- Symptom: platform CI was terminated while many Go image builds remained in compiler steps, despite no functional assertion failure.
- Root cause: Compose started every independent service image build concurrently, exhausting the hosted runner's memory/CPU and leaving builds stalled until runner shutdown.
- Prevention: `COMPOSE_PARALLEL_LIMIT` alone is insufficient when Compose submits one BuildKit graph. Install a CI BuildKit builder with a small worker `max-parallelism`, retain the Compose limit for engine operations, and keep failure log collection enabled.

## 2026-08-31: service integration tests must stop at outbound policy boundaries

- Symptom: billing's isolated Testcontainers suite authenticated a PSK caller, then attempted an authorization-protected plan mutation and failed because authorization-service was intentionally not running.
- Root cause: the test conflated inbound authentication, local domain transport, and an external authorization decision, violating service-owned test isolation.
- Prevention: use an unprotected domain read to prove JWT/PSK reaches the service, cover the shared authorization resolver with an in-process unit fake, and test authorization-service's decision engine in its own repository; only platform system journeys run both services.

## 2026-08-31: long-running JetStream handlers need matching acknowledgement deadlines

- Symptom: an asynchronous export worker was designed for multi-hour jobs while the shared consumer default canceled every handler after 25 seconds.
- Root cause: the business job timeout and JetStream `AckWait`/handler timeout were configured independently.
- Prevention: set the consumer handler timeout from the bounded job timeout and require `AckWait` to be strictly longer; retain database state as the authoritative claim so redelivery remains safe.

## 2026-08-31: central-contract services must remove scaffold-local Proto CI

- Symptom: a generated service still ran Buf breaking/generation checks for deleted Hello Proto files after adopting the versioned central contract module.
- Root cause: domain conversion removed runtime imports but not generated artifacts, Make targets, CI steps, integration tests, and documentation as one unit.
- Prevention: when a service adopts `platform-protos`, delete local example Proto/generated files and their Buf CI, pin the released central module, and compile the integration-tag suite before the first commit.

## 2026-08-31: unary client interceptors do not cover streaming RPCs

- Symptom: unary gRPC calls authenticated successfully, while server-streaming and bidirectional provider calls arrived without JWT/PSK, request ID, or idempotency metadata.
- Root cause: `grpc.WithChainUnaryInterceptor` is never invoked for streaming RPCs.
- Prevention: install a stream client interceptor that shares the same metadata builder as the unary interceptor; retain a deterministic unit test for authentication and correlation metadata in the template and every streaming consumer.

## 2026-08-31: PostgreSQL initialization readiness must require TCP

- Symptom: Compose marked PostgreSQL healthy, then the reconciliation bootstrap immediately failed with a TCP connection refusal.
- Root cause: the image's temporary initialization server accepts Unix-socket probes before the final TCP listener starts, and `pg_isready` without `-h` probes that socket.
- Prevention: health checks used by dependent containers explicitly probe `127.0.0.1`; retain a static Compose invariant check because schema validation alone cannot detect this startup race.

## 2026-08-31: streaming retries must replay the current idempotent batch

- Symptom: a Provider stream returned `Unavailable` after receiving or committing a batch, and the import job became terminal even though discovery had another healthy instance.
- Root cause: unary retry interceptors do not retry bidirectional stream creation or `Send`/`Recv`, while reopening a stream without replaying the unanswered batch silently loses work.
- Prevention: bound stream-open and current-batch retries with context-aware backoff, report the failed instance to discovery, recreate the stream, and replay Apply with the exact same durable idempotency key; unit-test validation replay and the commit-then-fail Apply case, then exercise it in the isolated Testcontainers suite.

## 2026-08-31: unused infrastructure packages are still maintenance forks

- Symptom: services ran through the shared event SDK but still compiled and tested an unreferenced private NATS/Envelope implementation inherited from the scaffold.
- Root cause: runtime wiring was migrated without deleting the old package, so normal Go tests stayed green while two implementations remained available to future code.
- Prevention: remove superseded packages with their modules and tests, forbid direct NATS imports below services, and make the workspace event-reliability check require shared SDK, transactional Outbox/Inbox, and replay boundaries.

## 2026-08-31: deleting a direct adapter changes module classification

- Symptom: unit and vet passed after removing a private NATS adapter, but CI's `go mod tidy && git diff` failed because `nats.go` moved from direct to indirect through the shared event SDK.
- Root cause: source deletion changed the module graph classification, and local verification omitted the repository's module-tidiness gate.
- Prevention: run `go mod tidy` after adding or deleting imports/packages, review only `go.mod`/`go.sum` classification changes, and commit them with the source change before pushing.

## 2026-08-31: SQL migration checks must parse statements, not lines

- Symptom: an audit-field check reported a valid compact `CREATE TABLE` as missing every shared column because several columns appeared on one line.
- Root cause: the checker assumed one column declaration per line; an earlier regular expression also stopped at parentheses inside SQL types or constraints.
- Prevention: split migrations into complete semicolon-terminated statements, locate `CREATE TABLE` declarations independently of formatting, and match columns at statement start or comma boundaries; retain compact and multiline unit fixtures.

## 2026-08-31: applied migrations are immutable release history

- Symptom: a new retention index was initially added to a service's baseline migration, which would work only for fresh databases and leave already migrated environments without the index.
- Root cause: a schema optimization was treated as scaffold state instead of an incremental production change.
- Prevention: never edit an applied migration; add a numbered forward migration and matching rollback for every supported dialect, then validate both fresh migration order and upgrade paths in CI.

## 2026-08-31: container workflows must name non-root Dockerfiles

- Symptom: frontend verification passed, but the container CI job failed immediately with `open Dockerfile: no such file or directory`.
- Root cause: the repository keeps its image definition at `docker/Dockerfile`, while the build action defaulted to `./Dockerfile`.
- Prevention: set the build action's `file` input explicitly whenever the Dockerfile is not at the build-context root, and retain the image build as a separate CI gate.

## 2026-08-31: frontend pages must not compensate for missing owner APIs

- Symptom: the planned platform user-management page had no pageable Identity query contract, tempting the console to omit the page or derive users indirectly from another service.
- Root cause: page planning was initially constrained to currently generated frontend contracts instead of checking whether the owning service exposed the required platform capability.
- Prevention: when an application page needs an owned resource, add the capability to the owning service's repository/application/HTTP layers and central gRPC contract, cover it with service-local unit tests, publish the contract version, and regenerate the frontend OpenAPI model; never read another service's schema or invent client-only data joins.

## 2026-08-31: remote contract generation needs cache bypass and bounded retry

- Symptom: frontend contract drift checks alternated between stale and current Swagger content immediately after a service push, and transient GitHub Raw TLS resets failed otherwise valid checks.
- Root cause: mutable branch URLs were cached between generation runs and the downloader attempted each source only once.
- Prevention: add a per-generation cache-busting query parameter and bounded exponential retries to remote contract downloads; keep the generated artifacts committed so review still shows the exact contract change.

## 2026-08-31: frontend pure logic must not import browser request graphs

- Symptom: a Node unit test for JSON-object validation failed while loading an unrelated Vue layout because the helper lived in an API module that imports the browser request and routing graph.
- Root cause: deterministic transformation logic and side-effectful transport wiring shared one module boundary.
- Prevention: keep parsers, tree builders, validators, and state reducers in side-effect-free modules; let API adapters import those helpers, and point Node unit tests only at the pure module.

## 2026-08-31: frontend contract artifacts travel with backend API changes

- Symptom: frontend typecheck, lint, tests, and production build passed, but CI failed the contract drift gate after an authorization endpoint was added.
- Root cause: the backend Swagger changed after the frontend feature commit, while the committed OpenAPI snapshot and generated declaration were not refreshed in the same delivery chain.
- Prevention: after every consumed backend HTTP contract change reaches its source branch, run frontend `generate:contracts` and `check:contracts` before committing the dependent page; treat the generated snapshot as part of the feature, not as a later documentation task.

## 2026-08-31: foreign keys do not enforce tenant ownership

- Symptom: membership create/update could reference an organization unit owned by another tenant because the database only verified that the organization ID existed.
- Root cause: referential integrity was mistaken for the domain invariant that both resources must share a tenant and that the referenced organization must be active.
- Prevention: before persisting any tenant-scoped cross-resource reference, load it through the owning repository and validate tenant and lifecycle state in the application layer; retain a unit regression using a valid foreign ID from a different tenant.

## 2026-08-31: soft-removed unique assignments must be reactivated

- Symptom: re-adding a removed role permission or group member failed on the unique pair constraint even though the UI correctly presented the resource as unassigned.
- Root cause: removal retained the junction row for audit and optimistic locking, while the add path always attempted a new insert.
- Prevention: for audited unique junctions, add means “ensure active”: return an already active row idempotently, reactivate a removed row with its current version, and insert only when the pair does not exist; cover both active and removed cases with deterministic unit tests.

## 2026-09-01: Swagger response DTOs must not embed cross-package domain models

- Symptom: Go tests passed after introducing a structured REST view, but `swag init` failed with `cannot find type definition` for an embedded domain entity imported under an alias.
- Root cause: the transport DTO reused a database/domain struct through cross-package embedding; the Swagger parser could not resolve it, and the design also leaked persistence fields into the public contract implicitly.
- Prevention: define explicit service-local HTTP response DTO fields and map domain entities deliberately; run Swagger generation before the full test gate and retain structured-JSON transport regressions.

## 2026-09-01: frontend runtime injection must cover every registered service

- Symptom: development configuration and generated contracts supported new application modules, but the production container rendered empty URLs because its environment-variable template still listed only the earliest services.
- Root cause: the service registry type, local config, container defaults, `envsubst` allowlist, and runtime template were maintained independently without a parity gate.
- Prevention: update all runtime-config surfaces whenever a service is registered, and keep a CI check that verifies every platform service URL variable has a default, is allowlisted for substitution, and appears in the rendered template.

## 2026-09-01: Outbox retention must follow the replay contract

- Symptom: transactional Outbox dispatch was reliable, but successfully published rows accumulated forever and some services lacked an index usable by bounded cleanup.
- Root cause: the shared cleanup primitive existed without service lifecycle wiring, retention validation, or an upgrade migration in each owning schema.
- Prevention: every event producer schedules the shared bounded cleaner, validates that published retention is not shorter than the JetStream replay window, deletes only rows with `published_at IS NOT NULL`, and adds a forward/rollback index migration for `(published_at,id)` in every supported dialect.

## 2026-09-01: MySQL TEXT identifiers need prefix indexes

- Symptom: PostgreSQL migration and all Go checks passed, but the MySQL Testcontainers migration failed with error 1170 when indexing an Outbox `TEXT` identifier.
- Root cause: a cross-dialect `(published_at,id)` index assumed that MySQL could index `TEXT` like PostgreSQL; the table primary key already used an explicit 191-character prefix.
- Prevention: inspect the actual MySQL column type before copying index DDL; use `id(191)` for legacy `TEXT` identifiers, retain the full column for `VARCHAR`, and add a static migration regression alongside Testcontainers coverage.

## 2026-09-01: schema artifacts must reflect actual runtime ownership

- Symptom: an event-consumer service appeared to own an Outbox during a platform audit because an old migration still created an Outbox table, although no runtime code ever wrote or dispatched it.
- Root cause: generated infrastructure survived after the service boundary changed, so schema-name scans produced a false ownership signal and left an unmaintained table in new installations.
- Prevention: determine ownership from write paths and lifecycle wiring, not table names alone; remove unused generated tables with a reversible forward migration instead of adding retention workers for data the service does not own.

## 2026-09-01: authenticated tenant scope must survive transport adaptation

- Symptom: several older services successfully verified an Identity JWKS token but copied only its subject into a service-local principal, allowing request-body tenant IDs to replace the trusted token tenant.
- Root cause: scaffold-era HTTP/gRPC adapters defined their own reduced authentication context instead of propagating the shared principal returned by `platform-go/authn`.
- Prevention: all transports store `platform-go/principal.Principal` unchanged, PSK callers become explicit system principals, user-scoped application methods compare the requested tenant with the trusted claim, and CI rejects service-local principal packages or non-Identity token verification.

## 2026-09-01: authorization denial and decision outage are different failures

- Symptom: a timeout or unavailable authorization-service was returned as `403 permission denied`, hiding an infrastructure incident and making clients treat a retryable dependency failure as a permanent policy decision.
- Root cause: the shared enforcement helper wrapped every Authorizer error with the denial sentinel.
- Prevention: Authorizers return `ErrDenied` only for an explicit negative decision and `ErrDecisionUnavailable` for missing clients, deadlines, or RPC failures; HTTP maps them to 403 and 503 respectively, gRPC maps them to `PermissionDenied` and `Unavailable`, and both classifications have unit regressions.

## 2026-09-01: newly published Go module tags may lag at the public proxy

- Symptom: immediately after pushing a valid shared-contract tag, `go mod tidy` failed with an EOF while fetching the new version from `proxy.golang.org`.
- Root cause: the public module proxy had not indexed the just-created Git tag yet; the repository and tag were already reachable directly.
- Prevention: publish contracts before dependent modules, verify the remote tag, use `GOPROXY=direct` only for immediate local validation when proxy propagation lags, and keep released semantic versions in committed module files so CI validates the normal proxy path.

## 2026-09-01: CLI diagnostic writes are still fallible I/O

- Symptom: unit, race, vet, and build checks passed, but CI lint rejected an unchecked `fmt.Fprintln` used to print a Cobra execution error to stderr.
- Root cause: diagnostic output was treated as consequence-free even though the project enables `errcheck` for all writer calls.
- Prevention: route CLI output through injected writers, explicitly handle or intentionally discard both write results, and run the repository's pinned golangci-lint version before pushing CLI changes.

## 2026-09-01: HTTP PSK authentication must preserve the caller credential

- Symptom: a PSK-protected machine endpoint produced a valid service-account principal locally, but its subsequent centralized authorization check had no credential to authenticate to authorization-service.
- Root cause: the HTTP PSK branch stored only the principal, while the Bearer branch also attached the original `Authorization` value for downstream forwarding; gRPC hid the gap because incoming metadata is forwarded automatically.
- Prevention: every authenticated HTTP branch attaches both the shared principal and the original caller credential with `authz.WithCallerCredential`; cover PSK-protected authorization paths when rolling shared authorization into a service.

## 2026-09-01: dependency upgrades must not duplicate the pinned version in CI

- Symptom: all local tests and lint passed after upgrading a service to the latest central Proto module, but CI failed before testing because its contract-ownership gate still asserted the previous exact version.
- Root cause: `go.mod` and the workflow's central-contract version assertion were maintained as separate sources of truth.
- Prevention: keep the exact dependency version only in `go.mod`; immediately run `go mod tidy` after every dependency upgrade so superseded checksums are removed. CI verifies that the dependency is a released semantic version, has no local `replace`, and that the service contains no copied `proto`/`gen` trees. Do not duplicate the exact version string in workflow YAML.

## 2026-09-01: gRPC authorization must cover streaming methods

- Symptom: unary business RPCs were centrally authorized, but a server-streaming discovery/watch method would bypass the unary interceptor entirely.
- Root cause: the shared authorization SDK exposed only a unary server interceptor, and service integration treated that as complete gRPC coverage.
- Prevention: inventory unary and streaming methods separately, apply both shared unary and stream authorization interceptors, include every business stream in the resolver table, and add an exhaustive resolver test plus denial/outage classification tests.

## 2026-09-01: transport E2E tests must stub centralized decisions

- Symptom: expanding authorization from mutations to every business RPC made a service-local Testcontainers E2E test fail with `Unavailable` before reaching its own domain assertion.
- Root cause: the old fixture relied on unprotected read/provider methods and configured no authorization client; complete enforcement correctly introduced an outbound decision boundary.
- Prevention: service-local E2E tests start an in-process authorization gRPC stub and register it through the normal named outbound configuration; never require authorization-service to run, and keep exhaustive resolver unit tests so the E2E stub tests wiring rather than policy ownership.

## 2026-09-01: batched events must not borrow scope from the first item

- Symptom: a usage ingestion batch could contain several tenant/application pairs, while its single Outbox envelope used the first item's scope and exposed unrelated facts to a scoped consumer.
- Root cause: batch atomicity was mistaken for a single authorization and event-routing boundary.
- Prevention: validate and authorize every distinct scope once, group successfully inserted records by authoritative tenant/application scope inside the transaction, and publish one envelope per group; retain a unit regression that decodes every envelope and compares all payload item scopes with its metadata.

## 2026-09-01: use the repository formatter through its own lint command

- Symptom: directly invoking Prettier rewrote Vue and TypeScript files with default double quotes, while the repository ESLint/Prettier integration required single quotes and reported a large formatting diff.
- Root cause: the standalone formatter did not load the same effective rules as the project's lint command.
- Prevention: use `pnpm lint:fix` for platform-console formatting, then run `pnpm typecheck` and `pnpm lint`; do not assume a globally familiar formatter command has the repository's effective configuration.

## 2026-09-01: contract consistency checks must not generate into the worktree

- Symptom: a transient remote Swagger download failure interrupted parallel generation and left temporary `.swagger.json` files; the next consistency check snapshotted those artifacts and reported a false drift after the generator cleaned them.
- Root cause: the read-only consistency check generated directly into the authoritative `src/service/contracts` directory before comparing a snapshot.
- Prevention: the generator accepts an isolated output directory, and the consistency check generates entirely under a temporary directory before diffing it against the committed contracts. Failed or concurrent generation must never mutate the worktree.

## 2026-09-01: budget MySQL composite indexes in bytes before migrations ship

- Symptom: PostgreSQL integration passed, but MySQL rejected an application-scope migration because its `utf8mb4` composite key exceeded InnoDB's 3072-byte limit.
- Root cause: adding another broadly sized identifier to inherited unique and lookup indexes was reviewed by column count rather than worst-case encoded bytes.
- Prevention: give identifiers genuine domain bounds, keep lookup indexes to the selective query prefix, never use collision-prone prefix uniqueness for business keys, and add a unit regression that calculates the worst-case byte budget for every changed composite index.

## 2026-09-02: application verification must remain service-local in E2E tests

- Symptom: enabling application-scoped persistence makes application startup construct a real grant verifier, so an otherwise service-local E2E fixture can fail before serving requests when the named application upstream is absent.
- Root cause: compile-only integration checks do not exercise the Fx startup lifecycle, and an older fixture enabled the database without modeling the newly required outbound boundary.
- Prevention: every database-enabled service E2E test supplies an in-process application-service gRPC contract stub through the normal outbound registry, exercises a non-empty tenant/application scope, and asserts that emitted events retain both values. Never require another deployed service.

## 2026-09-02: asynchronous task scope spans every lifecycle boundary

- Symptom: adding an application filter only to an async task list still leaves collisions and cross-application behavior in idempotency lookup, worker claims, object keys, downloads, retries, provider calls, and events.
- Root cause: application ownership was treated as a presentation filter instead of part of the task identity and execution context.
- Prevention: for every application-owned async task, carry `application_id` through persistence and unique keys, every user and worker predicate, external storage paths, downstream Provider requests, event metadata and payloads, and stale UI state cleanup. Unknown legacy ownership must be disabled or explicitly backfilled, never guessed.

## 2026-09-02: migration down fixtures must satisfy the restored invariant

- Symptom: an application-scope integration test correctly inserted the same tenant/idempotency key for two applications, then the down migration failed while restoring the older tenant-only unique constraint.
- Root cause: the test mixed forward-schema behavior data with migration reversibility without reconciling rows that cannot exist under the previous schema.
- Prevention: test the expanded uniqueness rule first, then explicitly remove or reconcile incompatible fixture rows before running down. A production rollback that contracts a uniqueness scope likewise requires an operator-owned data reconciliation step; down migrations must not silently discard business rows.

## 2026-09-02: frontend application filters are not authorization

- Symptom: an application page submitted a selected application filter, but a caller could bypass the page and query another application directly; tenant-wide search visibility could then expose documents from an ungranted application.
- Root cause: UI scope was mistaken for authorization evidence, and the type-ahead endpoint did not even carry the same filter as the full query.
- Prevention: every application-aware query and suggestion contract carries the same explicit scope, the backend verifies all requested application grants in one bounded batch, and by-ID reads authorize the application derived from persisted data. UI selection narrows results but never grants access.

## 2026-09-02: a component allow-list must also enforce application ownership

- Symptom: dynamic menu configuration could reference any registered component key, so a menu owned by one application could accidentally mount a privileged page owned by another application.
- Root cause: page resolution checked only whether the component key was globally known, not whether its namespace matched the route's authoritative application code.
- Prevention: resolve dynamic pages with both the component key and application code, require an exact namespace match, and cover cross-application rejection with a unit test. Keep backend component values declarative and never evaluate them as import paths.

## 2026-09-02: pin generated API contracts in the URL path

- Symptom: consecutive contract-consistency checks alternated between old and new Swagger output even though the source URL carried a revision query parameter.
- Root cause: the raw source path still referenced the mutable `main` branch; query parameters did not guarantee that every CDN edge resolved the same Git commit.
- Prevention: put an immutable commit SHA in the raw GitHub URL path, regenerate once from that artifact, and run the consistency check twice. Never treat a cache-busting query as a version pin.

## 2026-09-02: do not weaken package install policy for embedded API docs

- Symptom: adding `swagger-ui-dist` to the console pulled an indirect package with an install script that the repository supply-chain policy rejected.
- Root cause: the same Swagger UI distribution was already served by swagger-service, but was unnecessarily introduced as a second frontend dependency.
- Prevention: reuse the fixed-version assets from the configured, HTTPS-validated swagger-service origin and keep the install-script deny policy intact. Unit-test asset URL derivation and JWT request injection.

## 2026-09-02: optional application columns need caller-specific read rules

- Symptom: audit records correctly allowed an empty application for tenant/platform events, but the same optional query filter let a user omit `application_id` and read every application's records in the tenant.
- Root cause: persistence optionality was copied directly into the user-facing read contract without distinguishing interactive users from service-level compliance callers.
- Prevention: require and verify application scope for user query/export, authorize Get from the record's persisted application, and reserve empty/tenant-wide reads for explicitly authorized service identities. Keep ingestion service-authorized so a revoked grant does not prevent immutable history from being recorded.

## 2026-09-02: registered frontend pages still need first-boot catalog reconciliation

- Symptom: the console had a complete local page registry, but a new platform database returned no applications or published menus, leaving users on an empty application launcher with no route into platform administration.
- Root cause: frontend component registration was mistaken for application-service catalog data, and there was no repeatable interface-driven bootstrap after migrations.
- Prevention: maintain a validated declarative application/menu manifest and reconcile it through application-service with an idempotent CLI/Job. Resolve parent IDs from stable menu codes, publish only changed drafts, optionally grant initial tenants, preserve undeclared resources by default, and never seed another service's schema directly.

## 2026-09-02: use the console's ESLint formatter configuration

- Symptom: running standalone Prettier on console files rewrote the established quote and Vue formatting style, after which ESLint reported hundreds of `prettier/prettier` warnings.
- Root cause: the console's authoritative formatting configuration is composed through its ESLint preset and is not equivalent to an unqualified standalone Prettier invocation.
- Prevention: format touched console files with `pnpm exec eslint --fix <files>` (or the repository lint-fix command), then run `pnpm lint`; do not invoke bare Prettier unless its repository configuration has been explicitly verified.

## 2026-09-02: console views are an automatic route discovery boundary

- Symptom: adding an application-scoped workspace component under `src/views` silently generated a global `/platform/workspace` route during build.
- Root cause: Soybean's route plugin treats files under `src/views` as route sources, even when the component is intended to be mounted only by a dynamic application route.
- Prevention: keep reusable or dynamically scoped shell components under `src/components` or `src/apps`; reserve `src/views` for intentional generated routes, and inspect generated route/type diffs after every build.

## 2026-09-02: tenant selection is a session-scope exchange

- Symptom: the console remembered a selected tenant and immediately called application-service, but the login token had no trusted tenant/membership claims and was correctly denied; the internal token RPC also accepted caller-supplied membership scope from ordinary users.
- Root cause: UI selection state was confused with authenticated server-side scope, and issuing a one-off access token would not have survived refresh.
- Prevention: validate membership in tenant-service, restrict identity token issuance to trusted services, pass the existing session ID in the shared contract, atomically persist session scope before returning the token, and serialize client scope changes. Self-service tenant listing must bind the requested user to the authenticated principal.

## 2026-09-02: module upgrades must finish with tidy

- Symptom: local tests, race, vet, lint, and builds passed after a shared Proto upgrade, while CI failed its module consistency step because `go.sum` still contained checksums for the superseded Proto version.
- Root cause: `go get` added the new module version but did not remove every now-unused checksum; the local gate omitted the same `go mod tidy && git diff` assertion used by CI.
- Prevention: after every Go dependency version change, run `go mod tidy`, verify `git diff --exit-code -- go.mod go.sum` after a second tidy, then run the service gates before pushing.

## 2026-09-02: authorization preview APIs must bind interactive subjects

- Symptom: authenticated users could submit arbitrary `tenant_id`, `subject_id`, and `subject_type` values to generic authorization decision endpoints, making those endpoints an authorization-information oracle even though management routes were protected.
- Root cause: a service-to-service preview contract was exposed to browser users without replacing caller-supplied subject fields with trusted JWT scope.
- Prevention: bind interactive decisions to the JWT tenant and membership, permit arbitrary-subject previews only for authenticated service/system callers, and provide a dedicated current-membership permission-code batch endpoint for navigation. Filtered menus improve UX but never replace backend authorization.

## 2026-09-02: tenant token exchange is a client-side commit boundary

- Symptom: after the session token successfully changed from tenant A to tenant B, a later application or permission request could fail while the console still displayed tenant A resources under the tenant B credential.
- Root cause: token exchange and dependent navigation loading were treated as one ordinary request chain without distinguishing failure before versus after trusted server scope changed.
- Prevention: serialize tenant changes, record the exchange commit point, preserve the previous context only when exchange itself fails, and fail closed by clearing applications, navigation, and selected application after a post-exchange failure. Cover both branches with a deterministic state-transition unit test.

## 2026-09-02: navigation cannot evaluate resource-dependent ABAC

- Symptom: a menu permission backed by an ABAC expression referencing `resource_id` or request attributes caused the entire navigation permission batch to fail because those facts do not exist at menu-load time.
- Root cause: full resource authorization evaluation was reused for coarse navigation visibility.
- Prevention: navigation decisions admit only unconditional active grants (including unconditional wildcard roles), hide conditional grants conservatively, and let the protected domain operation evaluate ABAC later with authoritative resource facts. Never invent attributes merely to render a menu.

## 2026-09-02: a menu permission code is incomplete without decision scope

- Symptom: a single application workspace mixed tenant administration with platform-level Identity and registry pages, but a bare permission code could be checked only against the selected membership or only against the reserved platform user.
- Root cause: menu metadata modeled the policy name but omitted the authorization subject namespace in which that policy is evaluated.
- Prevention: persist and publish `permission_scope` with every protected menu, add it compatibly to the central Proto, derive both subjects exclusively from the JWT, and keep allowed-code sets keyed by `(scope, code)`. Never union platform and tenant results by code alone.

## 2026-09-02: self-authorized policy management needs a tenant bootstrap path

- Symptom: once authorization-service protected its own role and permission management routes, a newly created tenant had no role capable of creating the first role, producing a circular authorization deadlock.
- Root cause: platform administrator bootstrap existed, but tenant ownership was persisted only in tenant-service and never projected into the authorization domain.
- Prevention: consume the authoritative tenant-created event with a durable, transactional, idempotent projection; create only a reserved tenant-local wildcard role bound to the event's owner membership, bump the policy version, and leave all other members unprivileged. Never solve this by disabling self-authorization in production or querying tenant tables directly.

## 2026-09-02: request-body scope is not authenticated scope

- Symptom: a frontend permission selector could reuse an administrative list endpoint by sending either the selected tenant ID or `__platform__`, even though route middleware authorized only the caller's Token scope and did not bind the body field.
- Root cause: possession of a management permission in one namespace was confused with authority to choose another namespace in request data.
- Prevention: expose a current-principal lookup API that validates the selected tenant against JWT claims, derives the membership or global-user subject server-side, authorizes the exact derived namespace, and queries only that namespace. Apply the same persisted-or-derived scope binding to every browser-facing read and mutation; never trust tenant, subject, or platform markers merely because they were valid JSON.

## 2026-09-02: Vue component option props may require mutable arrays

- Symptom: ESLint and unit tests passed for an `ElSegmented` scope selector, but `vue-tsc` rejected a readonly tuple created with `as const` because the component Prop expects mutable `Option[]`.
- Root cause: literal narrowing and component assignment compatibility were treated as the same concern; a readonly tuple narrows values but cannot be assigned to a mutable third-party Prop.
- Prevention: export an explicitly typed mutable array such as `Array<{ label: string; value: Scope }>` when passing options to Element Plus, and run `pnpm typecheck` after UI component changes rather than relying on ESLint alone.

## Standard GET routes must be path-scoped at the gateway

- Symptom: enabling GET for JWKS and Swagger on a catch-all external route also exposed metrics, health probes, and potentially pprof.
- Root cause: the HTTP method exception was widened on `/*` instead of being coupled to the exact standard endpoint paths.
- Prevention: split business and standards route groups, allow GET/HEAD only on an explicit path allow-list, keep internal operational endpoints excluded, and assert rendered routes never contain `/metrics`.

## SQL insert shape needs a local deterministic gate

- Symptom: billing refund creation failed in both PostgreSQL and MySQL integration CI because an INSERT listed 16 columns but supplied 17 placeholders.
- Root cause: the write path had no unit-level shape assertion and was only exercised after a new repository integration scenario was added.
- Prevention: keep reusable INSERT placeholder lists next to their column lists, assert their counts match in a fast unit test, and retain cross-database Testcontainers coverage for actual execution.

## Verify a supposedly new path before adding a file

- Symptom: adding a small SQL-shape regression test replaced an existing repository test file and silently removed unrelated coverage.
- Root cause: the target path was assumed to be new without checking the worktree or repository history first.
- Prevention: resolve the target with `rg --files` or `test -e` before an Add File patch; when it exists, inspect and append with a scoped update, then review commit deletion statistics before pushing.

## Browser multipart upload must also stream its checksum

- Symptom: a UI split uploads into bounded parts but computed SHA-256 with `Blob.arrayBuffer()` first, so the entire large file was still copied into browser memory.
- Root cause: network chunking and client-side preprocessing were reviewed independently even though both participate in the same memory budget.
- Prevention: use a maintained incremental hash implementation with fixed-size slices, bound concurrent part uploads, abort failed server sessions, and unit-test exact byte ranges and worker partitioning. Require bucket CORS to expose `ETag` before enabling browser multipart completion.

## Scaffold authentication screens are not delivered capabilities

- Symptom: the console exposed demo credentials plus SMS login, registration, and password-reset screens even though only password login had a backend contract; some unsupported screens displayed success without making a request.
- Root cause: upstream scaffold modules were treated as product features instead of being reconciled against the platform's authoritative identity-service capabilities.
- Prevention: default credential fields to empty, expose only contract-backed authentication modes, and make unsupported route parameters fall back safely. Add a UI only after its verification, rate limiting, audit, session, and abuse-prevention backend is implemented and tested.

## Identity snapshots must clear fields as well as set them

- Symptom: enriching `/me` with optional tenant and profile fields could leave a previous tenant or membership visible after a later response omitted an empty field, because the client merged the partial object into existing state.
- Root cause: a replacement snapshot was serialized and consumed with patch semantics; `omitempty` and `Object.assign` preserved values that should have been cleared.
- Prevention: return a stable response shape with explicit empty values and normalize the client snapshot from empty defaults before replacing state. Test transitions where scope-bearing fields disappear, not only where fields are added.
