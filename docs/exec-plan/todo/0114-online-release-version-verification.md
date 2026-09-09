# online-release-version-verification
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

staging と production の Render backend が、実際に traffic を処理している build の full
commit SHA を read-only な `GET /version` で返せるようにする。staging deploy workflow は
Render deploy hook を起動しただけでは成功とせず、backend が target SHA を返すまで待機する。

response は次の JSON object に限定する。

```json
{"version_sha":"<full commit SHA>"}
```

`online-release-staging-verify.yml` の remote smoke は、frontend への接続、backend の exact
version、匿名の `/auth/session`、匿名 `/operator` の login redirect だけを確認する。remote
lane は operator mutation、fixture ZIP、machine account、OIDC、`OPERATOR_UI_TEST_AUTH` を
要求しない。

完了境界は、staging deploy が target SHA と完全一致する backend version identity を記録し、
同じ SHA に対する anonymous remote smoke が成功することである。worker ownership/readiness の
JSON 表現と staging の worker-ready 判定は `0116-online-release-worker-readiness-verification`
が、production の post-release verification と rollback は
`0117-online-release-production-readiness-rollback` が扱う。

## Context and Current References

- `.github/workflows/online-release-staging.yml:271-294`
  - Render deploy hook は `ref=${TARGET_SHA}` を指定するが、現在は hook の受付直後に workflow を成功させる。
- `.github/workflows/online-release-staging-verify.yml:1-24,97-163`
  - workflow_run の head SHA または dispatch input を検証対象として受け取り、remote URL を Playwright に渡す。
- `.github/workflows/online-release-production.yml:127-154`
  - production も deploy hook で target SHA を起動する。version endpoint は shared binary に入るため、production でも利用可能になる。
- `cmd/arena-service/main.go:369-401,509-550`
  - Render runtime と service adapter の構築箇所。build-time version を adapter へ渡す配線を追加する。
- `internal/platform/service/http.go:164-201,396-405`
  - `/healthz` と `/auth/session` は public route である。`/version` も同じ auth middleware 外に追加する。
- `typespec/namespaces/operator/health.tsp:12-14`
  - operational endpoint の TypeSpec source。version route は別 namespace とする。
- `typespec/namespaces/shared.tsp:106-114`
  - shared response model の配置。
- `operator-ui/playwright.config.js:3-55` と `operator-ui/tests/operator-ui.ci.spec.js:7-118,399-432`
  - remote scenario、anonymous redirect、backend request helper の既存 seam。
- `docs/development/platform-service-online-deploy.md:454-565`
  - staging / production deploy と remote verification の runbook 正本。

Render deploy hook は full SHA を `ref` として受け取れる。`ref` 指定は Render の auto deploy
設定にも影響するため、既存の staging / production の明示 deploy hook 運用を維持し、その
issuance path や secret value を workflow summary に出してはならない。

## Adopted Design

### Public version identity

- `GET /version` は auth middleware の外にある public read-only endpoint とする。
- response は `application/json`、`version_sha` だけを持つ object、HTTP `200` とする。
- `version_sha` は `cmd/arena-service` の package variable `main.Version` を `serve` 起動時に
  service adapter へ渡した値である。
- `make render-build` は `BUILD_VERSION_SHA ?= $(shell git rev-parse --verify HEAD 2>/dev/null)` を
  解決する。空値なら build を失敗させ、`-ldflags "-X main.Version=$(BUILD_VERSION_SHA)"` により
  full SHA を binary へ埋め込む。
- migration baseline 用の既存 `VERSION` に SHA の意味を混在させない。
- linker flag なしの local fixture/test binary は空文字でも response shape を保つ。staging/prod
  acceptance は空文字、短縮 SHA、branch 名、build 時刻、hostname を成功としてはならない。

TypeSpec を wire contract の正本とし、version namespace、response model、OpenAPI、generated
operator client を再生成する。operator UI に version 表示を追加しない。

### Staging version convergence

deploy hook の直後に repo-owned helper で `${STAGING_BACKEND_URL}/version` を polling する。

- expected value: `needs.prepare.outputs.target_sha`
- interval: 15 seconds
- maximum attempts: 80（最大 20 分）
- one-request timeout: 15 seconds
- HTTP/transport error、malformed JSON、empty value、SHA mismatch は retry 対象
- timeout は staging release acceptance の failure とし、最後に観測した HTTP status と value を
  workflow summary / log に残す
- cookie、access token、session secret は polling request に付けない

20 分はこの repository の release acceptance window であり、Render build の provider-side
terminal status を API なしに直接取得するものではない。timeout 後に provider state を成功と
推測してはならず、Render deploy record を operator が確認する。

### Remote smoke boundary

`verify:remote` は次の read-only surface に限定する。

