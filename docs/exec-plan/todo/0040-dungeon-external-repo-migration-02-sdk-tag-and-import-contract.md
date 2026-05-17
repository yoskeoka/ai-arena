# dungeon-external-repo-migration-02-sdk-tag-and-import-contract
**Execution**: Use `/execute-task` to implement this plan.

## Objective

ai-arena に残す public SDK surface を、external repo から安定して参照できる形で固定し、
`github.com/yoskeoka/ai-arena` に `v0.1.0` tag を打って `dungeon-game-ai-arena` が
その tag を import して JSON-RPC 2.0 + stdio transport 契約を維持したまま開発できる状態を作る。

親 plan:

- `0038-dungeon-sidecar-boundary.md`

depends on:

- `0039-dungeon-external-repo-migration-01-bootstrap-and-golden-parity.md`

## Scope

- ai-arena module の public SDK surface 確認
- `github.com/yoskeoka/ai-arena/gamemaster` 依存の安定化
- external repo 側 import を local path 前提から released module 前提へ切り替える準備
- `v0.1.0` tag 作成と、その tag を使う verification

この plan では以下は扱わない。

- dungeon ruleset や payload schema の更新
- ai-arena 側からの dungeon 実装削除
- `gamemaster` 以外の広い public API 化
- external backend 向けの network transport

## Spec Changes

### `docs/specs/platform-common-contract.md`

- external game repo が安定依存してよい DTO 正本が `github.com/yoskeoka/ai-arena/gamemaster` であることを
  versioned module consumption 前提で補足する

### `docs/specs/game-master.md`

- external repo sidecar は ai-arena module tag を通じて `gamemaster` package を参照し、
  JSON-RPC 2.0 + NDJSON stdio contract を維持することを明記する

### `docs/specs/platform.md`

- ai-arena runner が external game development の host として使われるときも、
  public import surface は `gamemaster` package に限定することを補足する

## Expected Code Changes

### `ai-arena` repo

- `gamemaster` package の public surface 点検と必要最小限の整理
- package doc / example / module consumption に必要な最小更新
- release/tag 手順または補助 docs

### `dungeon-game-ai-arena` repo

- `go.mod` の ai-arena 依存を release tag 前提へ更新
- local replace 依存が残るなら排除する
- tagged ai-arena module を使う build/test/CI 確認

## Verification

この plan の execution PR は、少なくとも以下を満たしたとき完了とする。

- `github.com/yoskeoka/ai-arena/gamemaster` だけで external repo sidecar が build できる
- `dungeon-game-ai-arena` が ai-arena local checkout ではなく `v0.1.0` tag を参照して動く
- tagged import に切り替えた後も same-golden local / CI e2e が維持される
- ai-arena 側の public import surface が `gamemaster` package を越えて広がっていない

## Sub-tasks

- [ ] ai-arena に残す public SDK surface を最終確認する
- [ ] external repo の ai-arena 依存を release tag consumption 前提へ更新する
- [ ] [parallel] ai-arena 側の package doc / release 補助 docs を整える
- [ ] [parallel] tagged import での build/test/CI 確認を用意する
- [ ] [depends on: ai-arena 側の package doc / release 補助 docs を整える, tagged import での build/test/CI 確認を用意する] `v0.1.0` tag を作成し、external repo verification を固定する

## Parallelism

- public SDK surface の確認後は、release 補助 docs と tagged-import verification を並行できる
- tag 作成自体は verification 導線が揃ってから直列で行う

## Risks and Mitigations

- external repo が `gamemaster` 以外の package に依存し始める
  - mitigation: import audit を行い、stable dependency を `gamemaster` に限定する
- tag を打ってから public surface を変えると外部開発フローが不安定になる
  - mitigation: same-golden verification と import audit を通してから `v0.1.0` を切る
- local `replace` 依存が残ると external repo 単独開発が成立しない
  - mitigation: CI を released module consumption 前提に切り替えてから完了扱いにする
