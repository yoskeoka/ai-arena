# Platform Service Postgres Stack 仕様

## 目的

このドキュメントは、online service skeleton が durable Postgres backend を使うときの
schema source、migration artifact、query generation artifact の責務分離を定義する。

runtime の queue/store contract 自体は `docs/specs/platform-service-persistence.md` と
`docs/specs/platform-service-skeleton.md` が正本であり、
この spec は Postgres 向け contributor-facing stack boundary だけを補足する。

## この spec の責務範囲

この spec が定義するもの:

- desired schema SQL の正本位置と役割
- migration SQL artifact の役割
- query SQL と generated code の役割分担
- local/CI の schema apply / query generation 入口

この spec が定義しないもの:

- production rollout や approval gate
- Atlas Pro feature 前提の drift detection / lint policy
- operator/read API 自体の contract

## Artifact Roles

Postgres stack は 4 種類の artifact を分けて扱う。

- desired schema SQL:
  現在ある table / index / constraint の正本
- migration SQL:
  desired schema と current DB schema の差分から生成する review/apply artifact
- query SQL:
  runtime が必要とする query の正本
- generated query code:
  query SQL から生成する typed adapter artifact

source of truth は desired schema SQL と query SQL である。
migration SQL と generated query code は生成物であり、
人間が直接 contract を定義する primary artifact ではない。

## Layout Contract

Postgres stack の repo layout は少なくとも次を分離する。

- schema source:
  `internal/platform/service/postgres/schema/`
- migration artifact:
  `internal/platform/service/postgres/migrations/`
- query source:
  `internal/platform/service/postgres/query.sql`
- generated query code:
  `internal/platform/service/postgres/sqlc/`

`0057` / `0058` 以降で read-side query が増えても、
同じ schema source と generated package を拡張してよい。
queue write path 専用の runtime seam を変えるために、別の schema source を増やしてはならない。

## Schema Apply Contract

- service runtime は startup 時に DDL を実行してはならない
- durable Postgres lane を使う contributor / CI / deploy job は、
  service 起動前に schema apply を済ませなければならない
- 空 DB bootstrap では desired schema を直接 apply してよい
- migration review が必要な変更では、desired schema 変更後に migration SQL を生成し、
  review/apply artifact として repo に残す

## Query Layer Contract

- runtime driver は `pgx/v5` を使う
- generated query code は `pgx` connection/transaction を受け取る薄い adapter とする
- service / worker から見える durable queue 抽象は既存のまま維持する
- generated query package は query authoring を代替しない。新しい read/write path を追加するときは、
  まず query SQL を更新し、その後 generated code を更新する

## Verification Entry Contract

local と CI は、同じ repo entrypoint から次を再現できなければならない。

- desired schema を target DB へ apply する
- desired schema 変更から migration SQL artifact を生成する
- query SQL から generated query code を再生成する

個別 contributor が Atlas や `sqlc` の内部 command を覚える必要はない。
repo-local の command surface から同じ操作に到達できることを優先する。
