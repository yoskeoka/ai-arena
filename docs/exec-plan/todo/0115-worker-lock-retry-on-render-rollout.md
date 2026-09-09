# worker-lock-retry-on-render-rollout
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

Incident reference: https://github.com/yoskeoka/ai-arena/pull/323

## Objective

Render の zero-downtime deploy で旧 instance と新 instance が同じ Neon Postgres を一時的に
共有しても、新 instance が worker lock の競合だけを理由に起動失敗しないようにする。新 instance
は HTTP liveness を提供したまま、旧 worker が advisory lock を解放するまで bounded に待ち、lock
取得後にだけ queue worker を開始する。

この shared `serve` path の変更は staging と production の両方に適用される。staging は rollout
handoff の acceptance 環境とし、production の deployed SHA / worker-ready verification と
automatic rollback は `0117-online-release-production-readiness-rollback` で別途扱う。

完了境界は次のとおり。

- `pg_try_advisory_lock` の ownership conflict だけを識別可能な sentinel error として retry する
- retry は 10 秒ごとに行い、最大 5 分で timeout する。lock 解放を 5 分間待ち続けるのではなく、各試行で ownership を再確認する
- DB connection/query failure と input error は retry せず、最初の試行で返す
- lock 未取得中は `RecoverExpired`、`Claim`、`ProcessNext` を呼び出さない
- context cancellation は graceful shutdown として retry を止め、lock を保持しない
- timeout は queue execution を fail closed し、2 worker の同時実行を許さない
- `/healthz` は HTTP `200` を返す liveness endpoint のままとし、Render は response body を判定しない
- lease expiry は crash recovery の責務に留め、advisory lock の強制 takeover に使わない

worker readiness の JSON body と release workflow によるその観測はこの plan の範囲外である。

## Incident and Current References

- `internal/platform/service/store_postgres.go:23-54`
  - `PostgresQueueStore.AcquireWorker` は dedicated pool connection 上の session-level advisory lock を取得する。
- `internal/platform/service/worker_loop.go:11-77`
  - `WorkerLoop.Run` は worker guard を一度だけ取得し、失敗時は recovery/queue processing 前に return する。
- `cmd/arena-service/main.go:535-584`
  - worker loop と HTTP server は別 goroutine で起動し、worker loop の起動 error は service shutdown につながる。
- `internal/platform/service/http.go:166-167,396-398`
  - `/healthz` は public HTTP `200` liveness route である。
- `internal/platform/service/store_postgres_test.go:138-153`
  - 現在は同じ queue に対する二つ目の worker ownership を拒否することを確認している。
- `.github/workflows/online-release-staging.yml:271-294`
  - deploy hook は current release workflow の起点である。
- `docs/specs/platform-service-single-worker-assumptions.md:35-58`
  - single logical queue authority、lease recovery、shutdown の既存契約を定義する。

Render は新 instance を healthy として traffic を切り替えた 60 秒後に旧 instance へ `SIGTERM` を
送る。既定の graceful shutdown delay は 30 秒である。したがって default topology の通常 handoff
は約 90 秒以内に lock を解放する見込みであり、5 分は provider jitter と異常観測に余裕を持たせる
safety bound とする。

## Adopted Design

### Worker ownership and error classification

Postgres advisory lock と dedicated connection を queue authority の fencing として維持する。lock を
持つ process だけが `RecoverExpired`、claim、match execution を実行できる。lock 待ち中の process
は HTTP request を受けても queue mutation / execution を開始してはならない。

`AcquireWorker` が `pg_try_advisory_lock` の `false` を受けた場合だけ、`errors.Is` で判定できる
`ErrWorkerQueueOwned` を返す。

- another worker owns the lock: retry 対象
- pool connection acquire failure: retry 対象外
- advisory-lock query/scan failure: retry 対象外
- empty worker ID などの input error: retry 対象外

sentinel の名称と配置は既存 error 定義に合わせ、呼び出し側が error message の文字列比較に依存
しないようにする。

### Bounded retry and shutdown-delay invariant

`WorkerLoop.Run` の開始時に ownership acquisition を retry する。helper は context、retry interval、
maximum wait を受け取り、unit test が短い duration を注入できるようにする。

- normal interval は 10 秒とする
- normal maximum wait は 5 分とする
- each retry は new `AcquireWorker` call を使い、lock 未取得時の dedicated connection が確実に release されることを確認する
- sentinel 以外の error は first attempt で返す
- maximum wait 到達時は ownership timeout sentinel/error を返し、queue execution を fail closed する
- context cancellation は timeout/infrastructure failure と区別し、正常終了する

`internal/platform/service/worker_loop.go` で maximum wait の既定値を定義する箇所には、次の趣旨を
English code comment として置く。

> Keep this at least 60 seconds plus Render's configured maxShutdownDelaySeconds plus a 30-second buffer. If the Render setting changes from its 30-second default, update this value and the release readiness timeout together.

この comment は、Render の `maxShutdownDelaySeconds` を 300 秒まで延ばす場合に、worker maximum
wait と `0116` の readiness wait も同時に見直す ownership を明示する。5 分は current 30 秒設定を
下回らず、上記の default lower bound 120 秒を十分に超える。

