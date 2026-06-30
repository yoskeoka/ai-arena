# operator-invite-bootstrap-and-ui
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

staging / production の first operator bootstrap を、repo checkout を持つ開発者マシンから remote Postgres DSN を渡して実行できる `invite-remote` helper として固定する。
同時に、authenticated operator が operator UI 上で `participant|developer|operator` の role を選んで signup invite を発行できる最小運営導線を追加する。

completion boundary は次の 2 点である。

- human operator が local shell から remote Postgres DSN と frontend origin を渡し、stg/prod 向け operator invite URL を 1 回の command で取得できる
- authenticated operator が `/operator` route family 内の invite UI から role を選んで invite token / invite URL を発行でき、backend は unauthenticated request と non-operator request を拒否する

## Context

- current auth contract は GitHub login + gated signup であり、first signup は one-time invite token を前提にしている
- existing code では `signup-invite-create` CLI と `PostgresAuthStore.CreateSignupInvite` がすでに存在するが、remote bootstrap 用 helper と operator-facing HTTP/UI surface は未提供である
- user decision として、first operator bootstrap は Render shell ではなく、local machine から Neon Postgres DSN を直接与える `invite-remote` helper で行う
- user decision として、operator UI invite は role fixed にせず `participant|developer|operator` を選択できなければならない
- current role model に `admin` は存在しないため、この plan では new top role を増やさず、invite UI の最小 authorization は `authenticated operator session required` に留める

## Option Snapshot

### Option A: `invite-remote` helper と operator UI invite を 1 本の executable plan にまとめる

- 利点:
  - invite issuance contract を auth spec / operator API / operator UI で一度に揃えられる
  - bootstrap helper と in-product invite surface の role/TTL/output shape が分岐しにくい
- 欠点:
  - docs + backend + frontend + helper を同じ PR で扱うため diff はやや広い

### Option B: remote bootstrap helper と operator UI invite を別 child plan に分ける

- 利点:
  - first operator bootstrap と in-product invite management の責務が分かれる
- 欠点:
  - shared invite contract が 2 本の plan へ分散し、spec parity が崩れやすい
  - user が今回まとめて必要としている 2 点の到達までに review cycle が 1 回増える

## Recommendation

- Option A を採る
- bootstrap helper と operator UI invite は同じ `signup_invite` contract を使うため、1 本の executable plan にまとめる
- ただし scope は invite issuance に限定し、invite resend、invite list/history、`admin` role 導入は扱わない

## Existing Implementation References

- `internal/platform/service/auth.go`
  - `AuthStore`, lines 55-61
  - `AuthPrincipal`, `SessionStatusResponse`, lines 77-91
  - `GitHubLogin`, lines 175-205
  - `GitHubCallback`, lines 207-262
  - `RequireOperator`, lines 285-302
- `internal/platform/service/auth_postgres.go`
  - `SignupInviteRecord`, lines 27-32
  - `ResolveIdentityLogin`, lines 58-98
  - `CreateSignupInvite`, lines 128-150
  - `lookupPrincipalBySession`, lines 166-189
  - `createPrincipalFromInvite`, lines 209-255
- `internal/platform/service/http.go`
  - `OperatorAPI`, `NewOperatorAPI`, lines 77-129
  - `Handler`, lines 131-160
  - `handleSessionStatus`, lines 166-172
- `cmd/arena-service/main.go`
  - `runWithFactory` subcommand/flag handling, lines 36-118
  - `usageFor`, lines 177-203
- `tools/dev/local-invite-url.sh`
  - local operator invite helper contract, lines 1-40
- `Makefile`
  - `local-invite-url` target, lines 41-60
- `operator-ui/src/api.ts`
  - `AuthPrincipal`, `SessionStatusResponse`, lines 76-88
  - `OperatorApiClient` existing authenticated fetch pattern, lines 212-360
  - `decodeJSON` / error surfacing, lines 378-390
- `operator-ui/src/App.tsx`
  - `ProtectedOperatorRoute` auth gate and principal load, lines 43-107
  - route dispatch / unknown route surface, lines 109-177
- `operator-ui/src/routes/operator/operatorRoutes.ts`
  - `OperatorRoute`, `operatorNavItems`, `parseOperatorRoute`, lines 1-64
- `operator-ui/src/routes/operator/OperatorLayout.tsx`
  - principal header / nav shell, lines 15-83
- `docs/specs/platform-product-auth.md`
  - gated signup / durable auth / deferred follow-up, especially lines covering invite token contract and deferred `invite 発行 / 再送 UI`
- `docs/specs/platform-service-operator-api.md`
  - auth companion routes and authenticated operator route family
