# service-postgres-schema-management-and-query-layer
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0022-service-postgres-schema-management-and-query-layer.md`

## Objective

`0056` で導入した Postgres durable store を、startup-time inline DDL から
versioned schema management へ移し、query layer も review しやすく保守しやすい形へ揃える。
最初のゴールは、Atlas を schema diff / migration plan 生成の基盤として導入し、
SQL 形式の schema を source of truth に固定したうえで、service の Postgres query を
`sqlc + pgx` の責務分離へ進めることに置く。

## Context

- 現状の `internal/platform/service/store_postgres.go` は `postgresQueueStoreSchema` を process 起動時に `Exec` している
- `0056` は durable write model の landing を成立させたが、DDL の versioning、schema diff 生成、apply artifact 管理は未整備のまま残っている
- durable state は Phase 6 以降に queue/write path だけで終わらず、operator query、read model、replay/resume input まで広がるため、query 数は早い段階で増える見込みがある
- user decision として、schema source of truth は SQL 形式でよく、migration review / production apply workflow / quality gate は Atlas の責務としては期待しない
- issue `0022` の比較論点を踏まえ、schema diff plan 生成は Atlas、query authoring / generated typing は `sqlc`、runtime driver は `pgx` に分離する

## Scope

- Atlas による SQL schema source-of-truth と migration plan generation の導入方針を固定する
- startup inline DDL を廃止し、service 起動前提の schema apply contract へ切り替える
- `service_queue_records` 周辺 query を `sqlc + pgx` ベースの query layer へ移す
- `0057` / `0058` が再利用できる query package / generated code layout を先に整える
- Atlas 運用を 2 レーンで整理する
  - review/deploy lane: `migrate diff` で review 対象の migration SQL を生成する
  - test/bootstrap lane: `schema apply` で空 DB を desired schema へ揃え、e2e / integration test の初期化に使う

この plan では以下を扱わない。

- production rollout、migration review policy、approval gate の自動化
- Atlas Pro feature 前提の drift detection / migration lint
- public HTTP API
- replay / resume / audit read model 自体の実装

## Spec Changes

### `docs/specs/platform-service-persistence.md`

- schema source of truth を SQL 形式で保持し、Atlas が current DB schema と desired schema の差分から migration plan を生成する contract を追記する
- startup inline DDL を許可しないこと、schema apply は service 起動前の手順として扱うことを明記する
- migration artifact と query artifact の責務分離を追記する

### `docs/specs/platform-service-skeleton.md`

- Postgres backend の初期化責務から schema bootstrap を外し、runtime は既存 schema に依存するだけであることを明記する
- `sqlc + pgx` を使う query layer の境界と、service / worker から見える抽象境界を補足する

### `docs/development/platform-service-postgres.md`

- Atlas 導入後の contributor workflow として、review/deploy lane と test/bootstrap lane の使い分けを定義する
- local/CI の Postgres harness 上で `schema apply` を使う空 DB 初期化手順を追記する
- migration 生成は `migrate diff`、テスト用 bootstrap は `schema apply` という役割分担を明記する

### `AGENTS.md`

- Postgres schema/query stack の repo-local 開発運用は `docs/development/platform-service-postgres.md` を参照することを追記する
- schema review/deploy lane と test/bootstrap lane の両方がある前提で、この repo の AI agent が導線を見落とさないようにする

### 新規または更新 spec

- Postgres schema / migration / query generation の責務をまとめた spec を追加または更新する
- schema source、migration directory、generated query code の配置と更新手順を定義する

## Expected Code Changes

- Atlas config、desired schema SQL、migration directory の追加
- `store_postgres.go` から inline schema bootstrap を除去
- `sqlc` config と generated query package の追加
- `service_queue_records` write path query を `sqlc + pgx` 経由へ移行
- migration generation / sqlc generation / integration verification 用の make target または同等の entrypoint 追加
- `docs/development/platform-service-postgres.md` へ 2 レーン運用と test/bootstrap 手順を追加
- `AGENTS.md` へ開発運用文書の参照を追加

## Sub-tasks

- [ ] Atlas 用の desired schema source、migration directory、apply contract を spec に落とす
- [ ] `service_queue_records` の current schema を Atlas 管理へ移し、baseline migration を用意する
- [ ] `sqlc` config と package layout を決め、`pgx/v5` target で generated query surface を作る
- [ ] Postgres queue store の write path を generated query へ移す
- [ ] `docs/development/platform-service-postgres.md` に review/deploy lane と test/bootstrap lane の運用を書き、`AGENTS.md` からリンクする
- [ ] local/CI で schema apply と sqlc generation を再現できる verification path を追加する
- [ ] `0057` / `0058` が使う query extension point を明文化する

## Parallelism

- schema source / migration directory 設計と query package layout 設計は並行できる
- baseline migration 作成後は generated query 導入と store integration test を分担できる

## Dependencies

- depends on: `0056-platform-online-foundation-02-01-durable-store-and-write-model.md`
- informs: `0057-platform-online-foundation-02-02-result-read-model-and-operator-query.md`
- informs: `0058-platform-online-foundation-02-03-replay-resume-audit-inputs.md`

## Risks and Mitigations

- Atlas source schema と generated migration のどちらが正本か曖昧になる
  - mitigation: desired schema SQL を source of truth、migration file は review/apply artifact と明記する
- `sqlc` 導入を急ぎすぎて query package 境界が `0056` 既存 seam を壊す
  - mitigation: `QueueStore` interface は維持し、まず Postgres backend 内部だけを generated query に寄せる
- generated code が review noise になり、plan の意図より diff が見づらくなる
  - mitigation: handwritten SQL file と generated artifact の配置を分離し、review では SQL 差分を主に見られる形を保つ
- Atlas と sqlc の両方を一度に入れて setup コストが膨らむ
  - mitigation: migration/apply workflow の自動化は最小に留め、まず `service_queue_records` 1 table と少数 query で landing する

## Design Decisions

- schema source of truth は SQL 形式で保持する
- Atlas は current DB schema と desired schema の diff から migration plan を作る用途に限定して採用する
- query layer は `sqlc` が SQL から generated code を作り、runtime driver は `pgx/v5` とする
- migration review / production apply workflow / quality gate は Atlas へ委譲せず、repo/workflow 側で別途設計する
