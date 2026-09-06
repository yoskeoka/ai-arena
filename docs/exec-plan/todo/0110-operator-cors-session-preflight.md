# operator-cors-session-preflight
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

staging と production の canonical Cloudflare Pages origin から operator UI が
credentials 付きで `GET /auth/session` を呼び出すとき、TypeSpec HTTP runtime が送る
`x-ms-useragent` による CORS preflight を通過させる。未認証 session は transport/CORS error ではなく
`200` の `authenticated: false` として UI の login flow に渡さなければならない。

完了境界は、許可済み staging / production origin だけが `Content-Type` と
`x-ms-useragent` を要求する preflight で CORS credentials contract を得られ、unknown origin または
許可していない requested header は CORS 許可を得られないことである。session cookie の属性・発行、OAuth
flow、route の payload、runtime dependency version、origin allowlist 自体は変更しない。

## 根本原因と採用済みの方針

`operator-ui/src/lib/operatorApiClient.ts` は `getClient` の default browser pipeline を使い、local
`credentialsPolicy` は `request.withCredentials = true` のみを設定する。`@typespec/ts-http-runtime` 0.2.1 の
default pipeline は `userAgentPolicy` を追加し、browser platform 実装では header 名を
`x-ms-useragent` とする。そのため cross-origin session request は preflight される。

一方 `internal/platform/service/http.go` の `applyOperatorCORSHeaders` は許可済み origin に
`Access-Control-Allow-Headers: Content-Type` だけを返す。browser は `x-ms-useragent` を許可されないため
actual `GET /auth/session` を送らず、UI は `Failed to fetch` を session error として表示する。

実装は次の固定方針に従う。実装者が別案を選ぶ余地はない。

- backend の CORS request-header allowlist は `Content-Type` と `x-ms-useragent` の2個だけとする。
- `OPTIONS` の `Access-Control-Request-Headers` は comma 区切りで parse し、trim 後の空要素を捨て、
  大小文字を区別せず全 token が allowlist に含まれることを判定する。
- exact origin allowlist と requested-header 判定の両方を満たす preflight だけが、
  `Access-Control-Allow-Headers: Content-Type, x-ms-useragent`、既存の methods、
  `Access-Control-Allow-Credentials: true`、request の origin を受け取る。
- unknown origin、または既知 origin でも allowlist 外の requested header を含む preflight は `204` のままとし、
  `Access-Control-Allow-Origin`、`Access-Control-Allow-Headers`、`Access-Control-Allow-Methods`、
  `Access-Control-Allow-Credentials` を返さない。任意 header の echo や wildcard は使わない。
- `Access-Control-Request-Method` の個別検証は今回追加しない。methods の contract は既存どおり
  `GET, POST, OPTIONS` の固定値とする。
- `operator-ui` の runtime dependency、`userAgentPolicy`、credentials policy、session cookie code は変更しない。
  JSON POST、multipart upload、runtime telemetry header は同じ CORS contract を通す。

`x-ms-useragent` を UI policy で削除する案は採らない。runtime の default telemetry policy に依存した順序で
header を除去する必要があり、dependency update で再発しやすく、session 以外の credentialed operator request
との挙動も分断する。header を固定列挙するだけで requested header を検証しない案より、上記は許可範囲を
明確に保ったまま unexpected header の CORS grant を防げる。

## 実装前に残る判断

なし。response header の正確な値、invalid preflight の response shape、UI/runtime を変更しない境界、remote
browser assertion の有効化 flag はこの plan で固定した。将来 runtime が別の non-safelisted request header を追加した
場合は、この allowlist を自動的に広げず、再現と必要性を記録した別 plan を作成する。

## Existing References

- `internal/platform/service/http.go:18-21`
  - staging と production の canonical origin allowlist。
- `internal/platform/service/http.go:159-196`
  - `/auth/session` を含む handler tree と CORS wrapper の適用点。
