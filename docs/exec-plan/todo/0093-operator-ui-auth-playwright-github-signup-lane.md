# operator-ui-auth-playwright-github-signup-lane
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`
の残課題として、
`GitHub account は存在するが ai-arena account はまだ存在しない`
状態からの first signup browser flow を
repo-owned Playwright lane で回帰確認できるようにする。

この plan は current `0091-operator-ui-auth-playwright-github-oauth-mock` が扱った
`existing account login -> callback -> session -> /operator`
lane を補完する child plan である。
goal は invite token を伴う
`/login -> /auth/github/login -> provider authorize -> callback -> account bootstrap -> session cookie -> /operator`
を deterministic に検証することにある。

Addresses: `docs/issues/done/0030-operator-ui-playwright-auth-coverage-gap.md`

## Context

- current auth regression lane は
  mock GitHub account と ai-arena account を同時に seed した
  existing account login path を主対象にしている
- user 指摘どおり、
  GitHub dummy account が存在しても ai-arena account は未作成、
  という signup-ready state を別に持たない限り、
  first signup invite flow の regression は捕まえられない
- current local helper `local-dummy-fixture` は
  invite token 発行用の manual helper であり、
  auth-enabled Playwright lane の deterministic signup source-of-truth にはなっていない
- current mock GitHub catalog は
  role 別 existing account 用 user と、
  invite 経由 signup 用 user を分離していない

## Scope

- mock GitHub test user catalog を
  `existing account` 用と `signup-only` 用に分離する
- auth regression bootstrap が seed する user と、
  seed しない signup-only user の contract を固定する
- signup-only user 向け invite token bootstrap と Playwright scenario を定義する
- browser verification lane に
  first signup callback / account bootstrap / role bind / session establishment を追加する
- local manual verification でも同じ signup-only user を使える runbook を整える

この plan では以下を扱わない。

- generic local OIDC provider の導入
- production での signup UX 拡張
- invite resend / invite management UI

## Option Snapshot

### Option A: existing login lane と signup lane を別 scenario として分離する

- 利点:
  regression failure の原因が
  `existing account login` なのか `first signup bootstrap` なのか切り分けやすい
- 欠点:
  Playwright scenario 数は 1 本増える

### Option B: existing login lane の中で signup case も切り替えて通す

- 利点:
  browser lane 本数は増えない
- 欠点:
  state 前提が混ざりやすく、
  seed 済み user と未 seed user の contract が曖昧になる

## Recommendation

- Option A を採る
  - existing login lane は `0091` の責務として維持する
  - signup lane は未 seed の GitHub dummy account と invite token を前提に分離する
  - mock GitHub catalog では少なくとも 1 user を
    `GitHub account あり / ai-arena account なし`
    の canonical signup fixture として固定する

## Spec Changes

- `docs/specs/platform-product-auth.md`
  - browser verification seam に
    `signup-only GitHub dummy account` の contract を追記する
  - mock GitHub catalog は
    existing account seed 対象と signup-only 対象を分けてよいことを明記する
- `docs/specs/platform-service-operator-ui.md`
  - auth-enabled GitHub regression lane に
    first signup scenario を追加する
  - invite token 付き login 開始から `/operator` 到達までの acceptance surface を追記する
- `docs/specs/platform-service-operator-api.md`
  - local / CI bootstrap が invite token 発行と signup-only user をどう扱うかを補足する
- `docs/development/operator-ui-local-verification.md`
  - signup lane の command、target user、invite bootstrap、artifact path を追記する

## Expected Code Changes

- mock GitHub catalog / bootstrap
  - existing account seed 対象 user
  - signup-only user
  - role と login 名の固定 contract
- local / CI bootstrap helper
  - signup lane 用 invite token 発行
  - signup-only user を seed 対象から外す制御
- Playwright scenario
  - invite token 付き `/login`
  - signup-only user で provider authorize
  - callback 後の authenticated session / role bind / `/operator` 到達確認
- 必要なら test helper / scenario selector
  - existing login lane と signup lane の分離

## Sub-tasks

- [ ] mock GitHub dummy account を existing account 用と signup-only 用に分ける
- [ ] auth seed が signup-only user を投入しない contract を固定する
- [ ] signup lane 用 invite token bootstrap を決める
- [ ] Playwright first signup scenario を追加する
- [ ] local manual verification runbook を更新する
- [ ] spec に `GitHub account あり / ai-arena account なし` state を明記する

## Parallelism

- [parallel] mock catalog 分離と runbook の整理は並行できる
- [parallel] invite bootstrap の設計と Playwright scenario 追加の下準備は並行できる

## Dependencies

- depends on: `docs/exec-plan/done/0091-operator-ui-auth-playwright-github-oauth-mock.md`
- depends on: `docs/exec-plan/done/0090-local-invite-result-summary.md`
- depends on: `docs/specs/platform-product-auth.md`
- depends on: `docs/specs/platform-service-operator-ui.md`

## Verification

- mock GitHub catalog に
  `GitHub account あり / ai-arena account なし`
  の signup-only user が存在する
- auth bootstrap 後も signup-only user に対して
  `GET /auth/session` は未認証のまま開始できる
- invite token 付き login 開始から callback 成功後、
  ai-arena account / identity / role が作成される
- callback 後に session cookie が成立し、
  `/operator` に到達できる
- existing login lane と signup lane が互いの state 前提を壊さない

## Risks and Mitigations

- existing login lane と signup lane が同じ dummy user を共有すると
  state 汚染で flaky になりやすい
  - mitigation:
    catalog 上で existing account user と signup-only user を明示的に分ける
- invite token bootstrap を browser lane 内に抱え込みすぎると
  failure 点が見えにくい
  - mitigation:
    invite 発行 helper と Playwright scenario の責務を分離する
- local helper 名が current behavior とずれると contributor が誤用しやすい
  - mitigation:
    `local-dummy-fixture` を signup lane の source-of-truth と見なさず、
    runbook に用途差分を明記する

## Design Decisions

- existing account login と first signup は別 regression scenario として扱う
- mock GitHub dummy account は
  `seed 対象 existing account` と
  `未 seed signup-only account` を併存させる
- first signup lane は
  `GitHub account はあるが ai-arena account はまだない`
  という state を first-class contract として持つ
