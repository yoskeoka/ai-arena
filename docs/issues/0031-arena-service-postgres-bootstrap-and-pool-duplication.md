# arena-service の Postgres bootstrap と pool ownership が分かりにくい

## Summary

`cmd/arena-service/main.go` の起動 wiring では、
queue 用 `PostgresQueueStore` と auth 用 `PostgresAuthStore` が
同じ DSN から別々の `pgxpool.Pool` を開いている。

また close path は最終的に `cliApp.closeFn` に集約されているが、
`newQueueStore` / `newAuthService` がそれぞれ `func()` を返し、
`newCLIApp` 側で都度 error path に手動で並べているため、
resource ownership が呼び出し元に漏れて見えやすい。

## Why this matters

- 現状の実装は動作上は閉じているが、
  `authCloseFn` / `closeFn` がどこで最終解放されるかを追いづらい
- 同一 process で queue と auth が同じ Postgres DSN を使うときに、
  pool を二重に持つ理由がコードから明確ではない
- 将来 auth / queue 以外の Postgres-backed component が増えると、
  同じ pattern の pool 重複と close path の手動合成が増えやすい

## Current behavior

- `runWithFactory` は `defer app.close()` を張る
- `newCLIApp` は `authCloseFn` と queue 側 `closeFn` を
  `cliApp.closeFn` に合成して返す
- そのため steady state の close 漏れは起きていない
- ただし `newQueueStore` と `newAuthService` は
  それぞれ独立に `pgxpool.New(...)` を呼ぶ

## Follow-up direction

- `arena-service` の Postgres dependency を
  service 単位の `func()` close 返却ではなく、
  process-owned bootstrap に寄せられないか検討する
- queue/auth が同じ DSN を使う場合は、
  共有 `pgxpool.Pool` もしくは共有 DB bootstrap object へ寄せる案を比較する
- file-backed / auth-disabled lane を壊さずに、
  Postgres-backed component だけを opt-in で束ねられる構造にする
- 少なくとも current code では、
  pool を分ける理由が intentional ならコメントか設計 note で明示する
