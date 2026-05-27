# Platform Service Postgres Harness

`0056` では、online service skeleton の durable write model backend を Postgres で検証する。
production target は `Neon Postgres` だが、local contributor workflow と CI では Docker ベースの Postgres を使う。

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

`make test-postgres` は次の DSN で Postgres integration test を有効化する。

```text
postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable
```

CLI で durable queue backend を使う場合は `--postgres-dsn` または `ARENA_SERVICE_POSTGRES_DSN` を指定する。

## CI Harness

GitHub Actions の `go-ci` test lane は Docker service container の Postgres を起動し、
`AI_ARENA_PG_TEST_DSN` を渡して同じ integration test を実行する。

artifact lane は引き続き local filesystem を使うため、この harness が追加しているのは
service write model の durable metadata backend だけである。
