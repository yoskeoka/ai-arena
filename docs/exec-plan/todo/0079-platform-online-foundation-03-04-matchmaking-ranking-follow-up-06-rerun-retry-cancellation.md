# platform-online-foundation-03-04-matchmaking-ranking-follow-up-06-rerun-retry-cancellation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

operator が rerun / retry / cancellation をどう扱うかを定義する。
最初のゴールは、execution failure、operator judgment、ranking correction の各理由で
どの entity を作り直し、どの履歴を残すかを固定することに置く。

## Context

- current lane では queued-only cancel はあるが、rerun / retry / correction path は未定義である
- ranking aggregate を入れるなら、どの再実行が順位へ影響するかを先に整理する必要がある
- audit / replay artifact は Phase 6 でそろっているため、ここでは operator rule を固める価値が高い

## Scope

- queued-only cancel と post-run rerun の違いを定義する
- retry と fresh rerun の entity / history difference を定義する
- ranking correction と audit trail の扱いを定義する

この plan では以下を扱わない。

- public dispute workflow
- self-service user appeal flow

## Spec Changes

- rerun / retry / cancellation lifecycle spec を追加または更新する
- ranking lifecycle spec と必要な cross-reference を追加する

## Expected Code Changes

- rerun / retry command surface
- lifecycle persistence / audit updates
- verification scenario

## Sub-tasks

- [ ] queued cancel、retry、rerun の定義を切り分ける
- [ ] history / audit trail contract を定義する
- [ ] ranking correction rule を定義する
- [ ] operator verification scenario を整理する

## Parallelism

- [parallel] lifecycle vocabulary 整理と verification scenario 叩き台は並行できる

## Dependencies

- depends on: `0078-platform-online-foundation-03-04-matchmaking-ranking-follow-up-05-ranking-lifecycle.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- retry と rerun を同一視すると ranking / audit の意味が曖昧になる
  - mitigation: entity と reason taxonomy を先に固定する

## Design Decisions

- queued-only cancel、execution retry、operator rerun は別 lifecycle として扱う
