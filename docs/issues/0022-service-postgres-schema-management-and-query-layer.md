# service-postgres-schema-management-and-query-layer

## Summary

`0056` で導入した Postgres durable queue は、
`internal/platform/service/store_postgres.go` の
`postgresQueueStoreSchema` を process 起動時に `Exec` して初期化している。

この方式でも single-table の初期投入はできるが、schema versioning と
DDL の変更履歴、環境差分の drift 検知、rollback / forward-only 方針がまだ定義されていない。
また query 層も `pgx` の手書き SQL だけで始まっており、
DDL 管理と query authoring の責務分離をどうするか未決のままになっている。

## Current State

- schema bootstrap:
  `internal/platform/service/store_postgres.go`
  の `postgresQueueStoreSchema`
- initialization path:
  `NewPostgresQueueStore()` が `pool.Exec(ctx, postgresQueueStoreSchema)` を実行
- current query layer:
  `github.com/jackc/pgx/v5` / `pgxpool` を直接利用
- missing today:
  - versioned migration file の管理
  - local / CI / 将来の remote env で共有する schema apply 手順
  - drift detection と migration review の運用
  - typed query generation や repository boundary の是非に関する判断

## Proposed Solution

- Postgres schema 管理を startup-time inline DDL から分離し、
  versioned migration を持つ仕組みに置き換える
- 候補として、少なくとも次を比較して意思決定する
  - Atlas などの migration / DDL 管理ツール
  - `sqlc` のような typed query generation
  - `pgx` 手書き SQL を維持する場合の最小運用ルール
- migration tool と query layer は別々に決めず、
  次の観点を 1 つの設計判断として整理する
  - schema diff / apply / review を誰が担うか
  - local Docker / CI / production-target env で同じ migration artifact を使えるか
  - SQL を source of truth にするか、schema DSL を source of truth にするか
  - current `QueueStore` seam と将来の read-model / operator-flow 拡張に無理がないか

## Priority

今は table が 1 つなので startup DDL でも破綻していないが、
online service の durable state が増える前に migration と query layer の基準を固めたほうがよい。
ad hoc な `CREATE TABLE IF NOT EXISTS` を増やしてから移行すると、
CI / local / remote env の差分吸収と review ルール整備のコストが上がる。
