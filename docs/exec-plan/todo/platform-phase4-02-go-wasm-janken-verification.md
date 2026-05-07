# platform-phase4-02-go-wasm-janken-verification
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Go で実装した AI player を WASM/WASI へビルドし、`janken` を使って `arena-runner` の正式経路として完走できることを固定する。Phase 4 ではまず Go 製 WASM を基準実装にし、ダンジョンゲームの AI 開発者が「Go で bot を書いて WASM へ出す」流れを迷わず再現できる状態にする。

親 plan:

- `platform-phase4-wasm-runtime.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

depends on:

- `platform-phase4-01-wasm-runtime-contract.md`

## Scope

- Go 製 `janken` sample AI を WASM/WASI へビルドできる参照実装の追加
- `arena-runner` から Go 製 WASM player を起動する `janken` 検証経路の追加
- 手元で実行結果を確認できる `make` helper や fixture の整備
- Go 製 WASM を Phase 4 の最初の supported reference path として doc/spec に反映

この plan では以下は扱わない。

- Rust/Zig/C++ など非 Go toolchain の評価
- online 提出 UX
- dungeon game の AI 実装そのもの
- 多言語 support matrix の確定

## Spec Changes

### `docs/specs/ai-runtime.md`

- Go から WASM/WASI へビルドする参照フローを example として追加する
- sidecar manifest と module artifact の対応例を追加する
- Go 製 sample で期待する `stdout` / `stderr` / exit/shutdown の例を追記する

### `docs/specs/janken-game.md`

- `janken` を Phase 4 の WASM runtime 検証ゲームとして位置づける
- sample AI verification で使う最低限の ruleset / player 構成 / expected artifact を明記する

### `docs/specs/go-quality-gates.md`

- Go 製 WASM sample build と runner verification を、常設 quality gate にするか手元 helper に留めるかを明記する
- 少なくとも手動確認用 `make` helper の存在を spec に合わせる

## Expected Code Changes

### sample AI / fixtures

- Go 製 `janken` sample AI の source を追加する
- WASM build 用の sidecar manifest と build target を追加する
- sample AI が `init` / `turn` / `game_over` を `janken` contract どおり扱うことを示す

### `testdata/ai/` and verification helpers

- `arena-runner` から直接参照できる Go 製 WASM fixture を追加する
- `make run-janken-go-wasm` などの lightweight helper を追加し、artifact path や期待結果を確認しやすくする

### tests / runner integration

- `janken` を対象に WASM player 2 体、または subprocess + WASM 混在構成の検証を追加する
- sample build 失敗、manifest 不整合、runtime 起動失敗のケースも最低限カバーする

## Verification

- `go test ./...`
- Go 製 WASM sample を build できる
- `arena-runner` で `janken` match を完走できる
- match artifact に record / history / stderr summary が残る
- `make` helper で人間が最短経路の動作確認を再現できる
- subprocess bot と Go-WASM bot の混在でも共通 contract が崩れない

## Sub-tasks

- [ ] Define the Go-to-WASM reference flow in spec
- [ ] Add a Go-based `janken` sample AI intended for WASM/WASI build output
- [ ] Add build helpers and sidecar metadata for the sample AI
- [ ] Add runner verification for `janken` with WASM players
- [ ] [parallel] Add a manual `make` helper for visible local verification
- [ ] [parallel] Add tests for manifest/build/runtime failure cases
- [ ] Decide whether the WASM build/run path belongs in default quality gates or only in dedicated verification helpers

## Parallelism

- sample AI source and `make` helper can proceed in parallel
- runner verification and failure-case tests can proceed in parallel after the sample artifact contract is fixed

## Risks and Mitigations

- Go の WASM 出力フローを sample 専用の特殊ケースとして作ると、将来の他 game に流用しづらい
  - mitigation: sample 専用 script ではなく `ai-runtime.md` の一般 contract と sidecar shape を先に固定する
- CI に載せる範囲を広げすぎると Phase 4 の初回導入が重くなる
  - mitigation: まずは manual helper + targeted verification を成立させ、常設 gate 化は明示的に判断する
- `janken` 用 fixture が sample 成功ケースしか持たないと runtime failure の見え方が分からない
  - mitigation: manifest/runtime 起動失敗の負例も最小限追加する

## Design Decisions

- Phase 4 の first supported path は Go 製 WASM とする
- `janken` を WASM runtime の最初の公式検証ゲームとして使う
- visible confirmation を重視し、automated test だけでなく `make` helper も追加対象に含める