- `docs/specs/platform-service-operator-ui.md`
  - current `/operator` route family and auth-enabled acceptance surface
- `docs/development/operator-ui-local-verification.md`
  - current local invite helper note and auth local verification lane
- `docs/development/platform-service-online-deploy.md`
  - GitHub OAuth env / return origin inventory and remote operator-facing operational runbook

## Code Change Map

- `docs/specs/platform-product-auth.md` (MODIFY)
  - invite issuance contract, first operator bootstrap note, and current role selection scope for signup invites
- `docs/specs/platform-service-operator-api.md` (MODIFY)
  - add authenticated operator invite issuance route contract and request/response shape
- `docs/specs/platform-service-operator-ui.md` (MODIFY)
  - add `/operator/invites` route and minimal invite issuance UX contract
- `docs/development/platform-service-online-deploy.md` (MODIFY)
  - document `invite-remote` usage from a developer machine against remote Neon DSN
- `docs/development/operator-ui-local-verification.md` (MODIFY)
  - clarify that local helper remains operator-fixed bootstrap while in-product UI supports role selection
- `internal/platform/service/auth.go` (MODIFY)
  - extend auth store/service surface so invite issuance can be exposed through authenticated operator-only HTTP handling
- `internal/platform/service/auth_postgres.go` (MODIFY)
  - validate allowed invite roles and reuse `SignupInviteRecord` for HTTP/API issuance
- `internal/platform/service/http.go` (MODIFY)
  - add `POST /api/v1/signup-invites` under the authenticated operator surface
- `cmd/arena-service/main.go` (MODIFY)
  - keep CLI invite creation stable while ensuring it remains the backend primitive used by bootstrap helpers
- `tools/dev/invite-remote.sh` (NEW)
  - create remote operator bootstrap helper that accepts remote Postgres DSN and frontend origin, then prints operator invite JSON/url
- `Makefile` (MODIFY)
  - add `invite-remote` target without disturbing existing local invite targets
- `operator-ui/src/api.ts` (MODIFY)
  - add typed request/response helpers for signup invite creation
- `operator-ui/src/App.tsx` (MODIFY)
  - route `/operator/invites` through the existing protected operator shell
- `operator-ui/src/routes/operator/operatorRoutes.ts` (MODIFY)
  - add `invites` route/nav metadata
- `operator-ui/src/routes/operator/OperatorLayout.tsx` (MODIFY)
  - surface the new invites navigation entry in the shared shell
- `operator-ui/src/routes/operator/InvitesPage.tsx` (NEW)
  - implement the minimal role-select invite issuance page with token/url result display and basic error handling

## Spec Changes

- `docs/specs/platform-product-auth.md`
  - define that signup invites may be issued for `participant|developer|operator`
  - define that first operator bootstrap may be performed from a repo-owned helper on a developer machine against a remote Postgres DSN
  - keep `invite resend` / broader invite management deferred
- `docs/specs/platform-service-operator-api.md`
  - add `POST /api/v1/signup-invites`
  - require authenticated operator session in auth-enabled mode
  - define request fields (`role`, optional `ttl`) and response fields (`invite_token`, `role`, `expires_at`, derived `invite_url`)
- `docs/specs/platform-service-operator-ui.md`
  - add `/operator/invites` as a minimal operator page
  - define that the page can choose role, issue one invite, and show the resulting token / URL once

## Sub-tasks

- [ ] Update auth / operator specs for remote bootstrap and operator-issued invite contract
- [ ] [parallel] Add `invite-remote` helper and runbook guidance for developer-machine bootstrap against remote Postgres DSN
- [ ] Add backend invite issuance route under the authenticated operator surface
- [ ] Enforce the minimal role gate and allowed role validation for invite creation
- [ ] Add `/operator/invites` UI with role selector and one-shot result display
- [ ] Update local/deploy docs so local bootstrap remains operator-fixed while UI invite supports role selection
- [ ] Verify with targeted backend tests and operator-ui build/tests

## Design Decisions

- Do not introduce `admin` in this plan; current top role remains `operator`
- `invite-remote` is a bootstrap helper run from a developer machine and stays operator-fixed
- In-product invite issuance supports only the current durable roles `participant|developer|operator`
- The minimal authorization rule for the new HTTP/UI invite surface is `authenticated operator session required`

## Parallelism

- [parallel] spec updates and `invite-remote` helper/runbook drafting can proceed independently before backend wiring is finished
- [parallel] operator UI page scaffolding can start after the API request/response shape is fixed, while backend tests are added in parallel
- depends on: auth/operator API contract update before final UI verification

