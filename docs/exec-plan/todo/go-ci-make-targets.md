# go-ci-make-targets
**Execution**: Use `/execute-task` to implement this plan.

## Objective

PR #108 で追加された Go platform foundation を、GitHub Actions CI とローカル開発の同じ入口で検証できるようにする。

この plan は以下を成立させる。

- `make test` で unit test を実行できる
- `make lint` で formatter check と必要最小限の Go lint を実行できる
- CI で `make test` と `make lint` を実行し、formatter fix が必要な状態を lint failure として扱う
- `golangci-lint` は導入せず、必要な checker だけを個別に固定して使う

## Context

- `docs/project-plan.md` は platform 実装言語を Go としている。
- `docs/design-decisions/adr.md` は Go 採用理由として wazero との統合、単一バイナリ配布、goroutine/context による締切管理を挙げている。
- 同 ADR は Go の型表現上の弱さを「仕様とテストで補強する」必要があるとしている。
- `docs/design-decisions/core-beliefs.md` は "Correctness over Speed" と spec-first を優先している。
- PR #108 は `internal/platform/**`、`cmd/arena-runner`、`go.mod` を追加済みで、以後の Phase 2 plans はこの foundation に依存する。

## Scope

- Go unit test CI
- Go lint CI
- ローカル `Makefile` entrypoint
- Go tool version pinning
- CI での module/tool cache
- Go quality gate の spec 追加

この plan では以下は扱わない。

- `platform-phase2-02-fixture-e2e.md` の fixture/e2e 実装
- `platform-phase2-03-replay-debug.md` の replay/debug 実装
- `platform-phase2-04-janken-integration.md` の janken skeleton
- `golangci-lint` 導入
- lint finding の大量抑制や project-wide policy tuning

## Spec Changes

### `docs/specs/go-quality-gates.md`

Go module の品質ゲート仕様を追加する。

- `make test` は `go test ./...` を実行する
- `make lint` は少なくとも以下を実行する
  - `goimports` formatting check
  - `go vet ./...`
  - `noctx`
  - `staticcheck ./...`
  - `gosec ./...`
- formatter は `goimports` とし、CI では自動修正せず、修正が必要なファイルを出力して失敗する
- lint tool は `golangci-lint` ではなく個別 tool として固定する
- CI とローカル Makefile は同じ command surface を使う

## Expected Code Changes

- `Makefile`
  - `make test`
  - `make lint`
  - `make fmt`
  - tool bootstrap / pinned tool invocation helper
- `go.mod`
  - Go tool dependency pins for `goimports`, `noctx`, `staticcheck`, and `gosec`
- `.github/workflows/go-ci.yml`
  - checkout
  - Go setup using the module Go version
  - cache for Go module/build/tool state
  - `make test`
  - `make lint`

Implementation should prefer Go tool dependency pinning supported by the repository's Go version so that local and CI runs use the same tool versions. If the exact tool-pinning mechanism needs adjustment during execution, preserve the invariant that tool versions are explicit and reproducible.

## Lint Commands

The intended lint surface is:

- `goimports -l` over tracked `.go` files; non-empty output fails
- `go vet ./...`
- `go vet -vettool=<noctx> ./...`
- `staticcheck ./...`
- `gosec ./...`

The execution may wrap these commands in Makefile variables, but the commands above are the behavioral contract.

## Sub-tasks

- [ ] Add `docs/specs/go-quality-gates.md`
- [ ] Add `Makefile` targets for `test`, `lint`, and `fmt`
- [ ] Pin `goimports`, `noctx`, `staticcheck`, and `gosec` tool versions
- [ ] Add GitHub Actions workflow for Go CI
- [ ] Run `make test`
- [ ] Run `make lint`
- [ ] Fix or explicitly document any first-run lint findings that are in scope for this quality-gate setup

## Parallelism

- [parallel] Spec writing and CI workflow drafting can proceed independently after the command contract is fixed.
- [parallel] Tool pinning and Makefile target implementation can proceed independently of CI cache tuning.
- CI verification depends on the Makefile targets and tool pinning.

## Risks and Mitigations

- `gosec` and `staticcheck` may report findings in PR #108 code on first introduction.
  - mitigation: fix straightforward findings in the execution PR; if a finding is a false positive or belongs to later feature work, document the narrow suppression or follow-up reason in the PR.
- `goimports` invocation can be slow or inconsistent if it relies on ambient developer installations.
  - mitigation: use pinned tool invocation and make CI call the same Makefile target as local development.
- `noctx` runs through `go vet -vettool`, so the Makefile must handle the built analyzer path portably.
  - mitigation: keep analyzer bootstrap inside the Makefile/tool helper instead of relying on `which noctx`.
- Over-broad lint suites can slow Phase 2 iteration.
  - mitigation: keep the initial lint list limited to the tools named in this plan and expand only through future plans.

## Verification

The execution PR is complete when the following pass locally and in CI:

- `make test`
- `make lint`

The PR body must include the exact local command results and note that formatter drift is treated as a lint failure.
