# online-release-version-verification
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

staging の Render backend に、現在の deploy commit SHA を read-only に返す
`GET /version` endpoint を追加する。response は次の JSON object に限定する。

```json
{"version_sha":"<full commit SHA>"}
```

`online-release-staging.yml` は Render deploy hook を起動した後、backend の
`/version` が `target_sha` と一致するまで待機する。これにより、workflow summary
だけでなく、実際に staging traffic を処理している backend が想定 commit を返すことを
確認する。さらに `/healthz` が HTTP server と worker loop の両方を `OK` と返すまで待機し、
worker loop の起動前に HTTP server だけが応答する状態を deploy 成功とみなさない。

`online-release-staging-verify.yml` は次の read-only remote smoke surface を確認する。

- frontend に接続できる
- backend `GET /version` が検証対象の full SHA を返す
- backend `GET /healthz` が HTTP `200` かつ `{"api":"OK","worker":"OK"}` を返す
- backend `GET /auth/session` が `auth_mode=enabled`、`authenticated=false` を返す
- 匿名 browser の `/operator` が login route へ redirect する

完了境界は、staging deploy が version identity の一致後に HTTP server / worker loop の
readiness を待って成功し、後続の remote verification が同じ SHA と匿名認証境界を確認する
ことである。staging 用 machine account、
OIDC provider、`OPERATOR_UI_TEST_AUTH`、game/AI/bot ZIP upload、registration、match、
ranking の remote mutation 検証はこの計画に含めない。

## Context and Current References

- `.github/workflows/online-release-staging.yml:271-280`
  - Render deploy hook に `ref=${TARGET_SHA}` を付けて起動するが、deploy が traffic に反映されたことを待たずに workflow が完了している。
- `.github/workflows/online-release-staging-verify.yml:97-163`
  - `workflow_run` の `head_sha` または dispatch input を検証対象として受け取る。
  - 現在は remote URL を指定して既存の managed operator UI flow を実行する。
- `internal/platform/service/http.go:164-201,396-405`
  - `/healthz` と `/auth/session` は認証 middleware の外側にあり、`/api/v1/` は auth-enabled service では operator role を要求する。
- `internal/platform/service/worker_loop.go:11-77`
  - worker loop は worker guard を取得した後、queue recovery と `ProcessNext` を繰り返すが、現在 readiness state を公開していない。
- `cmd/arena-service/main.go:369-401,509-550`
  - Render runtime の auth、`OperatorAPI`、worker loop が構築される。build-time version と worker readiness を API adapter へ渡す配線を追加する。
- `cmd/operator-ui-fixture/main.go:85-105`
  - fixture backend は worker loop を起動しないため、local fixture lane では deterministic な ready state を明示的に注入する必要がある。
- `typespec/namespaces/operator/health.tsp:12-14`
  - 現在の operational endpoint の TypeSpec source。
- `typespec/namespaces/shared.tsp:106-114`
  - `HealthResponse` などの shared response model。現在は `status` だけを持つ。
- `typespec/main.tsp:5-9`
  - TypeSpec namespace import の entrypoint。
- `operator-ui/playwright.config.js:3-55`
  - `remote` scenario は web server を起動せず、指定された staging frontend URL へ接続する。
- `operator-ui/tests/operator-ui.ci.spec.js:7-118,399-432`
  - remote の anonymous redirect assertion、service-backed protected flow、backend URL を使う request helper の実装。
- `docs/specs/platform-service-operator-ui.md`
  - operator browser surface と auth-enabled regression lane の observable contract。
- `docs/development/platform-service-online-deploy.md:454-565`
  - staging deploy / verification の現在の運用契約。protected mutation を remote smoke として記述している箇所は今回の境界に合わせて更新する。
- `docs/exec-plan/todo/0092-operator-ui-auth-playwright-local-oidc-provider.md`
  - local/CI 専用の auth regression seam を扱う既存 plan。staging machine account の実装を今回の依存にはしない。

