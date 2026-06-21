# operator-ui-auth-playwright-github-oauth-mock
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`
のうち、current GitHub login を本命 auth hand としたまま
`/auth/github/login -> provider authorize -> callback -> session cookie -> /operator`
の browser flow を repo-owned Playwright lane で回帰確認できるようにする。

この plan は product の login 手段を増やさない。
local / CI verification のために GitHub OAuth provider 相当の test double を持ち込み、
current public route と backend callback / session contract の回帰を主目的とする。

Addresses: `docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`

## Context

- `docs/specs/platform-product-auth.md` の current public auth contract は
  GitHub login first であり、backend callback + session cookie authority を前提にしている
- current Playwright lane は auth-disabled fixture backend または
  login を通らない managed backend に依存しているため、
  GitHub login start / callback / session establishment の回帰を repo-owned に捕まえられていない
- user の意図は GitHub OAuth login 自体を product login として維持しつつ、
  provider を増やすたびに個別 mock login 手段を product に増やさない verification seam を持つことにある
- `docs/exec-plan/done/0089-oauth-library-adoption.md` により、
  backend 内部には provider boundary と optional OIDC verifier seam が入り始めているが、
  current 本流 browser route は依然として GitHub login である

## Option Snapshot

### Option A: GitHub login の backend route はそのままに、local/CI だけ GitHub OAuth provider 相当の test double へ向ける

- 利点:
  current public route と callback / session contract をそのまま検証できる
- 欠点:
  provider ごとに test double を増やしたくなる圧力は残る

### Option B: GitHub login 回帰は手動確認に残し、local/CI は別の local auth provider だけを自動化する

- 利点:
  generic な local provider へ寄せやすい
- 欠点:
  current 本命の GitHub login route 回帰を repo-owned automation で捕まえられない

## Recommendation

- current 本命 login hand の regression capture を優先し、この child plan では Option A を扱う
- ただし product login hand は増やさず、
  test double は local / CI verification seam に限定する
- generic local provider 追加は sibling plan
  `0092-operator-ui-auth-playwright-local-oidc-provider.md` で別に扱う

## Scope

- GitHub login route の browser flow を自動 verification できる test seam を定義する
- local / CI だけで起動する provider test double の責務と bootstrap を定義する
- Playwright lane から tester account login、callback、session establishment、
  protected operator route 到達までを通す
- first signup invite flow をこの lane に含めるか、既存 account login と分けるかを整理する
- auth-enabled backend 起動時の schema apply bootstrap 責務を明確にする

この plan では以下を扱わない。

- product の新しい login provider 追加
- Google login 実装
- public UI に local/CI 専用 provider を常設表示すること

## Spec Changes

- `docs/specs/platform-product-auth.md`
  - local / CI verification seam として、
    GitHub login route が test double provider へ向く lane を持ってよいことを追記する
  - ただし public product login hand は増やさないことを明記する
- `docs/specs/platform-service-operator-ui.md`
  - browser verification lane に
    `GitHub login regression lane` を追加し、
    login page -> provider form -> callback -> operator surface を acceptance surface に含める
- `docs/specs/platform-service-operator-api.md`
  - auth-enabled browser lane の companion-route verification と
    schema bootstrap 前提を補強する
- `docs/development/operator-ui-local-verification.md`
  - local auth regression lane の起動方法、fixture account、artifact path、
    provider test double の責務を追記する

## Expected Code Changes

- auth provider bootstrap / config
  - local / CI verification 時だけ GitHub route の upstream を test double へ切り替える設定
- provider test double
  - authorize endpoint
  - fixed tester account login form
  - token exchange endpoint
  - identity/profile endpoint
- Playwright helper / scenario
  - provider login form 通過
  - callback 完了後の session status と operator page 到達確認
- local / CI backend entrypoint
  - auth schema apply bootstrap
  - tester account seed / invite bootstrap responsibility

## Sub-tasks

- [ ] current GitHub login route の回帰として最低限検証したい browser contract を固定する
- [ ] local / CI 専用 provider test double の責務と最小 endpoint 群を定義する
- [ ] fixed tester account と invite bootstrap の扱いを決める
- [ ] auth-enabled backend 起動時の schema apply bootstrap entrypoint を決める
- [ ] Playwright lane に login start / callback / session establishment / logout を追加する
- [ ] docs/specs と local verification runbook を更新する

## Parallelism

- [parallel] provider test double の責務整理と Playwright acceptance surface 定義は並行できる
- [parallel] schema/bootstrap の runbook 更新は provider test double 実装設計と並行できる

## Dependencies

- depends on: `docs/specs/platform-product-auth.md`
- depends on: `docs/specs/platform-service-operator-ui.md`
- depends on: `docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`
- depends on: `docs/exec-plan/done/0089-oauth-library-adoption.md`

## Verification

- Playwright から `/login` 経由で GitHub login start を起こせる
- provider form で fixed tester account を通し、backend callback が成功する
- callback 後に `GET /auth/session` が authenticated principal を返す
- protected operator route が session cookie 付き browser で到達できる
- local / CI lane で auth table 未作成のまま詰まらず、責務を持つ bootstrap が明示されている

## Risks and Mitigations

- GitHub route 専用の seam を持つことで provider ごとの test double 増殖圧力が残る
  - mitigation:
    この plan の目的を current 本命 login の regression capture に限定し、
    generic local provider は sibling plan に分離する
- test double が GitHub 実サービスの細部に寄りすぎると保守負担が増える
  - mitigation:
    authorize / token / identity の最小 contract に絞り、
    browser callback と backend session issuance の回帰だけを狙う
- invite bootstrap と existing account login を同時に詰め込みすぎると lane が不安定になる
  - mitigation:
    最初は existing tester account login を優先し、
    first signup verification は別シナリオ扱いを許容する

## Design Decisions

- current 本命 login route の回帰を repo-owned automation で捕まえる価値を認める
- ただし test seam は product login hand を増やす設計にしない
- local/CI verification のための provider test double は
  backend callback + session cookie authority を検証する補助面として扱う
