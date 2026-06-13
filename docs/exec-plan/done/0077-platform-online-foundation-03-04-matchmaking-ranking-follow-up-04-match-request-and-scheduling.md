# platform-online-foundation-03-04-matchmaking-ranking-follow-up-04-match-request-and-scheduling
**Execution**: Use `/execute-task` to implement this plan.

## Objective

general operator lane で使う match request と scheduling policy を定義する。
最初のゴールは、preset match enqueue を超えて、operator がどの単位で対戦要求を作り、
service がどの最小 policy で実行順を決めるかを固定することに置く。

## Context

- current remote lane は server-known preset から 1 試合を queue へ積むだけである
- Phase 7 では preset 以外の対戦要求と最小 matchmaking policy が必要になる
- queue / worker / runner の土台は Phase 6 である程度成立しているため、
  ここでは request entity と scheduling responsibility を切り出すのが主眼になる

## Scope

- operator-created match request entity を定義する
- minimal scheduling / queue selection policy を定義する
- preset request と general request の共存方法を定義する

この plan では以下を扱わない。

- ranking update
- rerun / retry policy
- multi-worker fairness の本格設計

## Spec Changes

- general match request / scheduling lifecycle spec を追加または更新する
- `docs/specs/platform-service-skeleton.md` に必要なら general request 入口との関係を補足する

## Expected Code Changes

- match request operator surface
- scheduling / queue orchestration path
- request visibility verification

## Sub-tasks

- [ ] operator-created match request entity を定義する
- [ ] preset lane と general lane の queue 接点を定義する
- [ ] minimal scheduling rule を決める
- [ ] verification / observability surface を整理する

## Parallelism

- [parallel] request entity の設計と observability requirement の整理は並行できる

## Dependencies

- depends on: `0076-platform-online-foundation-03-04-matchmaking-ranking-follow-up-03-general-submission-and-game-registration.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- scheduling を distributed fairness まで広げると Phase 7 first pass が止まる
  - mitigation: single logical queue authority 前提の minimal policy に限定する

## Design Decisions

- first matchmaking policy は preset queue の延長ではなく、general match request の最小 scheduling として定義する
