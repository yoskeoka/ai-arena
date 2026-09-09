# worker-lock-retry-on-render-rollout
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

Incident reference: https://github.com/yoskeoka/ai-arena/pull/323

## Objective

Render の zero-downtime deploy で旧 instance と新 instance が同じ Neon Postgres を
一時的に共有しても、新 instance が worker lock の競合だけを理由に起動失敗しないようにする。
新 instance は HTTP liveness を提供したまま、旧 worker が advisory lock を解放するまで
queue worker の起動を bounded に待ち、lock 取得後にだけ queue の claim / 実行を開始する。

完了境界は次のとおり。

- lock 競合は識別可能な sentinel error として扱い、設定された有限時間まで retry する
- DB connection failure や advisory-lock query failure は lock 競合と混同せず、即時に失敗させる
- lock 未取得中は `RecoverExpired`、`Claim`、`ProcessNext` を呼び出さない
- context cancellation は graceful shutdown として扱い、retry を停止して lock を保持しない
- 有限時間内に旧 worker が終了しない場合は fail closed し、2 worker の同時 queue 実行を許さない
- `/healthz` の HTTP liveness と worker ownership を分離したままにする
- 現行の single-worker / single logical queue authority と、lease expiry による run recovery の責務を維持する

この plan では version endpoint、Render 外部への log forwarding、Sentry 等の observability
provider、multi-worker 化、worker を別 Render service へ分離することは扱わない。

## Incident and Current Behavior

今回の staging incident は、PR #323 の merge commit を対象にした Render deploy で、旧
instance が session-level PostgreSQL advisory lock を保持したまま新 instance が起動し、
新 worker が一度だけ `pg_try_advisory_lock` を試みて終了したものだった。

現在の実装は次の構造になっている。

- `internal/platform/service/store_postgres.go:23-54`
  - `PostgresQueueStore.AcquireWorker` は専用の pool connection を保持し、
    `pg_try_advisory_lock(hashtext('ai-arena-service-single-worker'))` で queue ownership を取る
  - lock を取れなかった場合は connection を解放して、通常の formatted error を返す
  - lock を取れた場合は、その connection を保持した release function を返す
- `internal/platform/service/worker_loop.go:20-78`
  - `WorkerLoop.Run` は開始時に `AcquireWorker` を一度だけ呼ぶ
  - 失敗すると `RecoverExpired` や queue processing より前に return する
- `cmd/arena-service/main.go:535-584`
  - worker loop と HTTP server は別 goroutine で起動する
  - worker loop が error を返すと HTTP server を shutdown して `serve` も error で終了する
- `internal/platform/service/http.go:166-167,396-398`
  - `/healthz` は `{"status":"ok"}` を HTTP 200 で返す liveness endpoint であり、worker lock の状態を判定しない
- `internal/platform/service/store_postgres_test.go:138-153`
  - 既存テストは同じ queue に対する二つ目の worker ownership を拒否する契約を確認している

