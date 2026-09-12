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

## 確定した public visibility / access decision

public spectator は opt-in である。operator が match 作成時に `public` visibility を明示した logical `match_id` だけを
public resource に載せる。default は private であり、後から operator route の認可を弱めること、既存 private run を backfill
すること、signed operator artifact URL を再利用することはない。public resource 自体は anonymous read とし、product session は
要求しない。

`match_id` は public list/detail/latest/replay の唯一の resource key であり、client が `run_id` を指定して任意 attempt を
読むことはできない。server は completed official run がある場合はその run、ない場合は public match の current active/latest run
だけを解決する。retry が成功して初の completed run になればそれを official とし、rerun は explicit promote まで public view を
置換しない。promote は selection を atomically 切り替え、public-state version は `(match_id, selected_run_id)` scope で単調に
増加する。client は selected run の変更を新しい version namespace として扱い、旧 run response を捨てる。

| logical match visibility / selected run | list/detail | latest exported state | terminal replay |
| --- | --- | --- | --- |
| private | 非公開 | 非公開 | 非公開 |
| public, queued / leased | lifecycle のみ | `state_unavailable` | `replay_unavailable` |
| public, running / persisting | 可 | latest published exported state。未 publish は `state_unavailable` | `replay_unavailable` |
| public, completed official | 可 | final exported state | 可。public replay が bound を満たす場合だけ |
| public, failed / canceled / non-official rerun | generic lifecycle のみ | 最後に成功して publish 済みの state があれば可、なければ unavailable | `replay_unavailable` |

public detail は replay bytes を inline しない。terminal response は format/version/size/digest と availability だけを返し、
dedicated replay resource が game-produced artifact を最大 1 MiB まで bounded response として返す。1 MiB 超、欠落、retention 済み、
unsupported version は同じ public `replay_unavailable` result に正規化し、private locator、delegated credential、object-store error は
出さない。

## Existing References

- `docs/project-plan.md:116-125`: Phase 8 の spectator state / event stream / viewer 接続の milestone。
- `docs/specs/platform-common-contract.md:312-335` と `internal/platform/contract/snapshots.go`:
  `exported_snapshot` は game-specific `public_state` を持つ公開 shape であり、internal snapshot ではない。
- `docs/specs/platform-service-persistence.md:91-121`、`internal/platform/service/types.go:71-140`、
  `internal/platform/service/worker_local.go:37-96` と `internal/platform/service/worker_s3.go:22-94`:
  durable write model は artifact bytes ではなく stable locator と terminal summary を保持する。
- `docs/specs/platform-service-read-model.md:138-186` と `internal/platform/service/replay_inputs.go:20-67`:
  existing read model / replay inputs は operator / audit 用で、public response に転用してはならない。
- `typespec/main.tsp`、`typespec/namespaces/public/api.tsp:1-3`、`typespec/namespaces/shared.tsp`:
  public namespace は予約済みで、field-level public wire contract は未定義である。
- `docs/exec-plan/todo/0125-phase8-public-state-and-reversi-visualizer-roadmap.md:42-65`:
  terminal replay は `record.json` / `history.json` の filtering ではなく game-produced public artifact を入力にする。

## 変更 map

- `(MODIFY) docs/specs/index.md` と `(NEW) docs/specs/platform-public-spectator.md`:
  public resource の observable behavior、visibility、lifecycle、retention、cache / retry、unavailable response と private
  boundary を定義し、field inventory は TypeSpec へ委譲する。
- `(MODIFY) typespec/main.tsp`、`typespec/namespaces/public/api.tsp`、必要な shared TypeSpec:
  public API family、discover/list/detail、latest exported state、terminal replay resource と error model を定義し、OpenAPI / client
  generation target を public family へ追加する。
- `(MODIFY) gamemaster/types.go`、`internal/platform/game/game.go:74-85`、
  `internal/platform/gamemaster/gamemaster.go:126-147`、`internal/platform/match/match.go:43-104,523-556`:
  game master session に terminal-only `current_public_replay` request/response を追加し、`format`、`version`、opaque bytes を
  `match.Record` と `ExecutionResult` へ運ぶ。runner は initialize 後と committed turn 後に exported snapshot observer を呼び、
  public publisher が `(match_id, run_id, version)` を atomically 保存できるようにする。public replay は private event log を
  filter して生成してはならない。
- `(MODIFY) internal/platform/artifacts/artifacts.go:17-99`、`internal/platform/service/types.go:71-140`、
  `internal/platform/service/worker_local.go:173-230`、`internal/platform/service/worker_s3.go:22-94`:
  `public-replay.json` layout path と `TerminalArtifacts.PublicReplayPath/Format/Version/Size/Digest` を追加する。local/S3 persister は
  terminal completion 前に dedicated public artifact を write し、1 MiB limit を越える payload を public locator として persist
  しない。terminal public metadata は stable locator だけを durable に保持する。
- `(NEW) internal/platform/service/public_state.go`、`public_state_memory.go`、`public_state_postgres.go`、
  `public_state_test.go`、`public_http.go`、`public_http_test.go`:
  `PublicStateStore`/publisher/selecter を追加する。in-memory と Postgres lane は selected run、monotonic version、published exported
  snapshot、public visibility、replay metadata を atomic に扱い、`match_id` から official/current run を一意に選ぶ。public HTTP tree
  は `OperatorAPI.Handler` の `/api/v1/` protected mux と別 mux へ mount し、operator auth/CORS/ArtifactAccessIssuer を再利用しない。
