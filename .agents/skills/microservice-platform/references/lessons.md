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
