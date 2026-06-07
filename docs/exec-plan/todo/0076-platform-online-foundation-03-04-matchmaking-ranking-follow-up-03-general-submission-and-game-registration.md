# platform-online-foundation-03-04-matchmaking-ranking-follow-up-03-general-submission-and-game-registration
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 7 の first product lane として、general AI submission と game registration の
operator flow を定義する。
最初のゴールは、preset queue 専用 lane と切り離された durable entity と validation contract を固めることに置く。

## Context

- current service skeleton は `match submission` を 1 試合要求として扱う
- preset queue は Phase 6 confirmation 専用 lane であり、恒久的な AI submission registry ではない
- `platform-game-registry` は registered game lookup を定義しているが、
  online operator-facing registration metadata / admission surface はまだ薄い

## Scope

- AI submission の durable identity と validation flow を定義する
- game registration metadata と operator-facing validation surface を定義する
- preset queue lane と general submission lane の関係を定義する

この plan では以下を扱わない。

- ranking 集計
- rerun / retry policy
- public signup / self-service portal

## Spec Changes

### `docs/specs/platform-game-registry.md`

- online operator-facing registration metadata と validation surface を追記する

### 新規または更新 spec

- general AI submission / game registration entity と lifecycle spec を追加または更新する

## Expected Code Changes

- AI submission registry surface
- game registration operator surface
- admission / validation path

## Sub-tasks

- [ ] durable AI submission identity を定義する
- [ ] registered game metadata の operator-facing extension を定義する
- [ ] preset queue lane から general lane への変換点を決める
- [ ] validation / admission の verification を設計する

## Parallelism

- [parallel] AI submission entity と game registration metadata の整理は並行できる

## Dependencies

- depends on: `0074-platform-online-foundation-03-04-matchmaking-ranking-follow-up-01-phase6-release-flow.md`
- depends on: `0075-platform-online-foundation-03-04-matchmaking-ranking-follow-up-02-internal-surface-protection-and-developer-access.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- preset queue shape をそのまま恒久 entity にすると後で registration / ranking と噛み合わない
  - mitigation: general lane の entity を別 spec で固定してから接続する

## Design Decisions

- Phase 7 の first product entity は preset request ではなく durable AI submission / registered game とする