- frontend へ接続できる
- backend `/version` が `OPERATOR_UI_EXPECT_VERSION_SHA` と完全一致する
- backend `/auth/session` が `auth_mode=enabled` と `authenticated=false` を返す
- anonymous browser の `/operator` が login route へ redirect する

protected operator API、bundle path、preset input/output/env、ZIP upload、registration、bot creation、
match enqueue、ranking を remote path から除外する。これらは local / CI auth-mock lane の責務に
残す。`/healthz` の worker component はこの plan の success condition に含めない。

## Code and Documentation Change Map

- `(NEW) typespec/namespaces/operator/version.tsp`
  - public `GET /version` operation を定義する。
- `(MODIFY) typespec/namespaces/shared.tsp`
  - `VersionResponse` JSON model を追加する。
- `(MODIFY) typespec/main.tsp`
  - version namespace を import する。
- `(MODIFY) typespec/generated/openapi/operator/openapi.json`
  - TypeSpec build output を再生成する。
- `(MODIFY) operator-ui/src/generated/operator-api/`
  - generated client/model/serializer に version operation を反映する。
- `(MODIFY) cmd/arena-service/main.go`
  - linker flag 注入先の `main.Version` を定義し、service adapter へ渡す。
- `(MODIFY) Makefile`
  - `render-build` に full SHA の build identity を追加し、空値を拒否する。
- `(MODIFY) internal/platform/service/http.go`
  - public `/version` route と response handler を追加する。既存 `/healthz` contract は変更しない。
- `(MODIFY) internal/platform/service/http_test.go`
  - full SHA、content type、JSON shape、auth configured 下でも public であることを確認する。
- `(NEW) tools/dev/wait-for-remote-version.sh`
  - exact SHA、retry、timeout、last observation を扱う bounded helper を追加する。
- `(MODIFY) .github/workflows/online-release-staging.yml`
  - hook 後に version convergence を待ち、20 分 timeout を deployment failure とする。
- `(MODIFY) .github/workflows/online-release-staging-verify.yml`
  - target SHA を remote smoke へ渡し、protected remote flow と unused preset contract を除去する。
- `(MODIFY) operator-ui/tests/operator-ui.ci.spec.js`
  - remote version/session/anonymous redirect smoke を確認する。
- `(MODIFY) docs/specs/index.md`
  - version TypeSpec source を route lookup に追加する。
- `(MODIFY) docs/specs/platform-service-operator-ui.md`
  - remote read-only/auth boundary と backend version identity を記録する。
- `(MODIFY) docs/development/platform-service-online-deploy.md`
  - version convergence、20 分 acceptance window、summary evidence、provider-state investigation を記録する。
- `(DELETE) N/A`

## Black-Box Specification Changes

### `GET /version`

- status: `200`
- content type: `application/json`
- response: exactly one `version_sha` string property
- auth: session、role、operator credential を要求しない
- staging/prod release verification: full target SHA との完全一致だけを success とする

### Staging deploy completion

staging deploy は Render hook が受理されたことだけでは成功としない。20 分以内に serving backend の
`version_sha` が canonical target SHA と一致したときだけ version convergence を成功とする。
timeout、mismatch、HTTP failure、malformed response は workflow failure であり、provider state を
成功とみなさない。

## Subtasks and Dependencies

1. behavioral spec と online deploy runbook に public version identity と remote smoke boundary を記録する。
2. TypeSpec route/model を追加し、OpenAPI/generated client を再生成する。
3. build-time SHA と public handler を実装し、focused HTTP test を追加する。
4. version polling helper と script-level validation を追加する。
5. staging workflow と remote Playwright scenario を version-only boundary へ更新する。
6. quality gate を通し、latest head の staging version acceptance を 1 回実行する。

この plan は worker lock retry に依存しない。`0116-online-release-worker-readiness-verification` は
この plan と `0115-worker-lock-retry-on-render-rollout` の実装後に着手する。

## Verification

- `pnpm --dir typespec build` が成功し、generated OpenAPI/client に drift がない。
- focused Go test が full SHA、content type、JSON shape、auth configured 下の public access を確認する。
- remote Playwright が frontend connection、exact version、anonymous session、`/operator` redirect を確認する。
- local/CI auth-enabled lane は既存の protected operator surface と ZIP fixture flow を継続して確認する。
- applicable な `make test`、`make lint`、workflow linter、textlint、`git diff --check` が成功する。
- staging は hook 後に target full SHA を観測し、timeout 時は last observation と Render deploy record を確認する。

## Non-goals and Rejection Conditions

- worker readiness、worker lock retry、`/healthz` response body の component state を追加しない。
- staging machine account、OIDC provider、OAuth test double、service token、access cookie を追加しない。
- `OPERATOR_UI_TEST_AUTH`、game/AI/bot ZIP upload、registration、match、ranking を remote lane へ導入しない。
- production の post-release health verification / rollback をこの plan で実装しない。
- frontend の deployed commit identity をこの plan で証明しない。
