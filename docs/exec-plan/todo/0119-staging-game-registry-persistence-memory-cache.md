# staging-game-registry-persistence-memory-cache
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A - performance follow-up after `0118-staging-game-registry-persistence` is implemented and its local issue is deleted.

## Objective

plan 0118 の durable admitted-game descriptor store の手前に、bounded process-memory cache を加える。
cache は memory を先に読み、miss または expired entry のとき Postgres を読み、検証済みの小さな
`DescriptorRecord` を cache する。last-used entry を保持し、各 entry の TTL は固定 60 分、capacity は
既定 100 件とし、environment variable で変更可能にする。

完了境界は、cache hit が DB lookup を回避し、miss が DB result だけを cache へ入れ、TTL / LRU /
capacity / invalidation が deterministic に検証されることとする。R2/filesystem の archive/WASM
payload は一切 memory cache に載せない。cache-sourced descriptor の materialization が失敗したときは、
その descriptor の cache entry を無効化して HTTP request には retry を促す 500 を返す。同じ request
で DB lookup を再試行する複雑な fallback は実装しない。

## 前提条件と現行参照

- plan 0118 の merge 済み実装が必要である。これは Postgres primary `RegistryStore`、durable complete
  `DescriptorRecord`、validator/admission/registration/worker path で共有する `NewWASIOverlay` を提供する。
  同等の durable lookup が `origin/main` にない場合、この plan を開始してはならない。
- `docs/specs/platform-game-registry.md:55-151`: cache が高速化してよいが変更してはならない immutable
  artifact と external-first lookup contract を定める。
- `internal/platform/registry/registry.go:82-160`: registry/store lookup seam。
  `internal/platform/registry/store.go:12-169`: record copy、key selection、tier behavior。
  `internal/platform/registry/wasm_resolver.go:19-106`: 後段の R2/filesystem materialization error が観測可能になる箇所。
- plan-0118 Postgres store と `cmd/arena-service/main.go:230-350`: durable-store construction と
  service environment configuration を所有する。`internal/platform/service/http.go:390-420` と shared
  error-status mapper は operator response behavior を所有する。

### cache 設計

- cache は external durable descriptor store だけを wrap する。built-in は static fallback のままであり、
  cache により exact artifact miss、durable error、unsupported external record を built-in/version fallback に
  変えてはならない。
- 一つの cache entry は小さな `DescriptorRecord` data の defensive copy、artifact digest、expiry time、
  LRU position を持つ。index は二つ、exact artifact-id index と、DB から選ばれた latest record 用の通常
  `RegistryKey` index とする。capacity は index reference 二つではなく descriptor entry 一件を一度だけ数える。
  read は recency を更新するが、60 minute expiry は更新しない。
- `ARENA_SERVICE_GAME_REGISTRY_CACHE_SIZE` 未設定は `100` とする。non-negative decimal value を必須とし、
  `0` はこの external descriptor cache を明示的に無効化する。negative、malformed、overflow value は
  availability/performance を暗黙に変えず service startup を失敗させる。TTL は正確に 60 分で environment override を設けない。
- DB miss/error は cache しない。durable registration/update は return 前に local process で affected artifact と
  normal key を invalidate する。他 service instance は固定 60 minute TTL まで older normal-key selection を
  使い得るが、immutable exact-artifact entry は identity-safe のままである。
- cache は bundle bytes、manifest archive、WASM module bytes、materialization path、R2 client、response を
  持たない。registry resolution は metadata-only に保ち、`Materialize` を唯一の artifact-content path とする。
- cache-served descriptor に基づく session が bundle を materialize できない場合、その descriptor の
  artifact/key entry を invalidate し、digest-safe operational event を出し、typed temporary
  artifact-unavailable error を返す。operator HTTP adapter は generic `retry again` response の `500` へ map する。
  同じ request で DB/R2 を retry せず、durable row/artifact object も delete しない。worker failure は通常の
  worker failure evidence を保ち、後の retry は invalidated key で durable lookup に戻る。

## 変更対象

- `(MODIFY) docs/specs/platform-game-registry.md`: cache transparency、bounded staleness、fixed
  TTL/capacity、write/materialization invalidation、no-fallback failure contract を追加する。
- `(MODIFY) docs/development/platform-service-online-deploy.md`: `ARENA_SERVICE_GAME_REGISTRY_CACHE_SIZE`、
  default/zero/invalid value、temporary artifact-materialization failure の operator observation boundary を記録する。
- `(NEW) internal/platform/registry/cache_store.go`: synchronized/injectable-clock LRU/TTL `RegistryStore`
  decorator、二つの index、defensive copy、optional record invalidation capability を実装する。