- `internal/platform/service/http.go:724-750`
  - OPTIONS short-circuit と現在の CORS response header 固定値。
- `internal/platform/service/http_test.go:372-467`
  - 許可 origin と unknown origin の CORS regression coverage。
- `internal/platform/service/auth.go:164-179`
  - 無効または absent cookie を `auth_mode: enabled`, `authenticated: false` の `200` として返す session contract。
- `operator-ui/src/lib/operatorApiClient.ts:69-94`
  - TypeSpec browser client、credentialed request policy、`/auth/session` adapter。
- `operator-ui/src/App.tsx:44-87`
  - unauthenticated session response を login route に送る UI behavior と transport error surface。
- `operator-ui/tests/operator-ui.ci.spec.js:7-43,319-337`
  - remote lane と browser-context HTTP request helper。
- `tools/dev/run-operator-ui-playwright.sh:77-84` と
  `.github/workflows/online-release-staging-verify.yml:129-139`
  - deployed Pages/backend を使う remote staging verification の entrypoint。
- `docs/specs/platform-product-auth.md:74-87`
  - split-origin credentialed fetch と allowlisted-origin CORS の browser session contract。

## Black-box Specification Changes

`docs/specs/platform-product-auth.md` を implementation の最初に更新し、split-origin browser session の CORS
contract を次の observable behavior に明確化する。

- staging と production の canonical Pages origin だけが credentialed cross-origin operator API request を許可される。
- allowed origin からの preflight は `GET`, `POST`, `OPTIONS` と `Content-Type`, `x-ms-useragent` だけを許可し、
  `Access-Control-Allow-Credentials: true` を維持する。
- unknown origin、または allowlist 外の requested header を含む preflight は CORS permission headers を受け取らない。
- unauthenticated `GET /auth/session` は CORS transport failure ではなく既存 TypeSpec session response を返し、
  frontend は login flow に進む。

TypeSpec の route/payload contract と cookie attribute contract は変更しない。

## Code Change Map

- `docs/specs/platform-product-auth.md` (MODIFY)
  - split-origin browser session の origin/header/credentials/unauthenticated behavior を black-box CORS contract として追記する。
- `internal/platform/service/http.go` (MODIFY)
  - fixed request-header allowlist と OPTIONS requested-header validation を導入し、valid preflight に
    `Content-Type, x-ms-useragent` を返す。exact origin allowlist、`Vary: Origin`、credentials、method set を維持する。
- `internal/platform/service/http_test.go` (MODIFY)
  - staging/prod preflight、credentials、unknown origin、unknown requested header、session JSON response、JSON/multipart
    preflight coverage を table-driven で追加または拡張する。
- `operator-ui/tests/operator-ui.ci.spec.js` (MODIFY)
  - `OPERATOR_UI_ASSERT_ANONYMOUS_SESSION_REDIRECT=1` の remote staging lane で configured auth backend に
    anonymous `/operator` を開き、ship した `OperatorApiClient` の `/auth/session` が `authenticated: false` を受け、
    login route へ遷移して `Session check failed` にならないことを browser で観測する。
- `.github/workflows/online-release-staging-verify.yml` (MODIFY)
  - remote lane に `OPERATOR_UI_ASSERT_ANONYMOUS_SESSION_REDIRECT=1` を渡し、上記 assertion を常に有効化する。

## Execution Steps

1. `docs/specs/platform-product-auth.md` に CORS contract を先に追加する。
   - canonical origin と credentialed cookie contract は変更せず、allowed request header と rejected preflight の
     observable behavior だけを固定する。
2. `internal/platform/service/http.go` で CORS policy を実装する。
   - existing exact-origin map を唯一の origin authority とする。
   - preflight request-header token を lowercase/trim して allowlist subset を判定する。
   - unknown origin または invalid requested header に CORS grant を返さず、valid request には fixed canonical header
     list・credentials・methods・origin vary を返す。
