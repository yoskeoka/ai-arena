# Platform service postgres harness

`0056` では、online service skeleton の durable write model backend を Postgres で検証する。
production target は `Neon Postgres` である。
local contributor workflow と CI では Docker ベースの Postgres を使う。

## Local Verification

起動:

```sh
docker compose -f tools/dev/postgres-compose.yml up -d postgres
```

停止:

```sh
docker compose -f tools/dev/postgres-compose.yml down -v
```

便利 target:

```sh
make postgres-up
make test-postgres
make postgres-down
```

`make test-postgres` は `AI_ARENA_PG_TEST_DSN` が未指定なら、
次の local default DSN で Postgres integration test を有効化する。

```text
postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable
```

CI や別 port の local harness では、環境変数または `make` 変数 override で
同じ target を再利用してよい。

```sh
AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable make test-postgres
make AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable test-postgres
```

CLI で durable queue backend を使うときは、
`--postgres-dsn` または `ARENA_SERVICE_POSTGRES_DSN` を指定する。

## CI Harness

GitHub Actions の `go-ci` durable Postgres lane は Docker service container の Postgres を起動する。
その job だけに `AI_ARENA_PG_TEST_DSN` を渡し、`make test-postgres` を実行する。
file-backed default lane の `make test` job には Postgres DSN を注入しない。

artifact lane は引き続き local filesystem を使う。
この harness が追加しているのは、service write model の durable metadata backend だけである。
