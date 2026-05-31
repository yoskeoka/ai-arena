# platform-online-foundation-03-04-matchmaking-ranking-follow-up
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0047` 親 scope に残っていた broader operator flow を引き継ぎ、
general submission / game registration / matchmaking / ranking / rerun を後続 lane として整理する。
最初のゴールは、Phase 6 remote confirmation を壊さずに、Phase 7 本来の運営機能をどこまで次段で扱うかを絞ることに置く。

## Context

- user decision として、いま優先したいのは preset queue と active/completed visibility を持つ Phase 6 online confirmation である
- しかし `0047` 親 plan はもともと operator flow / matchmaking / ranking / rerun をまとめて受け持つ plan だった
- split-parent では親 scope を無言で捨てず、後続 child plan として明示する必要がある
- Phase 6 remote landing ができると、broader matchmaking/ranking は実データと運用感を踏まえて詰めやすくなる

## Scope

- general AI submission / game registration の operator flow を整理する
- preset ではない match request / scheduling policy を整理する
- ranking update の最小 contract を整理する
- rerun / retry / cancellation の運営ルールを Phase 7 scope として切り出す

この plan では以下を扱わない。

- Phase 6 first landing の Pages UI
- first remote deploy bootstrap
- delegated artifact download contract の基礎設計

## Spec Changes

### `docs/specs/platform-game-registry.md`

- operator-facing game registration metadata と validation surface の拡張を追記する

### 新規または更新 spec

- general submission / scheduling / ranking / rerun lifecycle spec を追加または更新する
- preset queue から general operator flow へ広げるときの entity/state model を定義する

## Expected Code Changes

- submission / registration operator surface の拡張
- scheduling / matchmaking orchestration
- ranking update path
- rerun / retry / cancellation lifecycle verification

## Sub-tasks

- [ ] parent `0047` から残る broader scope を棚卸しする
- [ ] preset queue lane と general operator lane の境界を明文化する
- [ ] matchmaking / ranking / rerun の最小 contract を再整理する
- [ ] Phase 6 remote landing 後の execution order を定義する

## Parallelism

- scope 棚卸しと entity/state model 整理は並行できる
- Phase 6 landing 後は submission lane と ranking lane を分担できる可能性がある

## Dependencies

- depends on: `0061-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- depends on: `0062-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`
- depends on: parent/base item `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md` (now retired to `docs/exec-plan/done/` after split)

## Risks and Mitigations

- Phase 6 confirmation 前に broader operator flow へ戻ると、また scope が広がる
  - mitigation: this plan explicitly follows the remote confirmation child plans and does not block them
- preset lane と general lane の境界が曖昧だと一時実装が恒久契約化する
  - mitigation: preset queue は first landing 専用 lane と明記し、general operator contract を別 spec で固定する

## Design Decisions

- `0047` 親 scope のうち immediate Phase 6 confirmation 以外は、この follow-up child plan へ送る
- broader operator flow は remote landing の実観測を踏まえてから詰める
