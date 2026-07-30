# phase7-artifact-backed-ai-submission-and-worker-dispatch
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

AI bundle admission と game/AI digest-pinned match execution を成立させる。queue と worker は mutable
path/URL ではなく admitted artifact identity を受け取り、game master と player WASM module を materialize
して同じ match runtime へ渡す。

## Existing Implementation References

- `internal/platform/service/general.go`: game registration and AI submission entities
- `internal/platform/service/request.go`: match request durable identity
- `internal/platform/service/worker*.go`: current local worker dispatch
- `internal/platform/registry/wasm_resolver.go`: game artifact session resolution
- `internal/platform/runtime/wasm_wasi.go`: WASI player runtime

## Code Change Map

- `artifactbundle/` (MODIFY): AI manifest/admission model
- `internal/platform/service/general.go` (MODIFY): artifact ID based AI revisions instead of caller `artifact_ref`
- `internal/platform/service/request.go` and stores (MODIFY): game/AI digest snapshot on accepted request
- `internal/platform/service/worker*.go` (MODIFY): materialize and launch game/AI WASI modules
- `typespec/` and generated operator client (MODIFY): AI bundle upload and immutable response identities
- tests (NEW/MODIFY): game+AI upload to queued WASI match and retry/rerun digest pinning

## Sub-tasks

- [ ] Define and validate AI bundle manifest and upload response.
- [ ] Persist stable submission/revision identity with immutable artifact digest.
- [ ] Replace worker path input with materialized game/AI modules.
- [ ] Snapshot exact release/revision digests at queue acceptance and preserve them for retry/rerun.

## Verification

- uploaded WASM game master and AI complete a match without host artifact path input
- match/retry/rerun records preserve the originally selected digests
- affected Go/TypeSpec/operator UI tests and workflow lint
