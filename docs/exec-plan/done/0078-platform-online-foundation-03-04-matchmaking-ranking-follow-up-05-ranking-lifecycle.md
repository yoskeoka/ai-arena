# platform-online-foundation-03-04-matchmaking-ranking-follow-up-05-ranking-lifecycle
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 7 の ranking lifecycle を定義する。
最初のゴールは、試合結果の compact summary からどの単位で ranking を更新し、
どこまで durable read model として持つかを固定することに置く。

## Context

- current read model は 1 match の compact summary / detail までであり、
  leaderboard や standings はまだ存在しない
- project-plan 上は Phase 7 が ranking を含む online operator cycle を求めている
- ranking を単なる view にするか durable aggregate にするかは、後続の rerun / replay / audit と強く結びつく

## Scope

- ranking input / aggregate / read model の最小 contract を定義する
- re-compute と durable snapshot の責務分離を定義する
- leaderboard family への引き継ぎ境界を定義する

この plan では以下を扱わない。

- public leaderboard UI
- tournament bracket
- season / division / reward system

## Spec Changes

- ranking / leaderboard lifecycle spec を追加または更新する
- `docs/specs/platform-service-read-model.md` に必要なら ranking 参照境界を補足する

## Expected Code Changes

- ranking aggregate path
- ranking read surface
- recompute / verification helper

## Sub-tasks

- [ ] ranking input をどの artifact / compact summary から取るか決める
- [ ] durable aggregate と recompute path の責務を決める
- [ ] minimal ranking read surface を定義する
- [ ] rerun / retry と整合する更新 rule を定義する

## Parallelism

- [parallel] ranking input 棚卸しと read surface 叩き台作成は並行できる

## Dependencies

- depends on: `0077-platform-online-foundation-03-04-matchmaking-ranking-follow-up-04-match-request-and-scheduling.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- ranking を単発 view のまま始めると rerun 時の整合が壊れやすい
  - mitigation: input source と recompute rule を先に固定する

## Design Decisions

- ranking は compact summary の派生 view ではなく、recompute 可能な aggregate lifecycle として扱う