3. Go HTTP regression tests を追加する。
   - canonical staging origin の `GET /auth/session` preflight with `x-ms-useragent`。
   - production origin の JSON POST と multipart upload に必要な `content-type, x-ms-useragent` preflight。
   - both canonical origins の credentials/origin/method/header response contract。
   - unknown origin と known origin + unknown requested header に CORS allow headers がないこと。
   - auth-enabled/no-cookie `/auth/session` が `200` と `authenticated: false` を返すこと。
4. remote Playwright staging verification を更新する。
   - `OPERATOR_UI_ASSERT_ANONYMOUS_SESSION_REDIRECT=1` のときだけ、deployed Pages の `/operator` を anonymous
     browser context で開く test を追加する。
   - login heading/route を待ち、Auth Error と `Session check failed` が表示されないことを assert する。
   - runtime の user-agent policy と CORS preflight を実際に通すため、この test は native fetch の代替ではなく、
     ship した `OperatorApiClient` を使わなければならない。
5. focused Go tests、operator UI build、remote staging verification workflow を最新 implementation head で実行し、
   origin/header/credentials response evidence を PR に残す。

## Dependencies and Parallelism

- Step 1 は CORS behavior が product/service contract であるため Step 2 に先行しなければならない。
- Go policy implementation と Go tests は `http.go`/`http_test.go` に対する一つの直列変更とする。
- spec contract 固定後、remote Playwright assertion は Go test design と並行して準備できるが、実行は implementation
  deploy に対してだけ行う。
- staging verification には auth enabled backend と canonical Pages build-time API base URL が必要である。anonymous
  session assertion に human OAuth secret または authenticated account は不要である。

## Verification

- `go test ./internal/platform/service` で次を確認する。
  - `x-ms-useragent` を要求する staging `OPTIONS /auth/session` が canonical origin、credentials、methods、
    両方の allowed request header つきで成功する。
  - production の JSON POST と multipart-upload preflight が `content-type, x-ms-useragent` で成功する。
  - unknown origin、および認識しない requested header を含む known origin が CORS permission headers を受け取らない。
  - cookie を持たない auth-enabled session が `200 {auth_mode: enabled, authenticated: false}` のままである。
- 適用対象の repository Go quality gates と `operator-ui/` の `pnpm run build` を実行する。
- deployed staging implementation head に対し、次を含む OPTIONS request を送る。
  `Origin: https://staging.ai-arena.pages.dev`, `Access-Control-Request-Method: GET`, and
  `Access-Control-Request-Headers: x-ms-useragent`。exact CORS response headers を記録する。
- 同じ deployment head に対して remote staging Playwright lane を実行する。anonymous browser が `Failed to fetch`
  なしで login へ到達し、設定済み auth mode 向けの既存 operator-flow verification を維持しなければならない。
- release 前に Go regression test で production も同じ static policy を使うことを確認する。deployment 後、
  production canonical origin から header-only OPTIONS check を再実行してから production fixed を宣言する。

## Risks and Mitigations

- reflective な `Access-Control-Allow-Headers` implementation は cross-origin permissions を広げてしまう。
  - mitigation: fixed case-insensitive allowlist とし、wildcard と request-value reflection を使わない。
- session だけで runtime header を変更または削除すると、JSON POST/upload behavior が不整合になったり、runtime upgrade 後に
  再発したりする。
  - mitigation: generated client/pipeline を変更せず、既知の runtime header を server boundary で許可する。
- Go HTTP request だけを使う CORS test では browser preflight behavior を証明できない。
  - mitigation: focused Go contract tests を維持し、deployed Pages-to-Render anonymous browser assertion を追加する。
- preflight 修正中に cookie behavior を誤って弱める可能性がある。
  - mitigation: `Access-Control-Allow-Credentials: true` を明示的に assert し、auth cookie code と attributes を変更しない。
- stale frontend/backend bytes に対して staging flow を確認してしまう可能性がある。
  - mitigation: remote workflow evidence を最新 implementation PR head に結び、deployed frontend/backend URLs を記録する。
