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
- backend auth provider boundary と provider 差分吸収責務
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

## Backend Auth Provider Boundary

- public browser route は current の GitHub first contract を維持する
  - `GET /auth/github/login`
  - `GET /auth/github/callback`
- backend 内部では provider descriptor が少なくとも次の責務を持つ
  - authorization URL generation
  - callback code exchange
  - provider response から normalized identity claim への変換
- normalized identity claim は少なくとも次を持たなければならない
  - `provider`
  - `subject`
  - `login` または display name
  - `email`
- session 発行、invite 検証、role bind、`account_identity` 永続化は
  provider 実装ではなく backend app responsibility とする
- current GitHub login は access token を得た後に provider API から identity claim を取得する
  OAuth provider として扱う
- 将来の Google login のように ID token を返す provider は、
  OAuth transport に加えて OIDC identity verification を持つ provider として扱う

## Supported Library Policy

- OAuth provider の authorization URL generation と token exchange は
  supported library で扱わなければならない
  - current first provider の GitHub は `golang.org/x/oauth2`
- OIDC provider の ID token verification と claim validation は
  supported library で扱わなければならない
  - future provider seam は `github.com/coreos/go-oidc/v3/oidc` を前提にしてよい
- OAuth transport と OIDC identity verification は同じ責務として潰さず、
  transport は全 provider 共通、ID token verification は capability を持つ provider だけが要求する
- frontend は provider library を持たず、backend redirect / callback / session contract に依存する

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

## Browser Verification Seam

- current public login hand は GitHub login first のまま維持しなければならない
  - `/auth/github/login`
  - provider authorize
  - `/auth/github/callback`
  - backend session cookie
- local / CI の auth regression lane では、
  public route を変えずに provider upstream だけ repo-owned test double へ切り替えてよい
- この seam は verification 専用であり、
  product に local 専用 login provider や別 public login button を増やしてはならない
- backend は provider endpoint override として次だけを受け取ってよい
  - `ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL`
  - `ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL`
- endpoint override は local / CI verification seam のためにだけ使ってよく、
  authorize / token / user の個別 URL override を product contract に増やしてはならない
- provider test double は backend process と別 process で起動し、
  次の最小 contract だけを持てばよい
  - browser が到達する authorize form
  - backend が呼ぶ token exchange endpoint
  - backend が呼ぶ identity/profile endpoint
- provider test double は canonical test user catalog を自前で持ってよい
  - 例:
    `spectator-user01`、`developer-user01`、`operator-user01`
  - catalog は seed 済み existing account user と
    ai-arena account をまだ持たない signup-only user を分けて持ってよい
  - browser form は password を要求せず、
    `user_id` text input と login button だけでよい
  - available test users の一覧を form 上に hint 表示してよい
- auth regression lane の backend 側 bootstrap は、
  provider test double と同じ test user catalog を使って
  role 付き account / identity を事前作成してよい
  - provider test double 起動時に same catalog を idempotent seed してよい
  - signup-only user は seed 対象から外し、
    first signup invite flow の callback で初めて account bootstrap されなければならない
  - first signup invite flow は別 verification scenario として分離してよい
- auth regression lane を起動する entrypoint は、
  auth table 未作成で詰まらないよう schema apply bootstrap を明示的に担わなければならない

## Deferred Follow-Ups

- Google login
- invite 発行 / 再送 UI
- session rotation と refresh policy の hardening
- CSRF detail hardening
- frontend same-origin proxy / custom domain 化
