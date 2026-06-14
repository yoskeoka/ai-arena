# product-auth-and-gated-signup-01-github-login-and-account-linking-foundation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

public auth の最初の実装として、GitHub login だけで account 作成と session 開始が成立する最小 contract を定義する。
最初のゴールは、same-origin + http-only cookie session を前提に、
Postgres 上の account / account identity / invite / role 境界を固定し、
後から Google や email identity を migration なしで追加できる土台を整えることに置く。

## Context

- `0080-platform-online-foundation-03-04-matchmaking-ranking-follow-up-07-product-auth-and-gated-signup.md`
  では product auth の責務分離と gated signup の論点整理を行い、
  implementation 向けの child plan へ分離する判断になった
- browser 向け auth は `docs/specs/platform-frontend-architecture.md` で
  same-origin + http-only cookie first が既定になっている
- user decision として、最初の実装は GitHub login のみを対象にし、
  Google login は GitHub 検証後に同じ contract へ追加できる前提を保つ
- user decision として、account は social provider 非依存の
  `ai-arena` 固有 ID を正本にし、provider ごとの login 情報は
  `account_identity` に分離する
- user decision として、invite 時に GitHub handle の rename へ依存しないよう、
  初回 signup gate は provider subject の事前 allowlist より
  TTL + one-time signup link を優先する
- user decision として、login 関連の durable data は PostgreSQL に保存する
- user decision として、GitHub login 実現に必要な provider-side 準備
  (OAuth app / callback URL / secret inventory など) も
  execution 前に見落とさないよう plan に含める

## Provider Bootstrap References

- GitHub OAuth app creation:
  https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app
- GitHub OAuth web application flow:
  https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
- Render environment variables and secrets:
  https://render.com/docs/configure-environment-variables
- Cloudflare Workers secrets:
  https://developers.cloudflare.com/workers/configuration/secrets/

これらの URL は、execution 時に GitHub OAuth app 作成、callback URL 登録、
secret 配置、redirect URI の確認をやり直すための正本参照として plan に残す。

## Option Snapshot

### Option A: GitHub handle / email allowlist で gated signup する

- 利点:
  invite 発行の実装は軽い
- 欠点:
  GitHub handle rename に弱く、provider subject を招待前に把握しづらい

### Option B: TTL + one-time signup link を発行し、claim 時に GitHub identity を bind する

- 利点:
  invite 時に provider subject の事前調査が不要で、
  rename 問題を避けつつ初回 account 作成を gate できる
- 欠点:
  invite token lifecycle と claim endpoint を追加で持つ必要がある

## Recommendation

- Option B を採用する
- 最初の gated signup は `signup_invite` の TTL + one-time token を入口にし、
  GitHub login 完了後に `account` と `account_identity(provider=github)` を作成する
- token は signup 成功時に即時失効させ、再利用を許可しない
- `identity_allowlist` は最初の signup gate の正本にせず、
  将来の provider subject 既知 invite や追加 identity 連携の補助機能として扱う

## Scope

- GitHub login による public-facing account 作成 / login / logout / session 開始 contract を定義する
- `account` と `account_identity` を分離した PostgreSQL 正本 schema を定義する
- TTL + one-time invite token による gated signup contract を定義する
- participant / developer / operator role の最小境界を定義する
- Google や email identity を後から足せる provider-agnostic identity contract を定義する
- platform 運営側が GitHub login のために事前準備すべき provider bootstrap を定義する

この plan では以下を扱わない。

- Google login の実装
- email/password login の実装
- passkey / TOTP / recovery flow
- payment
- public profile / social graph

## Spec Changes

- auth / account lifecycle spec を新規追加し、
  account、account identity、invite、session、role の責務を定義する
- `docs/specs/platform-frontend-architecture.md` に、
  login entry / auth shell / provider add-on 前提の boundary 補足を追加する
