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
