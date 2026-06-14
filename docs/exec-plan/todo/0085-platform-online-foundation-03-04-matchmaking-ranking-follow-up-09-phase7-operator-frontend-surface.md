# platform-online-foundation-03-04-matchmaking-ranking-follow-up-09-phase7-operator-frontend-surface
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 7 の operator frontend surface を定義し、先行実装済み backend API を
browser UI から実際に使える運営導線へつなぐ。
最初のゴールは、game registration、AI submission、general match request、
ranking、rerun/retry/cancellation を `operator-ui` から辿れる最小 UI contract を固定することに置く。

## Context

- `0076` と `0077` で game registration、AI submission、match request の backend API が先行した
- `0078` と `0079` では ranking lifecycle と rerun/retry/cancellation の backend/operator contract が続く
- current `operator-ui` は preset enqueue、active/completed polling、detail panel までであり、
  general operator lane の canonical flow をまだ表現していない
- project-plan の Phase 7 は online operator flow を成立させることを求めており、
  backend API だけでは運営基盤として未完成である

## Scope

- Phase 7 backend API を使う operator/admin browser workflow を定義する
- `operator-ui` の nav / route-local state / panel 構成を拡張する
- registration -> submission -> match request -> ranking / rerun / retry / cancellation までの browser verification を整える

この plan では以下を扱わない。

- public leaderboard / spectator UI
- signup / login form そのもの
- operator 以外の page family への本格展開

## Spec Changes

- `docs/specs/platform-service-operator-ui.md` を Phase 7 operator flow に合わせて更新する
- 必要なら `docs/specs/platform-frontend-architecture.md` に
  operator page family の Phase 7 nav / route boundary を補足する
- ranking / rerun surface との cross-reference を追加する

## Expected Code Changes

- `operator-ui` nav / route shell と API client / payload type 拡張
- game registration / AI submission / match request / ranking / rerun の route-local panel 群
- operator browser verification scenario と fixture/backend seeding 更新

## Sub-tasks

- [ ] current preset-first UI と Phase 7 canonical operator flow の差分を spec に落とす
- [ ] game registration / AI submission / ranking を無理なく置ける nav / route shape を決める
- [ ] game registration / AI submission の一覧・作成 surface を定義する
- [ ] general match request composer と accepted request visibility surface を定義する
- [ ] ranking / rerun / cancellation surface を backend lifecycle と整合させる
- [ ] browser verification を registration -> submission -> request -> result -> ranking correction まで拡張する

## Parallelism

- [parallel] registration/submission UI spec と ranking/rerun UI spec の叩き台作成は並行できる
- [parallel] API client type 整理と browser verification scenario 整理は並行できる

## Dependencies

- depends on: `0076-platform-online-foundation-03-04-matchmaking-ranking-follow-up-03-general-submission-and-game-registration.md`
- depends on: `0077-platform-online-foundation-03-04-matchmaking-ranking-follow-up-04-match-request-and-scheduling.md`
- depends on: `0078-platform-online-foundation-03-04-matchmaking-ranking-follow-up-05-ranking-lifecycle.md`
- depends on: `0079-platform-online-foundation-03-04-matchmaking-ranking-follow-up-06-rerun-retry-cancellation.md`
- coordinates with: `0087-product-auth-and-gated-signup-01-github-login-and-account-linking-foundation.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- 0078/0079 完了前に UI contract を固定しすぎると ranking / rerun API の最終 shape とずれる
  - mitigation: backend lifecycle spec を前提にしつつ、frontend plan の着手順は 0079 後を基本に置く
- nav を決めないまま実装へ入ると、registration / ranking / rerun を current 画面へ無理に押し込みやすい
  - mitigation: execution 開始前までに nav が未確定なら、spec-first で nav / route shape を先に決め切る

## Design Decisions

- Phase 7 frontend は preset queue を補助 entry に下げ、general operator flow を canonical surface として扱う
- Phase 7 frontend は current 1 画面への押し込みを前提にせず、必要なら operator nav / route を先に分割してよい
