# phase8-public-state-and-reversi-visualizer
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

operator-only artifact/read model とは独立した、exported-state-only の public spectator resource を実装する。
client は公開対象の match を発見し、running match では monotonic version を持つ latest exported state を polling でき、
terminal match では game が生成した versioned public replay payload を取得できる。private artifact を後処理で filter する
endpoint は作らない。

完了は TypeSpec を唯一の wire source とした public contract、game-produced public replay artifact の durable locator、
filesystem と S3-compatible backend における public read adapter と black-box verification である。Reversi UI、viewer と
platform の接続、SSE/WebSocket は含めない。

## 実装開始前の review decision

次の二点は security/product boundary を変えるため、implementation PR で暗黙に決めず、この plan PR の human review で
一つずつ選択して spec に記録する。選択がない場合は `/execute-task` を開始しない。

1. public discoverability: terminal を含む全 completed match を匿名 discoverable にするか、match 作成時の明示的な public
   visibility に限定するか。後者を選ぶ場合は visibility の owner、default、transition、backfill を同じ review で決める。
2. public read access: anonymous read を許可するか、既存 product session を要求するか。いずれでも operator authorization の
   緩和、operator route、signed operator artifact URL の再利用は許可しない。

## Existing References

- `docs/project-plan.md:116-125`: Phase 8 の spectator state / event stream / viewer 接続の milestone。
- `docs/specs/platform-common-contract.md:312-335` と `internal/platform/contract/snapshots.go`:
  `exported_snapshot` は game-specific `public_state` を持つ公開 shape であり、internal snapshot ではない。
- `docs/specs/platform-service-persistence.md:91-121` と `internal/platform/service/worker_local.go:173-230`:
  durable write model は artifact bytes ではなく stable locator と terminal summary を保持する。
- `docs/specs/platform-service-read-model.md:138-186` と `internal/platform/service/replay_inputs.go:20-67`:
  existing read model / replay inputs は operator / audit 用で、public response に転用してはならない。
- `typespec/main.tsp`、`typespec/namespaces/public/api.tsp:1-3`、`typespec/namespaces/shared.tsp`:
  public namespace は予約済みで、field-level public wire contract は未定義である。
- `docs/exec-plan/todo/0125-phase8-public-state-and-reversi-visualizer-roadmap.md:42-65`:
  terminal replay は `record.json` / `history.json` の filtering ではなく game-produced public artifact を入力にする。

## 変更 map

- `(MODIFY) docs/specs/index.md` と `(MODIFY) docs/specs/` の新規 public spectator companion spec:
  public resource の observable behavior、visibility、lifecycle、retention、cache / retry、unavailable response と private
  boundary を定義し、field inventory は TypeSpec へ委譲する。
- `(MODIFY) typespec/main.tsp`、`typespec/namespaces/public/api.tsp`、必要な shared TypeSpec:
  public API family、discover/list/detail、latest exported state、terminal replay resource と error model を定義し、OpenAPI / client
  generation target を public family へ追加する。
- `(MODIFY) gamemaster/types.go`、`internal/platform/contract/*`、artifact layout / writer contract:
  game master が opaque で versioned な terminal public replay payload を明示的に出力し、private record / history と別 artifact
  として persist できる共通 contract を追加する。
- `(MODIFY) internal/platform/service/*`、artifact backend adapters、HTTP public route wiring:
  durable locator と exported snapshot を読み、public contract にだけ map する read adapter を追加する。operator handler / auth
  middleware を流用せず、選択済み policy を専用 boundary で適用する。
- `(NEW) public API / service / artifact backend black-box tests` と `(NEW) versioned public fixture`:
  in-progress / terminal / unavailable と public/private artifact boundary を filesystem / S3-compatible lane で検証する。
- `(MODIFY) generated OpenAPI / client artifacts`:
  TypeSpec generation workflow が所有する出力だけを再生成する。
- `(DELETE)`: N/A。既存 operator read route、`record.json`、`snapshot.json`、`history.json`、stderr、AI/game bundle の public 化や
  rename は行わない。

## Black-box contract

- 選択済み visibility/access policy を満たす client だけが public match を discover / read できる。response は match identity、
  game metadata、lifecycle、monotonic public-state version/turn、opaque `public_state`、cache/retry hint と、terminal の場合だけ
  replay format/version/payload を含む。正確な field 名と requiredness は TypeSpec を正本とする。
- running response は保存済み exported snapshot だけから構成し、older version を newer version として返さない。client が stale
  response を識別でき、terminal lifecycle では polling stop を判断できる。
- terminal replay は game が生成し version を付けた public artifact のみを返す。platform は envelope、locator、size、version、
  retention を扱えるが、`record.json`、`history.json`、internal `snapshot.json`、structured log、stderr、AI/game bundle bytes を
  decode/filter/redirect して public replay にしてはならない。
- artifact が retention 済み・欠落・未生成のときは private locator や storage credential を漏らさず、documented unavailable response
  を返す。terminal metadata だけが public policy 上許可される場合も replay payload を推測して補完しない。
- first transport は bounded snapshot polling とする。SSE、WebSocket、per-event cursor、reconnect protocol は追加しない。

## 実施順序と依存

1. review decision を spec の visibility/access behavior に反映し、TypeSpec namespace、public error model、generation target を
   review する。この契約確定より先に HTTP handler や artifact writer を変更しない。
2. game-master output と artifact layout に public replay payload/format/version/locator を追加し、persist-before-terminal
   completion と filesystem / object-storage adapter parity を実装する。payload content は game repo の責務に残す。
3. durable terminal locator と latest exported snapshot を読む dedicated public read adapter を実装し、policy boundary、cache/retry、
   lifecycle/retention mapping を route へ接続する。
4. versioned public fixture と public/private boundary test を追加する。fixture は B と C が private artifact なしで利用できる
   stable cross-repository input とする。
5. TypeSpec output を regenerate し、contract test、service tests、filesystem/S3-compatible lane、staging evidence を実施する。

steps 2 と 4 は TypeSpec と artifact contract が fixed になった後に並行できる。B の replay core は step 4 の fixture を受けて
別 repo で並行できるが、public HTTP resource への接続は C まで開始しない。

## 検証

- TypeSpec compile/generation と generated output の diff を確認し、public API contract test が required field、lifecycle、
  cache/retry、error model を検証する。
- filesystem と S3-compatible artifact backend の双方で、running version monotonicity、terminal payload の format/version、
  retention/unavailable、stale response と terminal polling stop の判断材料を black-box test する。
- public client の request から、private `record` / internal snapshot / `history` / structured log / stderr / AI/game bundle / storage
  credential が response、redirect、error のいずれにも出ないことを negative test する。
- selected access policy の anonymous/session matrix と discovery visibility matrix を black-box test し、operator route を呼ばないことを
  request-level test で示す。
- remote staging では provider deploy revision、exact `/version`、`/healthz` readiness、public API response を独立した証跡で確認する。

## 後続

- `reversi-ai-arena/docs/exec-plan/todo/0001-phase8-public-state-and-reversi-visualizer-reversi-replay-viewer.md` は本 plan の versioned public fixture と terminal replay
  envelope を入力にする。
- A の実装後に `0128` を新しい詳細 execution plan へ分解し、public resource と Reversi viewer の network connection を扱う。
