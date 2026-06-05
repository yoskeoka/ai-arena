# Platform service postgres harness

online service skeleton の durable write model backend は Postgres を使って検証する。
production target は `Neon Postgres` である。
local contributor workflow と CI では Docker ベースの Postgres を使う。

schema/query stack の contract は `docs/specs/platform-service-postgres-stack.md` が正本であり、
この文書は repo-local contributor workflow を定義する。

## Artifact Layout

- desired schema SQL:
  `internal/platform/service/postgres/schema/`
- migration SQL:
  `internal/platform/service/postgres/migrations/`
- query SQL:
  `internal/platform/service/postgres/query.sql`
- generated query code:
  `internal/platform/service/postgres/sqlc/`

desired schema SQL と query SQL が人間の編集対象である。
migration SQL と generated query code は review/apply/generation artifact として扱う。

## Command Surface

```sh
make postgres-up
make postgres-schema-apply
make postgres-migrate-diff NAME=<migration_name>
make postgres-sqlc-generate
make test-postgres
make postgres-down
```

`make postgres-schema-apply` と `make test-postgres` は `AI_ARENA_PG_TEST_DSN` が未指定なら、
次の local default DSN を使ってよい。

```text
postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable
```

`make postgres-migrate-diff` は migration review/deploy lane の入口であり、
desired schema 変更後に versioned migration SQL を生成する。
`NAME` は migration filename suffix に使う short slug を要求する。

`make postgres-schema-apply` は test/bootstrap lane の入口であり、
空 DB を desired schema へ直接揃える。
service runtime 自体は schema bootstrap を行わない。

## Local Verification

起動:

```sh
docker compose -f tools/dev/postgres-compose.yml up -d postgres
```

停止:

```sh
docker compose -f tools/dev/postgres-compose.yml down -v
```

CI や別 port の local harness では、環境変数または `make` 変数 override で
同じ target 群を再利用してよい。

```sh
AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable make postgres-schema-apply
make AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable postgres-schema-apply
AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable make test-postgres
make AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable test-postgres
```

### Review/Deploy Lane

1. desired schema SQL を更新する
2. `make postgres-migrate-diff NAME=<migration_name>` で migration SQL を生成する
3. generated migration SQL を review/apply artifact として commit する

この lane では migration SQL を人間が review する。
desired schema SQL が正本であり、migration SQL はそこから導いた artifact である。

### Test/Bootstrap Lane

1. `make postgres-up`
2. `make postgres-schema-apply`
3. `make postgres-sqlc-generate` が必要なら実行する
4. `make test-postgres`

空 DB の初期化では migration replay を必須にしない。
test/bootstrap lane は desired schema への直接 apply を優先してよい。

CLI で durable queue backend を使うときは、
`--postgres-dsn` または `ARENA_SERVICE_POSTGRES_DSN` を指定する。
この lane でも DB 側 schema apply は起動前に完了していなければならない。

## CI Harness

GitHub Actions の `go-ci` durable Postgres lane は Docker service container の Postgres を起動する。
その job だけに `AI_ARENA_PG_TEST_DSN` を渡し、
少なくとも `make postgres-schema-apply` と `make test-postgres` を実行する。
file-backed default lane の `make test` job には Postgres DSN を注入しない。

artifact lane は引き続き local filesystem を使う。
この harness が追加しているのは、service write model の durable metadata backend だけである。

deploy-shaped artifact verification を併用するときは、
`docs/development/platform-service-object-storage.md` の `SeaweedFS` harness を組み合わせる。
