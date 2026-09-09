# online-release-worker-readiness-verification
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

Render の HTTP liveness を壊さずに、staging release workflow が worker ownership と initial queue
recovery の完了を観測できるようにする。public `GET /healthz` は worker pending 中も HTTP `200` を
維持するが、JSON body の component state で GitHub Actions が readiness を判定する。

response contract は次のとおり。

- ready: `{"api":"OK","worker":"OK"}`、HTTP `200`
- worker guard 未取得、または initial `RecoverExpired` 未成功: `{"api":"OK","worker":"NOT_READY"}`、HTTP `200`

staging deploy は `0114` が確認する exact `/version` の後、worker が `OK` になるまで 10 秒間隔で
最大 7 分 polling する。timeout、malformed body、HTTP failure、component の non-`OK` は release
workflow failure とする。Render health check は HTTP status だけを見るため、worker pending を
non-`200` にして old instance の shutdown を循環待ちにしてはならない。

完了境界は、lock handoff 中の service が HTTP liveness を保ちつつ queue mutation を開始せず、
worker ownership と first recovery の後だけ staging release/remote smoke が ready を成功と扱う
ことである。production の同じ post-release verification と automatic rollback は `0117` の責務である。

## 前提条件と現行参照

この plan は次の実装済み contract を前提とする。

- `0114-online-release-version-verification`
  - public `/version`、target SHA の staging convergence、remote anonymous smoke。
- `0115-worker-lock-retry-on-render-rollout`
  - `ErrWorkerQueueOwned`、10 秒 interval、5 分 ownership wait、lock 未取得中の queue 非実行。

current references:

- `internal/platform/service/worker_loop.go:11-77`
  - guard acquisition、initial `RecoverExpired`、polling loop の lifecycle。
- `internal/platform/service/http.go:164-201,396-405`
  - public `/healthz` route と JSON response handler。
- `cmd/arena-service/main.go:509-584`
  - same `WorkerLoop` と HTTP adapter の construction / shutdown boundary。
- `cmd/operator-ui-fixture/main.go:85-130`
  - fixture backend は real worker loop を持たない。
- `typespec/namespaces/operator/health.tsp:12-14` と `typespec/namespaces/shared.tsp:106-114`
  - health wire contract の正本。
- `.github/workflows/online-release-staging.yml:271-294`
  - 現在は Render deploy hook と summary だけを持つ。`0114` の実装が exact version convergence を追加した後、その直後に readiness helper を接続する。
- `.github/workflows/online-release-staging-verify.yml:97-163`
  - workflow_run / dispatch remote verification path。
- `operator-ui/tests/operator-ui.ci.spec.js:7-118,399-432`
  - remote backend request helper と anonymous browser assertion。

## 採用する設計

### readiness state

- `WorkerLoop` は race-safe な readiness state と `Ready()` accessor を持つ。
- state は worker guard 取得後、first `RecoverExpired` が成功した時点で true になる。
- `Run` の終了、context cancellation、ownership timeout、guard/recovery failure では false に戻る。
- `Ready()` が false の間、worker は `RecoverExpired`、claim、match execution を実行していないか、
  initial recovery を正常完了していないことを示す。
- `serve` は running `WorkerLoop` の callback を `OperatorAPI` に渡す。
- fixture backend は worker を起動しないため、static fixture service が ready である callback を明示的に渡す。

### `/healthz` body と liveness

- `/healthz` は auth middleware 外の public route として残す。
- handler に到達して JSON を返せる限り HTTP `200` を返す。`api` はこの状態で `OK` とする。
- `worker` は above readiness state に従って `OK` または `NOT_READY` を返す。
- Render health check は HTTP `200` だけを liveness success として扱う。body の `NOT_READY` は
  GitHub Actions の release readiness failure/retry 条件だけであり、Render instance recycle の条件にしない。
- TypeSpec、OpenAPI、generated client を新 body へ再生成する。既存 `status`-only model を参照する
  repository consumer は migration audit の上で更新する。

### staging readiness polling

`0114` の successful exact version polling の直後、repo-owned helper で
`${STAGING_BACKEND_URL}/healthz` を確認する。

- interval: 10 seconds
- maximum attempts: 42（最大 7 分）
- one-request timeout: 15 seconds
- ready: HTTP `200`、valid JSON、`api == "OK"`、`worker == "OK"`
- retry: transport/HTTP error、malformed JSON、`api`/`worker` non-`OK`
- timeout: last observed HTTP status、api、worker を workflow summary に残して release workflow を fail する

7 分は `0115` の 5 分 maximum ownership wait と retry boundary を上回る acceptance window である。
`0115` の Render shutdown-delay invariant が変わる場合、worker maximum wait とこの window を一緒に
更新する。

remote Playwright も exact version の後に同じ health body を確認する。local fixture / CI auth-mock
lane の protected operator flow は維持し、remote lane は read-only のままとする。

