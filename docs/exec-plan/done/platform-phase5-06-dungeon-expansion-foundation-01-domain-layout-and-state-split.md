# platform-phase5-06-dungeon-expansion-foundation-01-domain-layout-and-state-split
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 5 完了後の dungeon refactor の最初の段階として、`games/dungeon` の domain model を
`ruleset definition` / `generated layout` / `match state` / `contract payload` に分解する。
feature expansion 前に「何が静的ルールで、何が match ごとの生成結果で、何が進行中 state か」を
固定し、Rogue / NetHack 系まで伸ばすときの基礎境界を先に整える。

関連 issue:

- `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md`

depends on:

- `docs/exec-plan/done/platform-phase5-02-dungeon-seeded-generation.md`
- `docs/exec-plan/done/platform-phase5-03-dungeon-wasm-reference-ai.md`
- `docs/exec-plan/done/platform-phase5-04-dungeon-scenario-catalog-and-targeted-fixtures.md`
- `docs/exec-plan/done/platform-phase5-05-dungeon-reference-ai-memory-policy-split.md`

## Scope

- `Ruleset` が抱えている static rule と generated layout を分離する
- `FullState` / `PublicState` / `VisibleState` と internal mutable state の境界を分離する
- `Match` を domain facade として薄くし、state assembly の責務を別に出す
- 以後の plan が `actors` `items` `effects` `combat` を差し込める state shape を先に決める

この plan では以下は扱わない。

- turn progression の phase 分解そのもの
- monsters / items / trap の実ルール追加
- reference AI policy の強化

## Spec Changes

### `docs/specs/dungeon-game.md`

- dungeon domain の構成要素を責務ベースで整理する
  - `ruleset definition`
  - `generated layout`
  - `match state`
  - `visible/public/full contract payload`
- `generated layout` に含まれるものと、進行中 `match state` にだけ存在するものを明文化する
- 将来追加される actor / item / effect が state 上どこへ載るかの責務境界を記述する

### `docs/specs/platform.md`

- dungeon exported/public state が platform 側へどう見えるかについて、layout と match state の分離前提を補足する

### `docs/design-decisions/adr.md`

- dungeon を subsystem 指向の domain tree へ再編する判断を記録する
- `Match` を全責務の集積点に戻さない方針を記録する

## Expected Code Changes

### domain structure

- `games/dungeon` 配下で少なくとも次の単位を分ける
  - contract types
  - ruleset definitions
  - generated layout
  - mutable match state
- 現行 `Ruleset` の `Tiles` / `SpawnPoints` / `Goal` / `InitialChests` を
  generated layout 側へ移し、static rule は別 struct に寄せる

### state assembly

- fresh run / resume で共通に使う state assembly 入口を整理する
- `FullState` 復元で domain state を構築する責務を `Match` 本体から薄く切り出す

### tests

- layout と state 復元の unit test を追加または整理する
- scenario catalog が新しい state 境界に追従できるよう helper を見直す

## Verification

- `go test ./games/dungeon/... ./cmd/dungeon-gamemaster/... ./e2e/...`
- same `rng_seed` / same snapshot から同じ generated layout と state を復元できる
- domain package tree が引き続き `internal` import を持たない
- scenario helper が random generation 非依存で state を短く組み立てられる

## Sub-tasks

- [ ] 現行 `Ruleset` / `FullState` / `Match` の責務棚卸しを行う
- [ ] static rule と generated layout の新しい型境界を定義する
- [ ] mutable match state と contract payload を分離する
- [ ] fresh run / resume の state assembly を整理する
- [ ] scenario helper と state 復元 test を追従させる

## Parallelism

- state contract の spec 更新と ADR 追記は並行で進められる
- scenario helper の追従は新しい state boundary が固まれば並行で進められる

## Risks and Mitigations

- 型分割だけで namespace が増え、責務は実質変わらない
  - mitigation: file split ではなく static rule / generated layout / mutable state の ownership を先に spec で固定する
- public/full/visible payload まで同時に崩すと runner/gamemaster への影響が広い
  - mitigation: external contract shape の互換を維持しつつ internal assembly だけを先に整理する
- 将来要素を見据えすぎて今必要ない state を先回り実装する
  - mitigation: actor/item/effect slot は責務位置だけ定義し、実データ追加は後続 plan に送る

## Design Decisions

- dungeon の次段階は turn loop 微修正ではなく domain model の再編を先に行う
- `ruleset definition` と `generated layout` は別物として扱う
- `Match` は façade に寄せ、state 所有と構築責務を分離する

