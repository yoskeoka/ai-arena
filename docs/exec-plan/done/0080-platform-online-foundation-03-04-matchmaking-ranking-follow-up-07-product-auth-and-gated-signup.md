# platform-online-foundation-03-04-matchmaking-ranking-follow-up-07-product-auth-and-gated-signup
**Execution**: Use `/execute-task` to implement this plan.

## Objective

public GA を見据えた product auth と gated signup の最小 contract を定義する。
最初のゴールは、Cloudflare Access を internal surface protection と切り分けつつ、
public-facing account lifecycle をどこから platform の責務にするかを固定することに置く。

## Context

- Cloudflare Access は staging / internal operator surface の保護には使えるが、
  free-tier user limit と product ownership の観点から public auth の代替ではない
- user decision として、Phase 6 では internal protection と release flow を優先し、
  public auth は後続 child plan へ分離する
- future signup は allowlisted account ID / invited account に限定する可能性がある

## Scope

- product-facing account identity / session / gated signup boundary を定義する
- internal surface protection と public auth の責務分離を定義する
- initial allowlist-based signup option を整理する

この plan では以下を扱わない。

- payment
- public profile / social graph
- full role matrix beyond initial operator / developer / participant split

## Spec Changes

- auth / account lifecycle spec を追加または更新する
- `docs/specs/platform-frontend-architecture.md` の auth boundary に必要な follow-up を追加する

## Expected Code Changes

- account / session backend
- gated signup path
- frontend auth shell

## Sub-tasks

- [ ] internal protection と public auth の boundary を固定する
- [ ] allowlist-based signup option を比較する
- [ ] initial account / session contract を定義する
- [ ] operator / developer / participant role の最小境界を整理する

## Parallelism

- [parallel] auth boundary 整理と signup option 比較は並行できる

## Dependencies

- depends on: `0075-platform-online-foundation-03-04-matchmaking-ranking-follow-up-02-internal-surface-protection-and-developer-access.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- internal Access protection と product auth を混ぜると implementation path がぶれる
  - mitigation: internal surface と public surface の auth responsibility を分ける

## Design Decisions

- Cloudflare Access は internal surface protection に寄せ、public auth は platform の product contract として別実装にする
