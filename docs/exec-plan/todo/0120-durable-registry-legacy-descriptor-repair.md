# durable-registry-legacy-descriptor-repair
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A - staging regression in the already-admitted `reversi@1.0.0` release; no separate issue was requested.

## Objective

`runtime_args` と `memory_limit_pages` が存在しなかった時点で登録された admitted WASI game release を、同一 immutable artifact bundle の再 admission により安全に完全な durable descriptor へ修復する。staging の既存 `reversi@1.0.0` scope が、同じ game ZIP の upload 後に game activation と two-seat match request を作成できる状態へ戻す。

完了境界は、legacy row の二つの runtime field が `NULL` でも、operator が同一 digest の検証済み ZIP を再 upload すると manifest-derived metadata だけが補完され、exact artifact lookup と通常の `game_id + major` lookup が成功することである。異なる immutable identity を上書きせず、ZIP/WASM bytes を DB に保存せず、invalid durable record を lookup 時に黙って skip/fallback してはならない。

## 原因と採用設計

- `internal/platform/service/postgres/migrations/20260911000000_persist_game_descriptor_runtime.sql:1-3` は既存 `game_releases` に nullable な `runtime_args` / `memory_limit_pages` を追加する。既存 release の値を backfill せず、desired schema の `NOT NULL DEFAULT` とも一致しない。
- `internal/platform/service/registry_postgres.go:116-145` はこの二列のどちらかが `NULL` なら `registry: incomplete durable descriptor` を返す。key lookup は全 release row を scan するので、同じ major の `1.0.1` を追加しても legacy `1.0.0` row が先に scan された場合に失敗する。artifact conflict も現状は `DO NOTHING` のため、同一 ZIP の再 upload は legacy row を修復できない。
- `reversi-ai-arena/tools/release-packager/src/main.rs:9-16` の game manifest は runtime args と memory limit を省略する。この admitted artifact については manifest-derived value `[]` / `0` が正しい。一般の historical row に SQL default を推測で書くことは、過去 manifest が宣言した runtime budget を失うため採用しない。
- `docs/specs/platform-game-registry.md:76-115, 157-167` は complete immutable descriptor を admission 前に durable 保存し、invalid durable metadata を fallback せず lookup failure とする。この契約を維持して、validated exact re-admission だけを legacy repair entrypoint とする。

## 変更対象

- `(MODIFY) docs/specs/platform-game-registry.md`: 既存 artifact digest の再 admission は idempotent であり、legacy の欠落 runtime metadata だけを同じ verified manifest-derived descriptor で補完してよいこと、通常/exact lookup が incomplete row を fallback/skip しないことを明記する。
- `(MODIFY) internal/platform/service/registry_postgres.go`: durable register の conflict path を、immutable identity (`game_id`、exact version、artifact ID、build mode、builder、rulesets) が一致する場合に限って判定する。complete record は idempotent のままにし、runtime field が欠落する legacy record は transactionally backfill してから complete record として返す。不一致または validation failure は現在どおり conflict/error とする。
- `(MODIFY) internal/platform/service/registry_postgres_test.go`: raw legacy row の二 runtime field を `NULL` として seed し、same artifact record の registration が exact/key lookup の双方を回復することを追加する。complete idempotent record、runtime metadata を含む immutable mismatch、別 artifact の conflict は regression coverage を保つ。
- `(MODIFY) internal/platform/service/artifact_admission_test.go` または既存 Postgres integration test: verified game ZIP の re-admission が repair path を通り、scope activation / request admission が durable descriptor error にならない service boundary を確認する。
- `(MODIFY) docs/development/platform-service-online-deploy.md`: staging remediation は DB を直接推測更新せず、release asset の exact checksum を確認して同一 admitted game ZIP を operator upload し、activation と request creation の evidence を採る運用境界を記録する。

## black-box contract の変更

- accepted game bundle の immutable digest が既存 release と一致し、その verified manifest が同じ game identity / release metadata を表す場合、再 upload は idempotent である。過去の schema migration により runtime descriptor field だけが欠落していた場合は、その manifest-derived field を補完して release を再び usable にしてよい。
- repair は release の artifact digest、game ID、exact version、build mode、builder、ruleset metadata、scope identity を変更しない。異なる bundle、identity mismatch、または未検証の metadata は既存 release を更新できない。
- incomplete durable descriptor は通常 lookup、exact artifact lookup、activation、match admission のいずれでも error のままである。別 version、built-in game、default runtime budget、または artifact content を lookup 時に substitute しない。
- operator acceptance は既存 scope を消さず、checksum-confirmed exact ZIP の re-upload 後に `reversi-v1-standard` activation と selected two-seat submitted bots の request creation が成功することを確認する。worker materialization は別の artifact-content boundary のままである。

## サブタスク、順序、依存関係

1. spec と online deploy runbook を先に更新し、legacy repair の narrow な immutable-identity boundary と staging acceptance evidence を固定する。
2. Postgres registration conflict path を実装する。repair する前に supplied complete record を validation し、existing row の stable immutable fields が同一であることを transaction 内で確認する。欠落が二 runtime field 以外に及ぶ場合は repair しない。
3. legacy DB shape を deterministic に seed する test seam を加え、same-artifact re-admission の repair、exact/key lookup、scope/request service path を検証する。step 2 と unit-test scaffold は並行できるが、integration test は 2 に依存する。
4. staging では deployment 後、`reversi-game-v0.1.0.arena.zip` の release checksum と displayed artifact digest を照合して同一 ZIP を upload する。existing `reversi-v1-standard` scope の activation と `hoge2` / `hoge1` selected seats の match request creation を確認する。DB console の手修正、`v1.0.1` の作成、scope deletion は行わない。

## 検証

- PostgreSQL test で `runtime_args IS NULL OR memory_limit_pages IS NULL` の legacy row を作り、complete same-artifact record の `Register` 後に両 field が manifest record と byte/number-equivalent であり、`LookupArtifact` と normal `Lookup` が成功することを確認する。
- mismatch tests で game/version/build/builder/rulesets/artifact identity が異なる supplied record は legacy row を変えず error になり、complete same record の repeated registration は no-op であることを確認する。
- artifact-admission/service integration test で parsed verified bundle による re-upload、scope activation、two-player request validation が `registry: incomplete durable descriptor` を返さないことを確認する。incomplete row の lookup alone は error のままであることも確認する。
- `make postgres-up`、必要な migration hash/generation command、focused registry/service tests、`make test-postgres`、`make test`、`make lint`、`git diff --check`、`./tools/workflow-lint.sh --mode=pre-push` を実行する。
- deployed staging で exact release checksum/digest、ready scope、game activation、two-seat request creation の evidence を採る。upload HTTP success だけや scope list 表示だけを acceptance としない。

## 非目標と拒否条件

- 全 historical row への `[]` / `0` 一括 SQL backfill、DB console での手修正、archive/WASM bytes の Postgres 保存、object storage scan は行わない。
- `reversi@1.0.1` を作って old row を置換する回避策は取らない。old descriptor が残り続け、現行 normal lookup がそれを scan して failure し得るためである。
- lookup 時に invalid row を skip して最新 version を選ぶこと、built-in/version fallback、default budget の推測、artifact ID の置換は行わない。