`/version` の version source は Render 固有の `RENDER_GIT_COMMIT` には依存させず、build 時に
checkout の `git rev-parse --verify HEAD` で取得した full SHA とする。Render の deploy hook
には `ref=${TARGET_SHA}` を指定するため、Render build 時の checkout `HEAD` と polling の
比較対象は同じ commit identity になる。

## Adopted Design

### Version endpoint

- endpoint は public read-only の `GET /version` とする。
- auth middleware の外側に登録し、匿名 request でも取得できるようにする。
- `version_sha` は `cmd/arena-service` の package variable `main.Version` を `serve` 起動時に service adapter へ渡した値を返す。
- `make render-build` は `BUILD_VERSION_SHA ?= $(shell git rev-parse --verify HEAD 2>/dev/null)` を解決し、空なら build を失敗させたうえで、`-ldflags "-X main.Version=$(BUILD_VERSION_SHA)"` を使って binary に full SHA を埋め込む。
- 既存の `VERSION` は PostgreSQL migration baseline 用の意味を持つため、git SHA のデフォルト値をそこへ設定しない。build identity には専用の `BUILD_VERSION_SHA` を使う。
- linker flag なしで起動した local fixture/test binary では `version_sha` が空でも response shape は維持する。staging acceptance では空文字を成功とみなさず、target SHA との完全一致を要求する。
- `version_sha` に短縮 SHA、branch 名、build 時刻、hostname、追加 metadata を含めない。

TypeSpec を wire contract の正本とし、version 用 namespace/operation と response model を
追加して OpenAPI と generated operator client を再生成する。operator UI 本体に version
表示を追加することは今回の範囲外である。

### Health readiness

- `GET /healthz` は public endpoint のままとし、response body を次の JSON object に変更する。
  - ready: `{"api":"OK","worker":"OK"}`、HTTP `200`
  - worker loop 未起動または初回 queue recovery 前: `{"api":"OK","worker":"NOT_READY"}`、HTTP `503`
- request が handler まで到達して JSON を返せる時点で `api` は `OK` とする。HTTP server
  自体が listen していなければ response は返らないため、別の api readiness flag は設けない。
- `WorkerLoop` に race-safe な `Ready()` state を追加し、worker guard の取得と初回の
  `RecoverExpired` 成功後に ready とする。`Run` の終了時は not ready に戻す。
- `serve` は同じ `WorkerLoop` の readiness callback を `OperatorAPI` に渡す。local fixture
  は実 worker loop を持たないため、fixture が static backend として ready であることを
  明示的に adapter へ渡す。
- `/healthz` は auth middleware の外側に置き、auth configured staging でも匿名で確認できる
  ようにする。HTTP `503`、`worker != "OK"`、malformed body はすべて not ready として扱う。

### Render deploy readiness

既存の Render deploy hook 呼び出し直後に、repo-owned helper を使った polling step を追加する。

- endpoint: `${STAGING_BACKEND_URL}/version`
- expected value: `needs.prepare.outputs.target_sha`
- interval: 15 seconds
- maximum attempts: 40（最大 10 分）
- one-request timeout: 15 seconds
- HTTP error、connection failure、malformed JSON、empty `version_sha`、SHA mismatch は retry 対象。
- 最大試行回数を超えた場合は step を fail し、最後に観測した HTTP status と version value を log/summary に残す。
- secret、session cookie、Access token は polling request に付けない。

version polling が成功した後、同じ backend の `${STAGING_BACKEND_URL}/healthz` を同じ bounded
policy で polling する。

- ready condition: HTTP `200`、JSON object の `api == "OK"`、`worker == "OK"`
- HTTP `503`、connection failure、malformed JSON、いずれかの component の non-`OK` は retry 対象。
- timeout 時は最後に観測した HTTP status、`api`、`worker` を log/summary に残す。
- version が一致しても health が ready にならなければ staging deploy workflow は失敗する。

この wait step を staging deploy workflow の成功条件にする。`workflow_run` による verify は
その後に起動するため、automatic path では version-ready であることを引き継ぐ。dispatch による
verification path では、verify job 自身が `/version` を同じ target SHA と比較し、別 commit の
staging に対して成功しないようにする。

