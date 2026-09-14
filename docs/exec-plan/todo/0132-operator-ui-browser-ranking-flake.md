# operator-ui-browser-ranking-flake

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0122-go-1-27-wasi-runtime-stopped.md`

## 目的と完了境界

operator browser の file-backed、Postgres、local OIDC lane が built-in Go-WASM AI を再実行・promotion したとき、AI bundle が worker の memory contract を明示して起動し、completed official run から ranking snapshot まで到達できるようにする。

完了後、browser fixture generator が生成する game と AI の全 WASI bundle は同じ十分な linear-memory upper bound を manifest に持つ。public API、ranking 集計、worker の retry/rerun semantics、実行 runtime の default 値は変更しない。

## References and current behavior

- `tools/dev/package-builtin-game-bundles.sh:16-31` は game bundle に `memory_limit_pages: 1024` を明示する。
- 同ファイルの `pack_ai` (`:34-50`) は AI bundle の memory limit を省略し、worker の WASI default（64 pages）へ落ちる。
- `internal/platform/service/worker_local.go:119-153` は admitted AI bundle manifest の memory limit を runtime config へ伝播する。
- `internal/platform/runtime/runtime.go:21-23` は manifest が省略された場合の default を 64 pages と定める。
- `docs/issues/0122-go-1-27-wasi-runtime-stopped.md:1-28` は Go 1.27 WASI artifact の不足 memory による `runtime-stopped` を記録する。
- PR #356 の failed artifacts は rerun candidate の `init failed for p1: runtime-stopped` と、その結果として ranking snapshot が作られないことを示した。

## Change map

- (MODIFY) `tools/dev/package-builtin-game-bundles.sh`: generated AI fixture manifest に 1024 pages の WASI memory limit を明示し、同一 generator の game fixture と一致させる。
- (DELETE) `docs/issues/0122-go-1-27-wasi-runtime-stopped.md`: 上記の browser fixture omission が残っていた原因を解消した後に削除する。
- (NO SPEC CHANGE) `docs/specs/ai-runtime.md`, TypeSpec, service/worker/runtime code: observable runtime contract と API は既に memory limit propagation を定義しており、変更しない。

## Work

1. AI fixture manifest に game fixture と同じ `memory_limit_pages: 1024` を追加する。
2. generator が作る echo と janken の AI bundle manifest を inspection し、limit が明示されることを確認する。
3. focused Go-WASM artifact-admission regression と applicable operator browser lane を実行し、rerun/promotion 後の ranking snapshot path を確認する。
4. verification evidence を確保した後、解決済み local issue を implementation PR で削除する。

## Dependencies and parallelism

この plan は merged Go 1.27 WASI runtime compatibility work に依存するが、ranking discovery UI とは独立する。fixture generator と browser lane の correction は同じ execution branch で直列に扱う。

## Verification

- generated `echo-ai-alpha` と `janken-ai-beta` ZIP の `manifest.json` に `memory_limit_pages: 1024` がある。
- `GOTOOLCHAIN=go1.27.1` と writable cache で `TestArtifactSubmissionUploadToWASIStartAcrossBundleStores` が通る。
- `pnpm --dir operator-ui run build`。
- applicable file-backed、Postgres、local OIDC operator browser lanes が rerun/promotion/ranking snapshot まで通る。
- workflow lint。

## Non-goals

- memory default の引き上げ、runtime adapter の変更、retry/rerun policy の変更、public API/TypeSpec の変更、ranking UI の変更。
