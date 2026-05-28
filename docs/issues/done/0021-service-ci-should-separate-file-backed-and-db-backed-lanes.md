# service-ci-should-separate-file-backed-and-db-backed-lanes

## Summary

`0056` 時点の CI は `.github/workflows/go-ci.yml` の `go-test` job 1 本で
`make test` を実行し、その job 全体に `AI_ARENA_PG_TEST_DSN` を注入している。
このため、次の 2 系統が 1 つの lane に混在している。

- file-backed / in-memory queue を前提にした CLI-first lane
- Docker Postgres を使う durable queue / online service local reproduction lane

現在でも両 backend の test code 自体は回っているが、verification 意図が job から読めず、
片方だけ rerun したいときや、CLI scenario と durable backend scenario の所要時間を別管理したいときに扱いづらい。

## Current State

- CI file:
  `.github/workflows/go-ci.yml`
- current test step:
  `make GOPATH=/home/runner/go GOMODCACHE=/home/runner/go/pkg/mod GOCACHE=/home/runner/.cache/go-build test`
- current env:
  `AI_ARENA_PG_TEST_DSN=postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable`

observed coverage today:

- in-memory / file-backed queue path:
  `cmd/arena-service/main_test.go`,
  `internal/platform/service/integration_test.go`
- Postgres durable queue path:
  `internal/platform/service/store_postgres_test.go`

gap today:

- CI job 名の上で `CLI-first file-backed scenario` と
  `durable queue with Docker Postgres` が分離されていない
- `arena-service --postgres-dsn ...` を使う CLI/operator-facing lane が
  in-memory lane と独立した verification target としてはまだ明示されていない

## Proposed Solution

- `go-ci` の test lane を少なくとも 2 系統へ分離する
  - file-backed / in-memory queue lane
  - Docker Postgres を使う durable queue lane
- file-backed lane では `AI_ARENA_PG_TEST_DSN` を渡さず、
  既存 CLI-first / in-memory queue scenario を baseline として回す
- durable lane では Docker service container を立て、
  Postgres queue store と `--postgres-dsn` を使う local online-service reproduction を明示的に回す
- 必要なら `make test` と `make test-postgres` をそのまま job 分割に使い、
  後続で CLI/operator scenario 専用 target を追加する

## Priority

backend ごとの verification 目的が混ざったままだと、CI 失敗時の切り分けと rerun が遅くなる。
`0056` 以降、service line が in-memory lane と durable lane の両方を持つ前提になったので、
CI も同じ責務分離で見える化したほうが次の read-model / operator-flow 系 plan を進めやすい。