- `(MODIFY) internal/platform/service/request.go:16-67`、`typespec/namespaces/operator/api.tsp`、
  `internal/platform/service/postgres/schema/service_queue_records.sql`、`postgres/query.sql`、generated `postgres/sqlc/*`、
  `postgres/migrations/<next>_public_spectator_state.sql`:
  create-time `public` opt-in を `MatchSubmission` と queue row に snapshot し、retry/rerun/promotion が同じ logical match の policy と
  official-run selection を保持できるようにする。request/read row は visibility を operator へ明示するが public list は private match を
  existence も含めて返さない。
- `(MODIFY) internal/platform/service/http.go:172-236` と `cmd/arena-service/main.go:205-223`:
  public API composition root、public store/publisher、filesystem/S3 reader を wire する。public API は `GET` only で anonymous
  handler を使い、operator API handler を wrapper として公開しない。
- `(NEW) internal/platform/service/public_state_*_test.go`、`public_http_*_test.go`、
  `internal/platform/artifacts/*_test.go` と `(NEW) versioned public fixture`:
  in-progress / terminal / unavailable と public/private artifact boundary を filesystem / S3-compatible lane で検証する。
- `(MODIFY) generated OpenAPI / client artifacts`:
  TypeSpec generation workflow が所有する出力だけを再生成する。
- `(DELETE)`: N/A。既存 operator read route、`record.json`、`snapshot.json`、`history.json`、stderr、AI/game bundle の public 化や
  rename は行わない。

## Black-box contract

- anonymous client は create-time public opt-in の match だけを discover / read できる。response は match identity、selected run identity、
  game metadata、lifecycle、monotonic public-state version/turn、opaque `public_state`、cache/retry hint と、terminal の場合だけ
  replay format/version/size/digest/availability を含む。replay bytes は dedicated bounded resource にしか含めない。正確な field 名と
  requiredness は TypeSpec を正本とする。
- running response は observer が atomically publish した exported snapshot だけから構成し、older `(run_id, version)` を newer
  state として返さない。client が stale
  response を識別でき、terminal lifecycle では polling stop を判断できる。
- terminal replay は game が生成し version を付けた public artifact のみを返す。platform は envelope、locator、size、version、
  retention を扱えるが、`record.json`、`history.json`、internal `snapshot.json`、structured log、stderr、AI/game bundle bytes を
  decode/filter/redirect して public replay にしてはならない。replay resource は 1 MiB を超える payload、欠落、retention、unsupported
  version を同じ unavailable result にし、public response に private locator/credential/backend error を含めない。
- retry/rerun/promotion を含めても public list は logical `match_id` ごとに一件だけを返す。completed official run がある場合はそれだけを
  read し、promotion は atomically selected run を置換する。non-official run は request parameter で選べない。
- artifact が retention 済み・欠落・未生成のときは private locator や storage credential を漏らさず、documented unavailable response
  を返す。terminal metadata だけが public policy 上許可される場合も replay payload を推測して補完しない。
- first transport は bounded snapshot polling とする。SSE、WebSocket、per-event cursor、reconnect protocol は追加しない。

## 実施順序と依存

1. fixed decision と lifecycle matrix を spec/TypeSpec に反映し、operator create-time public flag、public namespace、error model、
   generation target を review する。この契約確定より先に handler、publisher、artifact writer を変更しない。
2. match observer と `current_public_replay` producer protocol を追加し、in-flight exported state publication、public replay
   payload/format/version/size/digest、persist-before-terminal completion、filesystem/S3 parity を実装する。payload content は game repo
   の責務に残す。
3. durable public state/selecter と terminal locator を読む dedicated public read adapter を実装し、official/current run selection、
   cache/retry、lifecycle/retention mapping を anonymous route へ接続する。
4. versioned public fixture と public/private boundary test を追加する。fixture は B と C が private artifact なしで利用できる
   stable cross-repository input とする。
5. TypeSpec output を regenerate し、contract test、service tests、filesystem/S3-compatible lane、staging evidence を実施する。

steps 2 と 4 は TypeSpec と artifact contract が fixed になった後に並行できる。B の replay core は step 4 の fixture を受けて
別 repo で並行できるが、public HTTP resource への接続は C まで開始しない。

## 検証

- TypeSpec compile/generation と generated output の diff を確認し、public API contract test が required field、lifecycle、
  cache/retry、error model を検証する。
- filesystem と S3-compatible artifact backend の双方で、running publication の atomic version monotonicity、terminal payload の format/version/size/digest、
  retention/unavailable、stale response と terminal polling stop の判断材料を black-box test する。
- public client の request から、private `record` / internal snapshot / `history` / structured log / stderr / AI/game bundle / storage
  credential が response、redirect、error のいずれにも出ないことを negative test する。
- private/public と queued/running/persisting/completed/failed/canceled の anonymous matrix、retry/rerun/promotion selection、1 MiB
  boundary を black-box test し、operator route を呼ばないことを request-level test で示す。
- remote staging では provider deploy revision、exact `/version`、`/healthz` readiness、public API response を独立した証跡で確認する。

## 後続

- `reversi-ai-arena/docs/exec-plan/todo/0001-phase8-public-state-and-reversi-visualizer-reversi-replay-viewer.md` は本 plan の versioned public fixture と terminal replay
  envelope を入力にする。
- A の実装後に `0128` を新しい詳細 execution plan へ分解し、public resource と Reversi viewer の network connection を扱う。
