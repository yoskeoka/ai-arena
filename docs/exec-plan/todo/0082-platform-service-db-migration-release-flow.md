# platform-service-db-migration-release-flow
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0027-platform-service-db-migration-release-flow.md`

## Objective

schema change を伴う ai-arena backend release を、`desired schema update -> generated migration SQL review -> staging DB apply -> staging backend deploy -> staging verify -> production DB apply -> production backend/frontend deploy`
の一連の flow として repo-owned workflow に固定する。
最初の到達点は、今回の staging failure が `service_queue_records` schema 未適用だったことを
repo の release contract と runbook に反映し、今後の schema change が同じ事故を起こさない
運用面を閉じることに置く。

## Context

- `0059-service-postgres-schema-management-and-query-layer.md` は Atlas を使った desired schema / migration artifact / query layer の責務分離までを導入したが、
  deploy-time schema apply lane は scope 外だった
- `0074` は Phase 6 release flow を `local -> CI -> stg -> verify -> prod` の形に固定したが、
  DB migration step と secret contract は未定義のまま残っている
- current Postgres contract では runtime startup DDL を禁止し、schema apply は service 起動前提になっている
- current deploy workflow は Pages deploy と Render deploy hook までは repo-owned だが、
  Neon staging / production DB へ migration を apply する step がない
- Neon dashboard では pooled connection string が default であり、
  schema migration のような session-dependent operation では direct connection を別に取得する必要がある
- current secret naming proposal は、runtime は Render service env の `ARENA_SERVICE_POSTGRES_DSN`、
  release workflow migration lane は GitHub Actions secrets の
  `NEON_STAGING_MIGRATION_DSN` / `NEON_PRODUCTION_MIGRATION_DSN` とする
- staging DB には手動 migration を適用して current startup failure は解消済みだが、
  同じ操作を repo-owned release flow へ戻していないため再発余地がある

## Scope

- current staging failure の切り分けと runbook を追加する
- schema change PR に required な artifact contract を固定する
- staging / production release workflow に DB migration step を組み込む
- migration 用 direct/admin DSN と runtime 用 pooled DSN の secret contract を定義する
- Neon から secret value を取得・設定する手順を development doc に残す

この plan では以下を扱わない。

- 既存 migration を reversible にする高度な rollback automation
- drift detection や migration lint の高度化
- DB schema redesign 自体
- production release 実行そのものの完了保証

## Spec Changes

### `docs/specs/platform-service-postgres-stack.md`

- deploy lane では service 起動前に versioned migration SQL を target DB へ apply する責務を明記する
- runtime 用接続と migration/admin 用接続を別 contract として扱うことを追記する
- schema change PR は desired schema SQL と generated migration SQL を同じ review surface に含めることを明記する

### `docs/development/platform-service-postgres.md`

- schema change contributor flow を `desired schema edit -> make postgres-migrate-diff -> generated migration commit` に固定する
- staging / production apply lane で使う command surface を追記する
- direct/admin DSN を Neon Console の Connect modal から取得する手順を追加する
- pooled runtime DSN と direct migration DSN の使い分け、および次の推奨 secret 名を追加する
  - Render runtime env:
    `ARENA_SERVICE_POSTGRES_DSN`
  - GitHub Actions migration secrets:
    `NEON_STAGING_MIGRATION_DSN`
    `NEON_PRODUCTION_MIGRATION_DSN`

### `docs/development/platform-service-online-deploy.md`

- staging release に `DB migration apply -> Render deploy -> remote verify` の順序を追加する
- production release に `DB migration apply -> backend/frontend deploy` の順序を追加する
- required GitHub Actions secrets と、それぞれどこから取得するかを追加する
- migration lane は `NEON_STAGING_MIGRATION_DSN` / `NEON_PRODUCTION_MIGRATION_DSN` を読み、
  Render runtime は各 service 上の `ARENA_SERVICE_POSTGRES_DSN` を使い続ける前提を明記する
- staging で migration rehearsal を済ませてから production tag を作る運用を明記する

### `docs/specs/platform-service-skeleton.md`

- production-shape release flow の前提として、DB schema が deploy 前に target environment へ反映済みであることを補足する

## Expected Code Changes

- staging release workflow に migration step を追加する
- production release workflow に migration step を追加する
- migration apply 用の repo-local command or helper script を追加する
- GitHub Actions から Neon target DB へ apply するための env/secret wiring を追加する
- staging 切り分けに使う runbook / curl / SQL confirmation 手順を文書化する

## Sub-tasks

- [ ] current staging failure の切り分け手順と、手動 remediation を再現できる runbook を定義する
- [ ] schema change PR の artifact contract を spec に追記する
- [ ] migration apply helper の command surface を決める
- [ ] staging release workflow に `migration -> deploy -> verify` を組み込む
- [ ] production release workflow に同等の migration step を組み込む
- [ ] Neon direct/admin DSN と runtime pooled DSN の secret contract を定義する
- [ ] Neon Console から secret 値を取得し、GitHub Actions / Render に設定する手順を development doc に残す
- [ ] schema change の backward-compatible rollout policy を docs に明記する

## Parallelism

- [parallel] spec wording と secret inventory/runbook 叩き台は並行できる
- [depends on: migration apply helper] staging / production workflow への組み込みを進める
- [depends on: secret contract] doc の取得手順と workflow env wiring を固める

## Dependencies

- depends on: `0059-service-postgres-schema-management-and-query-layer.md`
- depends on: `0074-platform-online-foundation-03-04-matchmaking-ranking-follow-up-01-phase6-release-flow.md`

## Risks and Mitigations

- pooled DSN を migration に流用して PgBouncer 由来の session 制約や migration tool incompatibility を踏む
  - mitigation: runtime 用 pooled DSN と migration/admin 用 direct DSN を分離する
- migration apply を deploy より後ろに置くと、今回と同じ startup failure を再発する
  - mitigation: staging / production ともに migration step を backend deploy の前段へ固定する
- destructive schema change を 1 deploy で完結させると rollback 余地がなくなる
  - mitigation: expand / dual-read-write / cleanup の backward-compatible policy を運用ルールとして明記する

## Design Decisions

- schema change PR では desired schema SQL と generated migration SQL を同じ PR に含める
- migration apply は runtime startup ではなく CI/release workflow が担う
- runtime DB 接続は pooled DSN、migration/admin lane は direct DSN を使う
- production release は staging で同じ migration/deploy 順序を rehearsal した commit に限定する

## Verification

- staging migration remediation 後の release/verify evidence をもとに、workflow へ組み込んだ順序が同じ結果を再現できること
- local/CI verification が既存 path を壊さないこと
- staging / production workflow definition 上で required secret / input / ordering が追跡可能であること