### Remote smoke boundary

`verify:remote` は protected operator mutation を実行しない remote-specific test surface に
整理する。

- current service-backed test の remote 実行を skip または remote smoke と分離し、bundle path、operator nav、mutation API を要求しない。
- remote version test は backend `/version` の JSON と `OPERATOR_UI_EXPECT_VERSION_SHA` を完全一致で確認する。
- anonymous session test は既存の frontend redirect を維持する。
- backend `/auth/session` は remote scenario で直接取得し、`auth_mode=enabled` かつ `authenticated=false` を確認する。
- local fixture、real-local、CI auth-mock lane の protected operator flow と fixture ZIP は変更しない。
- remote workflow の未使用 `preset_id` input/output/env は削除し、remote lane の契約を smoke surface に合わせる。

## Change Map

- `typespec/namespaces/operator/version.tsp` (NEW)
  - `GET /version` operation を定義する。
- `typespec/namespaces/shared.tsp` (MODIFY)
  - `VersionResponse` の JSON model を追加する。
- `typespec/main.tsp` (MODIFY)
  - version namespace を import する。
- `typespec/generated/openapi/operator/openapi.json` (MODIFY)
  - TypeSpec build で `/version` contract を再生成する。
- `operator-ui/src/generated/operator-api/` (MODIFY)
  - generated client/model/serializer に version operation を反映する。
- `cmd/arena-service/main.go` (MODIFY)
  - linker flag の注入先となる `main.Version` を定義し、`serve` から service adapter へ渡す。
- `Makefile` (MODIFY)
  - `render-build` で `git rev-parse --verify HEAD` を `BUILD_VERSION_SHA` として解決し、空値を拒否して `-X main.Version=...` を build に渡す。
- `internal/platform/service/http.go` (MODIFY)
  - public `/version` route と build-time version response handler を追加する。
  - `/healthz` を api/worker の JSON response と readiness に応じた `200` / `503` に変更する。
- `internal/platform/service/worker_loop.go` (MODIFY)
  - worker loop の race-safe readiness state と `Ready()` accessor を追加し、初回 queue recovery と終了境界を反映する。
- `internal/platform/service/http_test.go` (MODIFY)
  - adapter に full SHA を設定した `/version` response、status、content type、JSON shape、auth configured 下でも public であることを検証する。
  - worker readiness 前の `503` / `NOT_READY` と ready 後の `200` / `OK` response を検証する。
- `cmd/operator-ui-fixture/main.go` (MODIFY)
  - worker loop を持たない fixture の `/healthz` readiness を明示的に `OK` とする。
- `tools/dev/wait-for-remote-version.sh` (NEW)
  - Render backend の version identity を bounded polling する repo-owned helper を追加する。
- `.github/workflows/online-release-staging.yml` (MODIFY)
  - Render deploy hook 後に version wait、続けて health wait を実行し、SHA mismatch または worker not ready を deploy failure とする。
- `.github/workflows/online-release-staging-verify.yml` (MODIFY)
  - target SHA を remote smoke test へ渡し、`/version` 確認後に `/healthz` の HTTP status と api/worker body を検証する。
  - protected operator flow、ZIP env、未使用 preset input/output/env を remote path から除外する。
- `operator-ui/tests/operator-ui.ci.spec.js` (MODIFY)
  - remote version/session/health smoke test を追加し、version 一致後に `api=OK` / `worker=OK` を検証する。protected service-backed test は remote では実行しない。
- `operator-ui/tests/operator-ui.spec.js` (MODIFY)
  - fixture local lane が新しい health response shape と `200` readiness を確認する。
- `docs/specs/index.md` (MODIFY)
  - `typespec/namespaces/operator/version.tsp` を operator route lookup に追加する。
- `docs/specs/platform-service-operator-ui.md` (MODIFY)
  - remote staging verification の read-only/auth boundary と version identity assertion を追記する。
- `docs/development/platform-service-online-deploy.md` (MODIFY)
  - Render deploy の version-ready 後の health-ready completion condition、bounded wait、remote smoke checklist、summary evidence を更新する。