- `docs/development/platform-service-online-deploy.md` または同等 runbook に、
  GitHub OAuth app 作成、callback URL、client secret 登録、
  環境ごとの redirect URI と secret inventory を追記する
- operator / general submission 系 spec に、
  participant / developer / operator の最小権限境界への cross-reference を追加する

## Expected Code Changes

- PostgreSQL schema / migration:
  `account`、`account_identity`、`account_session`、`signup_invite` と
  必要最小限の role binding table 群
- GitHub OAuth login / callback / logout / session verification backend
- invite claim backend と first-login account bootstrap path
- frontend auth shell と GitHub login entry、
  invite claim から login 完了へ進む browser flow
- GitHub provider bootstrap runbook と env / secret inventory 更新

## Sub-tasks

- [ ] `account` と `account_identity` の provider-agnostic schema を定義する
- [ ] same-origin + http-only cookie session contract を定義する
- [ ] TTL + one-time signup invite と claim flow を定義する
- [ ] GitHub login 専用の first provider contract を定義する
- [ ] GitHub OAuth app / callback URL / secret inventory / environment wiring の運用準備を定義する
- [ ] participant / developer / operator の最小 role 境界を定義する
- [ ] Google / email identity を migration なしで追加できる拡張点を明記する

## Secret Inventory Baseline

- primary secret placement:
  current backend が `Render` 上の `arena-service` で token exchange / session 発行を担う前提なら、
  `GITHUB_OAUTH_CLIENT_ID` と `GITHUB_OAUTH_CLIENT_SECRET` は
  Render service の Environment Variables を第一選択にする
- environment grouping:
  staging / production で同一 key を共有したい場合は、
  Render Environment Group を使って service へ配布してよい
- Cloudflare fallback:
  callback handler や token exchange を Cloudflare Worker / Pages Functions へ移す場合に限り、
  Cloudflare Secrets を代替候補として扱う
- do not default to dual write:
  同じ GitHub OAuth secret を Render と Cloudflare の両方へ常設する前提にはしない
  - どちらが token exchange の実行主体かを先に固定し、その実行面だけに secret を置く

## Parallelism

- [parallel] account/session schema 整理と invite flow 整理は並行できる
- [parallel] role boundary 整理と frontend auth shell 整理は並行できる

## Dependencies

- depends on: `0075-platform-online-foundation-03-04-matchmaking-ranking-follow-up-02-internal-surface-protection-and-developer-access.md`
- depends on: `0076-platform-online-foundation-03-04-matchmaking-ranking-follow-up-03-general-submission-and-game-registration.md`
- depends on: `0082-platform-service-db-migration-release-flow.md`
- depends on: parent/base item `0080-platform-online-foundation-03-04-matchmaking-ranking-follow-up-07-product-auth-and-gated-signup.md` (retired after child split; lives under `docs/exec-plan/done/`)

## Risks and Mitigations

- provider 固有 field を `account` へ混ぜると Google / email 追加時に migration が増える
  - mitigation: provider 由来の subject / email / profile metadata は `account_identity` に寄せる
- GitHub handle allowlist を signup gate に使うと rename と typo に弱い
  - mitigation: invite 正本は TTL + one-time token にし、identity bind は login 完了時に行う
- participant と developer を混同すると game developer 権限と AI competitor 権限が混ざる
  - mitigation: AI submit の主体は participant とし、developer は game/provider integration 側へ寄せる
- provider-side 準備を code plan から外すと、OAuth app 未作成や callback mismatch で execution が止まる
  - mitigation: GitHub OAuth app 設定、redirect URI、secret 登録先、環境差分を plan と runbook へ明記する

## Design Decisions

- public auth の first landing は GitHub login only とする
- `ai-arena` 固有 account ID を正本にし、provider login 情報は `account_identity` に分離する
- 初回 signup gate は TTL + one-time invite token を正本にし、provider subject の事前把握を要求しない
- Google や email identity は後続 plan で `account_identity.provider` の追加として扱える形を保つ
