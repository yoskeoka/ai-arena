# dungeon-external-repo-removal-gate-verification
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0041-dungeon-external-repo-migration-03-ai-arena-removal.md` を安全に実行する前提として、
`dungeon-game-ai-arena` 側へ dungeon 用 verification asset と CI coverage が移り切ったことを確認し、
ai-arena から削除してよい hard gate を閉じる。

親 plan:

- `0038-dungeon-sidecar-boundary.md`

depends on:

- `0039-dungeon-external-repo-migration-01-bootstrap-and-golden-parity.md`
- `0040-dungeon-external-repo-migration-02-sdk-tag-and-import-contract.md`

## Scope

- `dungeon-game-ai-arena` 側へ残すべき dungeon fixture bot / local AI / WASM AI / golden / e2e 資産の棚卸し
- ai-arena 側で dungeon 用に使っていた verification path を external repo 側へ移すか、不要化理由を固定する
- external repo 側で tagged ai-arena import 前提の local / CI verification を完成させる
- `0041` が参照できる removal gate 証跡を docs と CI config に固定する

この plan では以下は扱わない。

- ai-arena からの dungeon code 削除そのもの
- dungeon ruleset / payload schema の変更
- `v0.1.0` より先の SDK surface 拡張
- external repo 側の新 mechanic 追加

## Spec Changes

### `docs/specs/game-master.md`

- external game repo が host platform を consumer として使う際、
  game-specific fixture / AI verification も consumer repo 側で持てることを明記する

### `docs/specs/platform.md`

- ai-arena が提供するのは runner / SDK / runtime contract であり、
  game 固有の fixture bot / golden / WASM verification asset は external repo ownership へ移せることを明記する

### `dungeon-game-ai-arena` repo docs

- dungeon 固有 verification の canonical source が external repo 側にあることを明記する
- same-golden だけでなく fixture bot / Go-WASM / Rust AI player / CI coverage の parity を削除 gate として記録する

## Expected Code Changes

### `dungeon-game-ai-arena` repo

- `testdata/ai/dungeon/...` の不足 asset 補完
- 必要なら dungeon 用 fixturebot helper や Rust AI player fixture を追加
- dungeon 用 Go-WASM / Rust AI player verification を実行する `Makefile` target 整備
- local verification と同じ資産を使う CI workflow の追加または拡張
- removal gate 証跡を残す docs / README / spec 更新

### `ai-arena` repo

- `0041` が参照する削除 gate 文面の更新
- external repo ownership 前提に読める platform spec wording の補強

## Verification

この plan の execution PR は、少なくとも以下を満たしたとき完了とする。

- external repo 側で tagged ai-arena import 前提の build/test が通る
- external repo 側で same-golden local / CI e2e が通る
- external repo 側で dungeon fixture bot / local AI verification が通る
- external repo 側で dungeon 用 Go-WASM verification が通る
- external repo 側で Rust AI player verification を通すか、dungeon 専用 asset が不要である理由を docs に固定した上で、
  external repo CI における扱いが reviewers に分かる
- `0041` から参照できる removal gate 証跡が docs または CI config に残る

## Completion Notes

- external repo の ownership 表と removal gate は
  `dungeon-game-ai-arena/docs/specs/dungeon-external-sdk-consumption.md` に固定する
- dungeon 固有 Go-WASM verification は
  `dungeon-game-ai-arena/testdata/ai/dungeon/dungeon-go-wasm-ai`,
  `dungeon-game-ai-arena/e2e/dungeon_go_wasm_e2e_test.go`,
  `dungeon-game-ai-arena/Makefile`,
  `dungeon-game-ai-arena/.github/workflows/wasm-verification.yml`
  を証跡とする
- Rust AI player runtime は dungeon 固有 asset を増やさず、
  host repo ai-arena の `.github/workflows/wasm-verification.yml` にある Rust-WASM lane を canonical coverage とする

## Sub-tasks

- [x] ai-arena 側に残っている dungeon verification asset と CI path を棚卸しし、external repo 側 ownership 表を作る
- [x] [parallel] external repo 側へ dungeon fixture bot / local AI asset を移し、local verification を通す
- [x] [parallel] external repo 側へ dungeon Go-WASM verification を移し、CI 化する
- [x] [parallel] external repo 側の Rust AI player verification 方針を確定し、asset/CI か不要化理由のどちらかを固定する
- [x] [depends on: external repo 側へ dungeon fixture bot / local AI asset を移し、local verification を通す, external repo 側へ dungeon Go-WASM verification を移し、CI 化する, external repo 側の Rust AI player verification 方針を確定し、asset/CI か不要化理由のどちらかを固定する] removal gate 証跡を docs と CI config にまとめる
- [x] [depends on: removal gate 証跡を docs と CI config にまとめる] `0041` の dependency と verification 条件が満たせることを確認する

## Parallelism

- fixture/local AI、Go-WASM、Rust AI player verification の 3 本は並行に進められる
- removal gate の最終整理は各 verification line の結論が揃ってから行う

## Risks and Mitigations

- same-golden だけで削除可能だと誤認し、ai-arena 側 CI から消した後に外部 repo verification が欠ける
  - mitigation: fixture bot / Go-WASM / Rust AI player / CI coverage を独立 gate として列挙する
- Rust AI player coverage の扱いが曖昧なまま reviewer ごとに解釈が割れる
  - mitigation: asset を作らない場合でも、不要化理由と platform-level coverage の所在を docs に固定する
- external repo CI が local verification と別 asset を使い、削除 gate 証跡として弱くなる
  - mitigation: local と CI が同じ fixture / golden / AI asset を参照する構成に寄せる