- `docs/development/operator-ui-local-verification.md` (MODIFY)
  - local fixture / managed backend の `/healthz` readiness contract を更新する。
- `tools/dev/wait-for-remote-health.sh` (NEW)
  - backend の `api=OK` / `worker=OK` と HTTP `200` を bounded polling する repo-owned helper を追加する。
- `DELETE: N/A`

## Black-box Contract Changes

### `GET /version`

- status: `200`
- content type: `application/json`
- response: exactly one `version_sha` string property
- staging: `version_sha` is the full commit SHA currently serving the Render backend
- auth: no session or role required
- failure behavior: endpoint itself does not fabricate a SHA; staging verification rejects empty or mismatched values

### Staging deploy completion

The staging deploy workflow is not complete when the Render hook accepts the request. It is complete
only after `/version.version_sha` equals the canonical full `target_sha` within the bounded polling
window and the subsequent `/healthz` response is HTTP `200` with both `api` and `worker` equal to
`OK`. A timeout, mismatch, HTTP `503`, or non-OK component prevents the downstream staging
verification from being treated as successful.

### Remote verification

The automatic staging lane is read-only with respect to operator data. It validates deployed commit
identity and public/anonymous boundaries only. It does not create game registrations, upload bundles,
create bots, enqueue matches, update rankings, or consume staging data.

## Sub-tasks and Dependencies

1. Update the behavioral spec and online deploy runbook with the new public version identity and remote smoke boundary.
2. Add the TypeSpec route/model and regenerate OpenAPI/generated client outputs.
3. Add the build-time SHA variable/linker flag, pass it into the service adapter, and implement the public handler with focused tests using a deterministic full SHA.
4. Add worker loop readiness state and health response/status tests, including explicit fixture readiness.
5. Add the bounded version and health polling helpers and unit/script-level validation for exact match, retry, and timeout behavior where practical.
6. Update both staging workflows. Keep version wait then health wait before workflow completion and keep verify-side exact comparisons as defense in depth.
7. Update the Playwright remote scenario and fixture lane while retaining protected coverage in local/CI auth-enabled lanes.
8. Run repo quality gates and perform one staging release verification against the latest head.

Steps 2-3 and 4-6 can proceed in parallel after step 1 is agreed; generated contract updates must land
before implementation code depends on them. Workflow lint and staging acceptance depend on all changes being
present.

## Verification

### Local and repository gates

- `pnpm --dir typespec build` completes and leaves no generated OpenAPI/client drift.
- focused Go tests cover `/version` with a full SHA, ensure both routes remain public when auth is configured, and cover health readiness before/after worker loop startup.
- remote Playwright tests cover frontend connectivity, exact `/version`, `/healthz` HTTP `200` plus `api=OK` / `worker=OK`, anonymous `/auth/session`, and `/operator` redirect.
- local/CI fixture and auth-enabled lanes continue to cover their existing protected operator surface and ZIP fixture flow.
- applicable `make test`, `make lint`, workflow linter, textlint, and `git diff --check` pass.

### Staging acceptance

- `online-release-staging` triggers Render with the canonical target SHA.
- the wait step retries while the old service, 404, 5xx, transport error, or malformed response is observed.
- the wait step succeeds only after the backend returns the exact full target SHA.
- `online-release-staging-verify` then succeeds for the same SHA and records frontend URL, backend URL,
  target SHA, observed version SHA, and smoke-test artifact locations in its summary.
- no staging operator mutation or authentication credential is introduced by this plan.

## Non-goals and Rejection Conditions

- Do not add a staging machine account, OIDC provider, OAuth test double, service token, or access cookie.
- Do not set `OPERATOR_UI_TEST_AUTH` in the staging workflow.
- Do not reintroduce game/AI/bot ZIP upload or registration into the automatic staging lane.
- Do not make `/version` an authenticated operator endpoint; its purpose is deployment provenance/readiness.
- Do not accept a branch name, short SHA, stale SHA, empty string, or merely successful deploy-hook response as proof of deployment.
- Do not change production release behavior in this plan except for shared API generation if required.
