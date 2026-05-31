# platform-online-foundation-03-operator-flow-and-matchmaking
**Execution**: Use `/execute-task` to implement this plan.

## Objective

service skeleton と persistence を前提に、operator が AI / game / match を継続運用できる最小 online flow を定義する。
最初のゴールは、submission と registered game を管理し、matchmaking / ranking / rerun を含む運営サイクルの骨格を作ることに置く。

## Context

- Phase 7 では AI 提出、game 提出、matchmaking、ranking、deploy pipeline を含む online 運営基盤が求められている
- ただし、いま必要なのは本格運営機能を一度に完成させることではなく、operator が最低限回せる flow を作ることである
- 01 が 1 本の online 実行フロー、02 が persistence/read model を担い、この plan はその上に運営導線を載せる位置づけになる

## Scope

- AI submission / game registration / match request / result publication を operator が扱える flow を定義する
- 最小 matchmaking policy、ranking update、rerun / retry の扱いを定義する
- 認証、本格 self-service portal、高度な ladder 運営、複数 tournament mode はこの plan では扱わない
- この plan は親 plan として扱い、実装前に submission registry、match scheduler、ranking/read model などの child plan へ分解する

## Spec Changes

### `docs/specs/platform-game-registry.md`

- online 運営で参照する registered game metadata と validation 入口の扱いを補足する

### 新規または更新 spec

- operator workflow / matchmaking / ranking lifecycle を扱う spec を追加する
- rerun / retry / cancellation の運営ルールを追加する

## Expected Code Changes

- AI submission / game registration を受ける operator-facing surface
- queued match から scheduling / ranking update まで進める orchestration
- rerun / retry / cancellation を扱う lifecycle code
- operator flow を確認する integration / acceptance test

## Sub-tasks

- [ ] operator が扱う最小エンティティと状態遷移を洗い出す
- [ ] matchmaking / ranking / rerun の最小ルールを spec に落とす
- [ ] [parallel] submission registry と result publication の read model 要件を整理する
- [ ] 実装前に child plan へ分解し、運営導線の execution order を確定する

## Parallelism

- operator entity の整理と result publication 要件の整理は並行できる
- child plan 分解後は submission / scheduling、ranking、verification を別 lane に切れる可能性がある

## Risks and Mitigations

- 運営機能を広く取りすぎると、Phase 7 が終わらず Phase 8/9 へ進めない
  - mitigation: まずは最小の operator flow と ranking 骨格に限定する
- submission / matchmaking / ranking を同時に厳密化すると、01/02 の未確定面まで凍結してしまう
  - mitigation: 先行 phase の実データを受けて、必要最小限の rule だけをこの段階で固定する

## Design Decisions

- operator-facing online flow は service skeleton と persistence の後続として扱う
- この plan 自体は parent/base plan であり、実装着手前に同じ `platform-online-foundation-03` 系の child plan へ分割する前提とする