- `(MODIFY) internal/platform/registry/registry.go` と `(MODIFY) internal/platform/registry/store.go`:
  decorator 越しにも typed no-record/error semantics を保ち、registry を R2/HTTP と結合せず narrow な
  invalidate-after-materialization wiring を公開する。
- `(MODIFY) internal/platform/registry/wasm_resolver.go`: `BundleMaterializer.Materialize` failure 時に
  optional descriptor invalidator を呼び、typed temporary artifact-unavailable error を返す。cleanup と
  no same-request retry を維持する。
- `(MODIFY) cmd/arena-service/main.go`: capacity environment variable を parse/validate し、Postgres
  external primary だけを wrap して resulting invalidation path を shared WASI overlay に注入する。artifact
  payload は configuration/cache に入れない。
- `(MODIFY) internal/platform/service/http.go` と `(MODIFY) internal/platform/service/http_test.go`:
  temporary materialization error を information-leaking しない HTTP 500/retry response に map し、他の
  validation/status mapping は保つ。
- `(NEW) internal/platform/registry/cache_store_test.go`: hit/miss、60 minute expiration、capacity での
  LRU eviction、capacity zero、two-index coherence、copy isolation、error non-caching、local write invalidation、
  concurrent access を検証する。
- `(MODIFY) internal/platform/registry/registry_test.go` と `(MODIFY) internal/platform/service/admission_test.go`:
  cache 後も exact/external-first semantics が保たれ、fake materializer failure が cached record を invalidate して
  next lookup が durable store に達することを証明する。

## black-box contract の変更

- registered external game descriptor に対し、個別 service process は同じ durable lookup を local metadata
  cache から最大 60 分提供してよい。cache capacity は most recently used descriptor entry 既定 100 件であり、
  zero に設定して無効化できる。
- cache operation は selected immutable digest、external-first priority、exact artifact behavior、required DB
  error behavior を変更しない。release write は local relevant selection を invalidate し、他 instance の normal-key
  selection が stale であり得るのは固定 TTL の間だけである。
- large game artifact はこの cache によって process memory へ保持されない。request/session start は configured
  artifact backend から immutable digest を materialize する責務を引き続き持つ。
- cache-associated artifact materialization failure は caller に retry を促す temporary server failure を返し、
  descriptor の local metadata を invalidate する。別 game の暗黙 substitution や in-request DB retry はしない。

## サブタスク、順序、依存関係

1. plan 0118 の merged DB-only behavior を確認し、code より先に black-box spec/runbook を拡張する。
2. fake clock/source seam を持つ cache decorator と unit test を実装する。これは HTTP mapping と独立である。
3. capacity parse を加え、service composition root の Postgres primary を wrap する。capacity zero は明示的に
   testable な no-cache path として保つ。
4. materialization-error invalidation と typed HTTP mapping を加える。これは decorator の invalidation seam に依存する。
5. focused cache/registry/service test、full quality gate、cache code が ZIP/WASM payload を持たないことを示す
   local service run を行う。spec fixed 後 step 2/3 は重ねられ、step 4 は 2 の後、final integration は全てに従う。

## 検証

- deterministic fake-clock test で first DB read、60 分前の hit、60 分時点の expired miss、`N` での LRU eviction、
  `0` 時の no retention/cache access を示す。
- spy durable-store test で version/exact-digest miss が successful copied metadata のみ cache し、DB error/miss は
  cache せず、external-first/exact-only rule を保ち、release write が local relevant index を invalidate することを示す。
- fake bundle-materializer test で cache-served descriptor を failure させ、invalidation と temporary error
  classification を確認した後、later request が durable store に戻ることを確認する。failed request 中に second DB/R2 attempt
  を期待してはならない。
- HTTP test で retryable materialization response が 500 で、storage credential、locator、archive byte、
  substituted artifact identity を disclose しないことを確認する。
- focused Go test、`make test-postgres`、`make test`、`make lint`、`git diff --check`、
  `./tools/workflow-lint.sh --mode=pre-push` を実行する。pre-existing release を使う local/staging operator request を
  行う。restart/redeploy evidence は DB-only plan の acceptance に保ち、本 plan は diagnostic CI に release-asset upload を
  加えず cache behavior を確認する。

## 非目標と拒否条件

- archive、archive payload としての manifest、WASM bytes、path、R2 response、object-store client を cache しない。
  distributed cache や process restart をまたぐ cache persistence も加えない。
- configurable TTL、unbounded capacity、stale-while-revalidate、background refresh、automatic DB/R2 retry loop、
  cross-instance cache invalidation を加えない。
- materialization failure 後に durable release metadata/object bytes を delete せず、cache failure を
  built-in/version fallback に変換しない。
