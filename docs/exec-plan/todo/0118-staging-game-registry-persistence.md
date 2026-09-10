# staging-game-registry-persistence
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0118-staging-game-registry-persistence.md`

## Objective

admitted WASI game release を process-local registry だけに置かず、Postgres を executable
descriptor metadata の source of truth にする。service の restart / redeploy 後も、既存 scope と
過去 match が snapshot した immutable game artifact digest を exact に解決し、match admission と
worker execution が同じ descriptor を使える状態にする。

完了境界は、Postgres を使う fresh `arena-service` process が、以前に admission / game registration
された game bundle の artifact digest を DB だけから lookup でき、artifact bytes を R2/filesystem
から読むのは WASI session の materialize 時だけであることとする。DB-only lookup に memory cache は
導入しない。次の `0119-staging-game-registry-persistence-memory-cache.md` は、この durable lookup
が main に入った後にのみ実行する性能 follow-up である。

## 現行参照と採用設計

- `docs/issues/0118-staging-game-registry-persistence.md:1-57`: scope の list/get は
  `game_releases` を読む一方、exact artifact admission は現状 process-local WASI overlay だけを読む
  という staging の症状を記録している。
- `docs/specs/platform-game-registry.md:55-151`: immutable admitted release の保持、external-first
  選択、exact artifact lookup、永続 record と runtime resolver の分離を定める。
- `docs/specs/platform-artifact-bundle.md:5-18`: immutable digest identity と、bundle の
  storage/materialization を request acceptance から分離する契約を定める。
- `internal/platform/registry/registry.go:12-160`: `DescriptorRecord`、`RegistryStore`、
  `ArtifactRecordLookup`、登録、version lookup、exact artifact lookup を提供する。
- `internal/platform/registry/store.go:12-169`: admitted record を現状 `InMemoryStore` のみに
  保持する。`internal/platform/registry/wasm_resolver.go:19-106` は built-in の上に writable
  in-memory external tier を作り、後段で bundle を materialize する。
- `internal/platform/service/artifact_admission.go:12-58`: admitted game bundle manifest から record
  を作る。`internal/platform/service/general.go:185-274` は competition scope を保存する前にその
  record を読む。
- `internal/platform/service/general_postgres.go:16-150`: `game_releases` と scope を永続化するが、
  完全な record の再構築に必要な WASI runtime args / memory limit は未保存である。
- `internal/platform/service/postgres/schema/06_game_scopes_bots.sql:1-25` と
  `postgres/migrations/20260828000000_add_game_scopes_bots.sql:1-25`: schema 正本と現行 migration
  baseline。`cmd/arena-service/main.go:251-350` は admission、general registration、validator、worker
  path で共有する一つの overlay を作る。

### 永続化境界

- 成功した game-bundle admission は先に configured bundle store へ archive を書き、次に
  `game_releases` へ immutable な `DescriptorRecord` 一件を永続化する。archive write または validation
  が失敗した場合 descriptor row は書かない。DB failure により到達不能 object が残り得るが、admitted
  release を成功として返さず、補償的な R2 deletion も試みない。
- row は archive bytes を読まずに descriptor を再構築できる小さな検証済み field 全て、すなわち
  `game_id`、exact `game_version`、artifact digest、build mode、builder id、supported rulesets、
  WASI runtime args、memory-page limit を持つ。module path と ZIP/WASM bytes は digest が指す
  immutable bundle に残し、`BundleMaterializer.Materialize` だけが読む。
- competition scope の作成は既に admitted された exact release を scope へ紐付ける。built-in から
  descriptor を再構築せず、artifact/version fallback もしない。scope が新 release へ移っても既存
  release row は保持する。
- Postgres configured 時の external/admitted lookup は DB-backed にする。通常 key lookup は external
  tier の最大 admitted semantic version を選び、その DB tier に対象 key がない場合だけ built-in を見る。
  exact `LookupArtifact` は external-only とし、version/built-in へ fallback しない。DB outage/invalid
  data は built-in fallback ではなく error である。
- lookup は raw ZIP、module、manifest archive、R2 response を cache/load しない。static built-in fallback
  は plan 0119 で後続導入する external memory cache ではない。

## 変更対象

- `(MODIFY) docs/specs/platform-game-registry.md`: durable admission-before-scope、完全な小型
  descriptor metadata row、DB-primary external lookup/error、restart/redeploy resolution を定める。
- `(MODIFY) docs/specs/platform-artifact-bundle.md`: descriptor lookup は durable な
  manifest-derived metadata だけを消費し、大きな bundle payload は materialization 時だけ読むことを定める。
- `(MODIFY) internal/platform/registry/registry.go`: 必要な registration/lookup seam を context-aware にし、
  durable-store failure を隠さず external-tier fallback に必要な not-found distinction を公開する。
- `(MODIFY) internal/platform/registry/store.go`: durable primary が既存の external-first/exact-artifact
  規則を保てるよう external-primary/built-in-fallback composition を一般化し、unit/local lane の
  in-memory store は維持する。
- `(MODIFY) internal/platform/registry/wasm_resolver.go`: materializer 境界を変えず、injected durable
  primary と immutable built-ins から WASI overlay を作る。
- `(NEW) internal/platform/service/registry_postgres.go`: Postgres `DescriptorRecord` store/registrar と
  exact/key query、validation、semver selection、idempotent same-record admission、typed no-record error を実装する。
- `(MODIFY) internal/platform/service/general_postgres.go`: release admission と scope attachment を分離し、
  `PostgresGameRegistrationStore` が partial duplicate ではなく durable release を再利用する。
- `(MODIFY) internal/platform/service/artifact_admission.go`: bundle storage 後かつ admission result を返す前に
  complete admitted game descriptor を永続化する。
- `(MODIFY) internal/platform/service/general.go`: artifact-backed registration は exact lookup を保ち、
  manifest/descriptor-consistent な scope だけを既存 durable release に紐付ける。
- `(MODIFY) cmd/arena-service/main.go`: `ARENA_SERVICE_POSTGRES_DSN` 設定時に一つの Postgres primary
  registry を作り、同じ overlay を artifact admission、general registration、command validation、worker
  invocation へ注入する。Postgres 非設定時の in-memory/local behavior は維持する。
- `(MODIFY) internal/platform/service/postgres/schema/06_game_scopes_bots.sql`: desired
  `game_releases` schema に runtime args と memory-page column を加える。
- `(NEW) internal/platform/service/postgres/migrations/<timestamp>_persist_game_descriptor_runtime.sql`:
  deployed schema を互換的に拡張し、repository migration-hash command で `atlas.sum` を更新する。
- `(NEW) internal/platform/service/registry_postgres_test.go`: durable record admission、exact lookup、
  external latest-release selection、error boundary、store reopen を検証する。
- `(MODIFY) internal/platform/registry/registry_test.go`、`(MODIFY) internal/platform/service/artifact_admission_test.go`、
  `(MODIFY) internal/platform/service/general_test.go`、`(MODIFY) cmd/arena-service/main_test.go`:
  tier precedence、persisted descriptor field、scope attachment、validator/worker resolution を通る
  two-app-process restart path を検証する。
- `(DELETE) docs/issues/0118-staging-game-registry-persistence.md`: durable lookup verification 後の
  implementation PR でこの resolved local issue を削除し、証跡は本 plan と Git history に残す。

## black-box contract の変更

### durable admitted-game resolution

- accepted game bundle は operator が competition scope として登録する前に、durable metadata store の
  immutable descriptor record 一件を持つ。
- ready と表示される scope は durable admitted release を裏付けに持つ。新しい service process でも
  restart/redeploy 後にその exact digest を request admission と worker session construction に使える。
- 通常 lookup は external admitted release を優先し、requested major 内で最大 exact semantic version を選ぶ。
  external durable tier に対象 key がない場合だけ built-in を fallback とする。exact artifact identity lookup
  は fallback しない。
- durable metadata の欠落/不正または DB read failure は lookup failure として返す。別の built-in/version
  game を暗黙実行してはならない。list/get/lookup だけでは bundle object-store availability は証明しない。

### storage と failure の境界

- Postgres は compact な検証済み descriptor metadata を、R2/filesystem は archive/module bytes を保存する。
  Registry lookup は archive bytes を materialize、download、保持しない。
- object-store materialization は WASI session start 時だけに起きる。失敗時 run/request はその failure を返し、
  digest を別 release として再解釈しない。
- この plan は descriptor memory cache を意図的に導入しない。後続 plan が process-local state を正本にせず
  cache できる DB contract を確立する。

## サブタスク、順序、依存関係

1. 上記の durable metadata、external-first、exact lookup、bundle-byte boundary を二つの black-box spec に先に更新する。
2. missing runtime metadata field 二つの desired schema を拡張し、runtime code が使う前に互換 Atlas migration と
   migration checksum を作る。
3. Postgres descriptor store と generalized external-primary composition を実装し、通常 key lookup に限る
   built-in fallback を許す typed no-record behavior を持たせる。
4. artifact admission が complete record を永続化し、game registration が manifest/descriptor consistency
   validation を維持して durable exact record へ scope を紐付けるようにする。
5. 一つの Postgres-backed WASI overlay を全 service path へ接続し、no-Postgres local behavior と正しい
   close ownership を維持する。
6. unit、Postgres、fresh-app restart coverage を追加する。spec fixed 後の store/schema work（2--3）と
   admission/scope change（4）は並行可能であり、wiring/restart E2E は双方に依存する。
7. automated gate 後に manual local/staging operator acceptance を行う。これは
   `online-release-staging-verify` diagnostic CI lane に追加しない。

## 検証

- Registry test で external-first normal lookup、external-only exact lookup、greatest-semver selection、
  DB error が built-in へ fall-through しないことを証明する。
- Postgres test で non-default WASI args/memory limit を持つ record を admit し、store を close/reopen 後に
  key/digest 両方から byte-identical metadata を取得する。idempotency と conflicting immutable identity の
  reject も証明する。
- service integration test は一つ目の app instance で game を upload/admit/register し、同じ Postgres と
  bundle store を使う fresh app instance で exact-digest match の acceptance と worker materialization を行う。
  二つ目の process-local registry を preload してはならない。
- `make postgres-up`、`make postgres-migrate-hash`、schema/query generation が必要なら
  `make postgres-sqlc-generate`、focused registry/service/Postgres test、`make test-postgres`、`make test`、
  `make lint`、`git diff --check`、`./tools/workflow-lint.sh --mode=pre-push` を実行する。
- manual local/staging evidence として、service restart/redeploy 後に `/operator/games` へ既存表示された game が
  shown game digest で request を作成し、`registry: unsupported artifact` なしに worker materialization へ達する。
  release upload/registration は Reversi release asset を diagnostic CI に追加せず human-operated に保つ。

## 非目標と拒否条件

- bounded memory cache、cache metric、TTL knob、cache-based retry logic は加えない。これらは plan 0119 専用である。
- ZIP/WASM bytes、R2 locator、materialized module を Postgres に保存しない。
- old release row の削除、missing R2 object の auto-repair、consumer game repository の変更、game-name hard-code をしない。
- DB failure、exact-artifact miss、descriptor/manifest mismatch を別 artifact、latest version、built-in game への
  fallback にしてはならない。
