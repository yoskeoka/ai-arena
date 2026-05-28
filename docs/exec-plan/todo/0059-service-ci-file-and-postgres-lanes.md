# service-ci-file-and-postgres-lanes
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0056` で service skeleton が file-backed default lane と Postgres durable backend lane の
両方を持つ前提になったので、GitHub Actions 上の Go CI でもその 2 系統を job レベルで
分離し、rerun と failure triage を lane 単位で行えるようにする。

Addresses: `docs/issues/0021-service-ci-should-separate-file-backed-and-db-backed-lanes.md`

## Context

- 現状の `.github/workflows/go-ci.yml` は `go-test` job 1 本で `make test` を実行し、その job 全体に
  `AI_ARENA_PG_TEST_DSN` を注入している
- `Makefile` には既に file-backed / in-memory baseline 用の `make test` と、
  Docker Postgres harness 用の `make test-postgres` が分かれて存在する
- `docs/specs/platform-service-skeleton.md` は local CLI / CI lane では file-backed first を default としつつ、
  durable backend 検証では production と同じ Postgres contract を満たす local DB を使ってよいとしている
- `docs/development/platform-service-postgres.md` は Postgres harness を説明しているが、
  CI 上で file-backed lane と Postgres lane が別 job で見える contract までは固定していない

## Scope

- Go CI の test lane を file-backed lane と Postgres lane に分離する
- file-backed lane では `AI_ARENA_PG_TEST_DSN` を渡さず、CLI-first / in-memory baseline を維持する
- Postgres lane では Docker service container を使い、`make test-postgres` または同等 target で
  durable queue verification を明示する
- spec / development docs を更新し、CI 上の lane 責務分離を明文化する

この plan では以下を扱わない。

- Postgres schema migration 管理の改善
- operator query / read model の追加
- object storage lane や remote artifact backend の CI 追加
- 新しい persistence backend の導入

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- local CLI / CI の verification lane について、file-backed default lane と durable Postgres lane を
  別々の verification target として扱うことを明記する

### `docs/development/go-quality-gates.md`

- Go CI が `make test` と Postgres-backed lane の両方を別 job として実行することを追記する

### `docs/development/platform-service-postgres.md`

- CI Harness を更新し、Postgres lane が専用 job として `make test-postgres` を実行することを明記する

## Expected Code Changes

- `.github/workflows/go-ci.yml`
  - file-backed lane と Postgres lane の job 分離
  - Postgres service container と DSN 注入を durable lane のみに限定
- 必要なら `Makefile`
  - CI から durable lane を明示的に呼べる target 名の調整

## Sub-tasks

- [ ] 現在の `make test` / `make test-postgres` の責務を spec / development docs に揃える
- [ ] `.github/workflows/go-ci.yml` の test lane を file-backed lane と Postgres lane に分離する
- [ ] Postgres service container と `AI_ARENA_PG_TEST_DSN` を durable lane 専用に閉じ込める
- [ ] lane 名から rerun / triage 意図が読めることを確認する
- [ ] local verification と CI verification の対応関係を docs に反映する

## Parallelism

- spec / development doc 更新と workflow job 名のドラフトは並行できる
- final verification は workflow 変更と docs sync の両方がそろってから行う

## Dependencies

- informed by: `0056-platform-online-foundation-02-01-durable-store-and-write-model.md`
- informed by: `go-ci-make-targets.md`
- related issue: `docs/issues/0021-service-ci-should-separate-file-backed-and-db-backed-lanes.md`

## Risks and Mitigations

- `make test` に Postgres 前提 test が紛れ込むと file-backed lane が見かけ倒しになる
  - mitigation: file-backed lane は DSN なしで実行し、Postgres test は env guard で skip されることを確認する
- workflow 上だけ lane を分けても docs が追随しないと contributor が rerun target を誤認する
  - mitigation: `docs/development/go-quality-gates.md` と `docs/development/platform-service-postgres.md` を同じ PR で更新する
- job 分離で cache key や runtime がぶれると CI 所要時間が読みにくくなる
  - mitigation: 既存 cache strategy は維持し、test command の入口だけを分ける

## Design Decisions

- CI lane 分離は既存 `make test` / `make test-postgres` surface をそのまま使う最小変更を優先する
- file-backed lane は service skeleton の default verification target として残す
- durable lane は Postgres contract の verification に限定し、artifact backend までは広げない