### Liveness and handoff sequence

`serve` の concurrent HTTP / worker startup は維持し、次の sequence を成立させる。

1. new instance が HTTP port を listen する
2. `/healthz` が HTTP `200` を返す。worker ownership は pending でもよい
3. Render が new instance を healthy として traffic を切り替える
4. 60 秒後に Render が old instance へ `SIGTERM` を送る
5. old DB session が advisory lock を release する
6. new instance の 10 秒ごとの retry が lock を取得する
7. new instance が初めて `RecoverExpired` / `Claim` / match execution を行う

`/healthz` の HTTP status を worker ownership に応じて non-`200` に変更してはならない。後続の
`0116-online-release-worker-readiness-verification` は同じ `200` response の JSON body を GitHub
Actions が読む readiness signal として拡張してよいが、Render health check の liveness 判定には使わない。

## Black-Box Specification Changes

### `(MODIFY) docs/specs/platform-service-single-worker-assumptions.md`

次を明記する。

- lock 待ち中の new process は queue mutation を行わない
- ownership conflict は 10 秒間隔・最大 5 分の bounded handoff wait の対象である
- timeout、DB failure、context cancellation の結果を区別する
- timeout 後は fail closed し、lease expiry を advisory lock の代用にしない
- HTTP liveness は worker ownership と別契約であり、staging / production の共通 runtime に適用する

### `(MODIFY) docs/development/platform-service-online-deploy.md`

Render staging / production rollout の handoff と確認項目を追記する。new revision の `/healthz` が
HTTP `200` になっても、lock を取得するまで queue を実行しないこと、ownership timeout は queue
safety failure として扱うこと、production release workflow の external verification/rollback は
`0117` が扱うことを記録する。

## Code Change Map

- `(MODIFY) internal/platform/service/errors.go`
  - `ErrWorkerQueueOwned` と ownership wait timeout の sentinel を定義する。
- `(MODIFY) internal/platform/service/store_postgres.go`
  - advisory-lock `false` を `ErrWorkerQueueOwned` として返し、DB errors の wrapping は維持する。
- `(MODIFY) internal/platform/service/worker_loop.go`
  - 10 秒 interval / 5 分 maximum wait の retry helper を追加する。
  - maximum wait の constant/configuration value に Render shutdown-delay invariant の English comment を置く。
  - lock 取得前に existing polling/recovery loop へ進まないようにする。
- `(NEW) internal/platform/service/worker_loop_test.go`
  - retry、lock 取得前の queue 非実行、5 分既定値を短い injected duration で検証する timeout、cancellation、non-conflict fast failure を確認する。
- `(MODIFY) internal/platform/service/store_postgres_test.go`
  - second worker が ownership sentinel を返すこと、release 後に後続 worker が取得できること、connection lifecycle を検証する。
- `(MODIFY) docs/specs/platform-service-single-worker-assumptions.md`
  - bounded handoff と staging/production scope を反映する。
- `(MODIFY) docs/development/platform-service-online-deploy.md`
  - Render rollout handoff と ownership timeout の operator contract を反映する。
- `(DELETE) N/A`

`cmd/arena-service/main.go` と `internal/platform/service/http.go` はこの plan では変更しない。HTTP
liveness に additional JSON readiness metadata を載せる場合は `0116` の責務とする。

## Subtasks and Dependencies

1. single-worker spec と online deploy runbook に shared staging/production handoff contract を先に記録する。
2. sentinel と Postgres store の false-return seam を追加する。
3. retry helper と `WorkerLoop.Run` を接続し、10 秒 / 5 分の default と shutdown-delay comment を実装する。
4. unit test と Postgres integration test を追加し、exclusive ownership を確認する。
5. quality gates の後、Render staging で old/new handoff を確認する。

この plan は `0114` に依存しない。`0116` はこの plan と `0114` の実装後に、release workflow に
worker-readiness verification を追加する。

## Verification

- `go test ./internal/platform/service/...`
- Postgres test lane で first/second worker rejection、release 後の retry acquisition、queue record の単一実行を確認する。
- `make lint`
- `./tools/workflow-lint.sh --mode=pre-push`
- `git diff --check`
- Render staging で new revision が HTTP `200` liveness を返した後、old revision の shutdown と lock release 後にだけ worker が queue を処理することを log / operator evidence で確認する。
- old process が lock を解放しない test では 5 分後に queue execution が fail closed し、二重実行がないことを確認する。provider-side release/rollback 判定は `0117` の workflow acceptance と混同しない。

## Alternatives and Non-goals

- blocking `pg_advisory_lock` に置換しない。context、timeout、error classification を application 側で保持するためである。
- lease expiry で advisory lock を takeover しない。二重 worker 実行を防ぐためである。
- multi-worker scheduling、fencing token、separate worker service は後続の architectural plan とする。
- `/healthz` を non-`200` worker readiness endpoint にしない。
- production backend の version/readiness verification、automatic rollback、frontend identity verification は実装しない。