## 変更対象

- `(MODIFY) internal/platform/service/worker_loop.go`
  - race-safe readiness state、`Ready()`、guard/recovery/exit lifecycle を追加する。
- `(MODIFY) internal/platform/service/worker_loop_test.go`
  - ownership pending、first recovery success、error/cancellation/exit 後の state transition を検証する。
- `(MODIFY) internal/platform/service/http.go`
  - readiness callback を adapter に受け取り、public `/healthz` の api/worker JSON body を返す。
- `(MODIFY) internal/platform/service/http_test.go`
  - pending/ready の HTTP `200`、content type、JSON shape、auth configured 下の public access を確認する。
- `(MODIFY) cmd/arena-service/main.go`
  - same `WorkerLoop` readiness callback を service adapter へ渡す。
- `(MODIFY) cmd/operator-ui-fixture/main.go`
  - static fixture backend の readiness callback を true として渡す。
- `(MODIFY) typespec/namespaces/shared.tsp`
  - `HealthResponse` を api/worker component contract へ更新する。
- `(MODIFY) typespec/generated/openapi/operator/openapi.json`
  - TypeSpec build output を再生成する。
- `(MODIFY) operator-ui/src/generated/operator-api/`
  - generated health model/client を反映する。
- `(NEW) tools/dev/wait-for-remote-health.sh`
  - api/worker exact readiness、10 秒 retry、7 分 timeout、last observation を扱う helper を追加する。
- `(MODIFY) .github/workflows/online-release-staging.yml`
  - exact version convergence 後に health readiness helper を実行する。
- `(MODIFY) .github/workflows/online-release-staging-verify.yml`
  - dispatch / workflow_run path で version と health components を確認する。
- `(MODIFY) operator-ui/tests/operator-ui.ci.spec.js`
  - remote version 後の health body readiness を確認する。
- `(MODIFY) operator-ui/tests/operator-ui.spec.js`
  - fixture local lane の pending/ready response shape を確認する。
- `(MODIFY) docs/specs/platform-service-operator-ui.md`
  - public liveness、component readiness、remote smoke boundary を記録する。
- `(MODIFY) docs/development/operator-ui-local-verification.md`
  - fixture / managed backend の health response contract を更新する。
- `(MODIFY) docs/development/platform-service-online-deploy.md`
  - staging version-then-readiness sequence、7 分 timeout、summary evidence を記録する。
- `(DELETE) N/A`

## black-box contract の変更

### `GET /healthz`

- auth: no session or role required
- liveness status: handler が応答可能な限り HTTP `200`
- ready body: `{"api":"OK","worker":"OK"}`
- pending body: `{"api":"OK","worker":"NOT_READY"}`
- Render: HTTP `200` だけを health-check success として扱う
- GitHub Actions: `api` と `worker` の両方が `OK` のときだけ release readiness success とする

### staging release の完了条件

staging workflow は `/version.version_sha` が target SHA と一致した後、7 分以内に `/healthz` の
api/worker components がともに `OK` になったときだけ成功する。worker pending、timeout、malformed
body、request failure は release workflow failure であり、queue execution を ready と推測してはならない。

## サブタスクと依存関係

1. `0114` と `0115` の implementation PR が latest `main` に到達していることを確認する。
2. behavioral specs と runbook に liveness/readiness の責務分離を先に記録する。
3. worker readiness state と adapter callback、fixture seam を実装する。
4. TypeSpec/OpenAPI/generated client と focused Go/browser tests を更新する。
5. health polling helper、staging workflow、remote smoke を version-then-readiness sequence へ更新する。
6. quality gates 後、staging deploy で old/new worker handoff を acceptance する。

Steps 3 and 4 は black-box contract 合意後に並行できる。Workflow / remote verification は both に依存する。
`0117-online-release-production-readiness-rollback` はこの plan の implementation 後に着手する。

## 検証

- `pnpm --dir typespec build`
- `go test ./internal/platform/service/...`
- local fixture と auth-enabled CI lane で HTTP `200`、pending/ready body、protected flows を確認する。
- remote Playwright で frontend connection、exact `/version`、health `api=OK` / `worker=OK`、anonymous session、`/operator` redirect を確認する。
- helper の script-level test で retry、malformed JSON、timeout、last observation を確認する。
- `make test`、`make lint`、workflow linter、textlint、`git diff --check`
- staging deploy で new process が `NOT_READY` を返し得る間も Render health check は通過し、old lock release 後だけ `worker=OK` となることを確認する。

## 非目標と拒否条件

- worker lock retry algorithm、retry interval、maximum ownership wait は変更しない。
- `/healthz` を worker pending 時に HTTP non-`200` にしない。
- production deploy の post-release verification、automatic rollback、frontend commit identity は実装しない。
- staging machine account、OIDC、operator mutation、ZIP fixture flow を remote lane に導入しない。