Render の HTTP health check は configured path への GET の `2xx` / `3xx` status を成功とし、
response body は判定対象ではない。[Render Health Checks](https://render.com/docs/health-checks)
したがって、新 worker が lock 待ち中でも `/healthz` を 200 にして Render が旧 instance の
停止へ進めることが、今回の handoff の前提になる。

## Design Contract

### Worker ownership

Postgres advisory lock と専用 connection を現行どおり queue authority の fencing として使う。
lock を保持している process だけが `RecoverExpired`、claim、match execution を実行できる。
lock 待ち中の process は HTTP request を受けられても、queue の mutation / execution を開始してはならない。

lease expiry は process crash 後の queue record recovery 用であり、旧 process の advisory lock
handoff を待たずに強制 takeover する仕組みとして再利用しない。

### Error classification

`AcquireWorker` が `pg_try_advisory_lock` の `false` を受けた場合だけ、`errors.Is` で判定できる
worker ownership conflict sentinel (`ErrWorkerQueueOwned`) を返す。

- lock が別 worker に保持されている: retry 対象
- pool connection の acquire failure: retry 対象外
- advisory-lock query / scan failure: retry 対象外
- empty worker ID 等の入力エラー: retry 対象外

sentinel の名称と配置は既存 error 定義との整合を保ち、呼び出し側が error message の文字列比較に
依存しないようにする。

### Bounded retry

`WorkerLoop.Run` の開始時に worker ownership の取得を retry する。retry helper は context、
retry interval、maximum wait を受け取れる形にして、時間依存の unit test を短い duration で
実行できるようにする。

- 通常起動の retry interval は既存の worker poll interval と同程度の低頻度とする
- maximum wait は有限の既定値を持たせ、Render の rolling deploy handoff を許容しつつ、
  旧 worker が停止しない場合に無期限で process を残さない
- sentinel 以外の error は最初の試行で返す
- maximum wait に到達した場合は ownership timeout と分かる error を返し、`serve` を
  fail closed させる
- context cancellation は timeout / infrastructure failure と区別し、retry を中断して正常終了する

実装では retry ごとに新しい `AcquireWorker` を呼び、lock 未取得時に返された専用 connection が
確実に release されることを保証する。lock 取得後は既存の defer release を維持する。

### Liveness and handoff sequence

`serve` の HTTP / worker goroutine 構成は維持し、次の sequence を成立させる。

1. 新 instance が HTTP port を listen する
2. `/healthz` が 200 を返す。worker ownership はまだ pending でもよい
3. Render が新 instance を healthy として扱い、旧 instance の shutdown を開始する
4. 旧 instance の DB session が advisory lock を解放する
5. 新 instance の retry が lock を取得する
6. 新 instance が初めて `RecoverExpired` / `Claim` / match execution を行う

`/healthz` を worker-ready endpoint に変更して、この sequence を循環待ちにしてはならない。
worker readiness を将来観測する必要がある場合は別の contract として扱い、この plan の
Render health check path には組み込まない。

## Black-Box Specification Changes

### `(MODIFY) docs/specs/platform-service-single-worker-assumptions.md`

Phase 7 の「生存 worker を観測した場合は fail closed」という記述を、queue execution の
exclusive ownership という不変条件と、deploy handoff 中の bounded wait を両立する形に更新する。
少なくとも次を明記する。

- lock 待ち中の新 process は worker として queue mutation を実行しない
- lock 競合は bounded handoff wait の対象である
- timeout、DB failure、context cancellation の結果を区別する
- timeout 後は fail closed し、lease expiry を advisory lock の代用にしない
- HTTP liveness は worker ownership と別契約である

### `(MODIFY) docs/development/platform-service-online-deploy.md`

Phase 7 staging recovery / deploy runbook に、Render rolling deploy 時の worker handoff と確認項目を
追記する。新 revision の `/healthz` が先に 200 になっても、worker が lock を取得するまでは
queue が実行されないこと、lock timeout なら deploy failure として旧 known-good revision を
維持することを operator が確認できるようにする。

## Code Change Map

- `(MODIFY) internal/platform/service/errors.go`
  - worker ownership conflict (`ErrWorkerQueueOwned`) と ownership wait timeout の sentinel を定義する
- `(MODIFY) internal/platform/service/store_postgres.go`
  - `pg_try_advisory_lock` の `false` を `ErrWorkerQueueOwned` で返す
  - DB connection / query errors は既存の error wrapping を保つ
- `(MODIFY) internal/platform/service/worker_loop.go`
  - ownership acquisition retry helper を追加する
  - lock 取得前に既存 polling / recovery loop へ進まない
  - bounded timeout と context cancellation を扱う
- `(NEW) internal/platform/service/worker_loop_test.go`
  - fake worker guard を使い、lock conflict の retry、lock 取得前の queue 非実行、
    timeout、context cancellation、non-conflict error の即時終了を検証する
- `(MODIFY) internal/platform/service/store_postgres_test.go`
  - 二つ目の worker が ownership sentinel を返すことを `errors.Is` で検証する
  - 先行 worker の release 後に後続 worker が ownership を取得できることを検証する
  - 専用 connection による advisory lock lifecycle を壊していないことを確認する
- `(MODIFY) docs/specs/platform-service-single-worker-assumptions.md`
  - 上記の bounded handoff 契約を反映する
- `(MODIFY) docs/development/platform-service-online-deploy.md`
  - Render staging rollout の operator-facing handoff / failure contract を反映する
- `(DELETE) N/A`

`cmd/arena-service/main.go` と `internal/platform/service/http.go` は、既存の concurrent startup
と `/healthz` liveness が契約を満たすため、原則として変更しない。実装上変更が必要になった
場合は、HTTP liveness と worker ownership の分離を壊さない差分に限定する。

## Dependencies and Parallelism

1. spec update
   - `platform-service-single-worker-assumptions.md` と online deploy runbook の black-box contract を先に更新する
2. error / store seam
   - sentinel を定義し、Postgres store の false-return path を分類する
3. worker retry
   - helper と `WorkerLoop.Run` を接続する。lock 未取得中の queue 非実行を維持する
4. tests
   - unit test と Postgres integration test を追加し、既存の second-worker rejection test を維持・強化する
5. verification
   - Go tests / lint と workflow checks を通してから、別途 Render staging deploy で handoff を確認する

1 と 2 は black-box 契約を確定した後なら並行に検討できるが、実装の merge 順は spec first とする。
3 は 2 に依存し、4 は 3 の retry semantics に依存する。Render 上の実機確認は全テスト完了後の
独立した acceptance とする。

## Verification

実装時には少なくとも次を確認する。

- `go test ./internal/platform/service/...`
- Postgres test lane で、同じ DB に対する first worker / second worker の拒否、release 後の再取得、
  queue record の単一実行を確認する
- `make lint`
- `./tools/workflow-lint.sh --mode=pre-push`
- `git diff --check`
- Render staging で新 revision の `/healthz` が 200 になった後、旧 revision 停止後にだけ
  worker が lock を取得して queue を処理することを deploy log / operator flow で確認する
- 旧 process が lock を解放しない試験では maximum wait 後に deploy が fail closed し、
  既存の known-good revision を維持することを確認する

## Alternatives and Non-Goals

- Render の zero-downtime を無効化するために persistent disk を導入する案は、provider-specific
  な運用変更と追加コストを伴うため採用しない。[Render Disks](https://render.com/docs/disks)
- Render の複数 deploy の overlapping policy を変更する案は、旧/new instance の同時稼働に
  よる DB advisory lock 競合を解消しないため採用しない
- blocking `pg_advisory_lock` へ単純に置き換える案は、context / timeout / error classification
  をアプリケーション側で扱いにくくするため採用しない
- queue lease の expiry を待たずに advisory lock を takeover する案は、二重 worker 実行の
  リスクがあるため採用しない
- multi-worker scheduling、fencing token、別 worker service への分離は後続の architectural plan とする
