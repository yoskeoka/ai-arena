# Platform Product Auth 仕様

## 目的

このドキュメントは、`ai-arena` の first public auth として導入する
GitHub login + gated signup の最小 contract を定義する。

この段階で固定するのは、browser login route、backend callback、session cookie、
invite token 消費、operator surface 保護の責務境界である。
Google login、email login、MFA、profile 編集は扱わない。

## この spec の責務範囲

この spec が定義するもの:

- GitHub OAuth login start / callback / logout / session 確認 route
- `Cloudflare Pages` frontend と `Render` backend が split-origin のまま成立する
  session cookie contract
- initial account / identity / session / signup invite / role 保存単位
- first-login 時の gated signup contract
- frontend `/login` route と post-login return flow
- operator API が session と role をどう要求するか

この spec が定義しないもの:

- Google login
- email / password login
- passkey、TOTP、recovery code
- public profile 編集
- billing / payment

## 参照関係

- `docs/specs/platform-frontend-architecture.md`:
  route-first frontend と browser auth boundary の正本
- `docs/specs/platform-service-operator-api.md`:
  operator-facing HTTP route と CORS contract の正本
- `docs/specs/platform-service-persistence.md`:
  durable write model と metadata backend の正本
- `docs/development/platform-service-online-deploy.md`:
  GitHub OAuth app callback URL と secret inventory の正本

## Current Topology Decision

current deploy shape では、frontend は `Cloudflare Pages`、
operator API と session 発行主体は `Render` backend である。

したがって first landing の OAuth callback は backend 側に置く。

- login start:
  browser は frontend の `/login` から backend の
  `/auth/github/login` へ top-level redirect する
- callback:
  GitHub は backend の `/auth/github/callback` へ戻す
- session issuance:
  backend は callback 処理中に http-only cookie を発行する
- post-login return:
  backend は callback 成功後、frontend 側の return route へ redirect する

frontend が callback code を自力で受けて backend exchange API へ中継する構成は、
current split-origin では first landing の正本にしない。

## Route Contract

### Frontend Routes

- `/login`
  - 目的:
    GitHub login 開始、invite token 受け取り、login error 表示
- `/operator`
  - 目的:
    authenticated operator surface の canonical route
- `/`
  - first landing では `/operator` と同じ surface を出してよい

### Backend Routes

- `GET /auth/github/login`
  - query:
    - `return_to`
      - login 完了後に browser を戻す frontend URL
    - `invite_token`
      - first signup 時だけ消費する one-time token。省略可
  - 動作:
    pending login state を cookie に保存し、
    GitHub authorize URL へ redirect する
- `GET /auth/github/callback`
  - query:
    - `code`
    - `state`
  - 動作:
    pending login state を検証し、GitHub code exchange、
    account bootstrap / identity bind / session 発行を行い、
    `return_to` へ redirect する
- `GET /auth/session`
  - 目的:
    current browser session の auth mode と authenticated principal を返す
- `POST /auth/logout`
  - 目的:
    current session を失効し、session cookie を削除する

## Return Flow Contract

- frontend は protected route 到達時、
  anonymous browser を `/login?return_to=<target>` へ送ってよい
- `return_to` は absolute URL とし、
  backend は allowlisted frontend origin にしか redirect してはならない
- login success 後の default return は `/operator` とする
- login failure や invite failure は、
  frontend の `/login` へ `error` query を付けて戻してよい

## Browser Session Contract

- session の正本は backend 側の durable session store とする
- browser に保存するのは opaque session token を入れた http-only cookie だけとする
- frontend は bearer token を local storage / session storage に保持してはならない
- frontend から backend への fetch は cookie-based request context を前提にする
- current split-origin では、frontend の cross-origin fetch が backend cookie を送れるよう、
  session cookie は secure cross-site fetch に耐える属性で発行しなければならない
- backend CORS は allowlisted frontend origin に対して
  credentials 付き request を許可しなければならない

## Gated Signup Contract

- existing `account_identity(provider=github, provider_subject=...)` があれば、
  login は invite token なしで成立してよい
- existing identity がない browser は、valid な `signup_invite` を持つときだけ
  first signup を成立させてよい
- first signup 成功時は次を同一 flow で行う
  - new `account` 作成
  - new `account_identity(provider=github)` 作成
  - invite に紐づく role binding 付与
  - invite token 失効
- invalid / expired / consumed invite token では first signup を成立させてはならない
- operator の first signup を実地確認するため、
  repo-owned CLI から invite token を発行してよい

## Durable Auth Data Contract

first landing の auth metadata backend は `Postgres` とする。

少なくとも次の durable data family を持つ:

- `account`
  - platform 固有 account identity の正本
- `account_identity`
  - provider と provider subject を account へ bind する
- `account_session`
  - opaque browser session token の hash と expiry
- `signup_invite`
  - TTL + one-time signup token と role grant
- `account_role`
  - `participant|developer|operator` の role binding

provider 固有 subject や login name は `account` に混ぜず、
`account_identity` 側に寄せなければならない。

## Authorization Boundary

- operator API は authenticated session を要求しなければならない
- first landing の operator API write/read surface は `operator` role を要求してよい
- authenticated でも `operator` role を持たない account は
  operator API を成功させてはならない
- auth 未設定の local fixture lane では、
  operator API を auth-disabled mode で動かしてよい
  - ただし this mode は login flow の正本 verification とはみなさない

## Local Development Contract

- local backend callback URL は
  `http://localhost:10000/auth/github/callback`
- agent は `.env` を直接読まず、
  `direnv exec` または同等の shell hook 経由で human-managed local secret を読む
- auth env が未注入の lightweight fixture lane では、
  login flow を前提にしない verification を維持してよい

## Deferred Follow-Ups

- Google login
- invite 発行 / 再送 UI
- session rotation と refresh policy の hardening
- CSRF detail hardening
- frontend same-origin proxy / custom domain 化
