# phase7-artifact-admission-hardening-and-runtime-wiring
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

0100 の game artifact foundation を production admission path にする。archive/schema/WASM policy を
定義済みの制約に従って検証し、`arena-service` が configured filesystem または R2 store、writable registry、upload API を
同じ runtime dependency graph で起動するようにする。

## Existing Implementation References

- `artifactbundle/bundle.go`: fixed ZIP reader and digest calculation
- `internal/platform/service/artifact_bundle_store.go`: filesystem materialization
- `internal/platform/service/artifact_bundle_s3.go`: S3/R2 store
- `internal/platform/service/artifact_admission.go`: game release admission
- `internal/platform/service/http.go`: multipart upload adapter
- `cmd/arena-service/main.go`: current service dependency wiring

## Code Change Map

- `artifactbundle/` (MODIFY): bounded ZIP read, entry-size/compression/hash/case checks, JSON Schema parity, wazero compile/import policy
- `schemas/arena-bundle-v1.schema.json` (MODIFY): complete game/AI manifest shapes and resource budget limits
- `internal/platform/service/artifact_*.go` (MODIFY): idempotent filesystem/R2 persistence and secure materialization
- `cmd/arena-service/main.go` (MODIFY): configured bundle store, composite registry resolver, and upload admission injection
- tests/fixtures (NEW/MODIFY): malicious archive, bad manifest/hash/import, oversize and local/R2 parity cases
- `docs/specs/platform-artifact-bundle.md` (MODIFY): observable validation/error and runtime behavior

## Sub-tasks

- [ ] Implement bounded streaming validation and the archive negative matrix.
- [ ] Validate WASM version/compile/import policy and manifest resource ceilings.
- [ ] Make JSON Schema and typed model parity testable.
- [ ] Wire selected artifact backend, writable registry, resolver and operator API in `arena-service`.
- [ ] Verify repeated bytes are idempotent and local/R2 produce the same digest/materialization result.

## Verification

- focused Go tests plus archive fuzz/property coverage where practical
- TypeSpec/schema verification and `make lint`
- filesystem and local S3-compatible admission/materialization round trips
