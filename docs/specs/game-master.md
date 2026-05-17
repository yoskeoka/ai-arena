# ゲームマスター開発仕様

## 目的

このドキュメントは、AI Arena の platform に新しい game master を載せる開発者向けに、
platform と game master の標準契約を定義する。player AI 向け共通契約とは分離し、
game master が満たすべき metadata / lifecycle / state exchange / turn resolution / shutdown の
要件をここに固定する。

## この spec の責務範囲

この spec が定義するもの:

- game master metadata
- local subprocess を含む共通 transport 前提
- platform との初期ハンドシェイク
- decision step の返し方
- action normalization / validation / apply の責務分離
- snapshot / exported snapshot / result / shutdown 契約

この spec が定義しないもの:

- 個別 game の payload schema
- trusted external game backend への実ネットワーク接続手順
- 観戦用 HTTP API や persistence backend

## 参照関係

- `docs/specs/platform-common-contract.md`: metadata / decision step / action status / failure 分類の正本
- `docs/specs/platform.md`: platform core と runner / registry / adapter の責務
- `docs/specs/platform-game-registry.md`: descriptor と game master 接続方式の登録契約

## 公開境界と internal 実装の分離

game master sidecar 実装者が依存してよい公開境界は
`github.com/yoskeoka/ai-arena/gamemaster` package とする。

- この package は sidecar SDK 候補として、metadata / decision / action status / snapshot / result DTO と
  stdio JSON-RPC 2.0 + NDJSON helper を持つ
- external repo の sidecar は、local `replace` や workspace 参照ではなく、review 済みの ai-arena module tag を
  `go.mod` から参照してこの package を import する
- `cmd/dungeon-gamemaster` のような sidecar entrypoint は、game 固有 package とこの公開 package だけへ依存してよい
- `internal/platform/runtime` / `internal/platform/session` / `internal/platform/registry` / `internal/platform/catalog` は
  platform 側 consumer / adapter 実装であり、sidecar 実装者向けの依存先ではない
- local subprocess transport の method 名と payload 契約は、この公開 package を通じて安定させる
- 将来 sidecar 実装を別 repo へ移すときも、この公開契約は import path 置換以外そのまま使える状態を狙う

### external consumer 向け release 運用

- external repo が ai-arena 側へ安定依存してよい import surface は
  `github.com/yoskeoka/ai-arena/gamemaster` package までとする
- external repo は local workspace checkout や `replace ../ai-arena` ではなく、
  review 済み ai-arena module tag を `go.mod` から参照する
- ai-arena runner / platform host の更新は、consumer repo 側では
  「ai-arena version を上げる」操作として取り込む
- `v0.1.0` のような release tag を切る前に、少なくとも以下を確認する
  - `gamemaster` package の公開 DTO / NDJSON helper が sidecar 開発に必要な最小面を満たしている
  - `cmd/dungeon-gamemaster` のような sidecar entrypoint が game 固有 package と `gamemaster` package だけで build できる
  - external repo 側 import audit で `gamemaster` を越える ai-arena 依存が残っていない
  - external repo 側で local `replace` なしの tagged import build/test/CI が通る

## game master metadata

game master は少なくとも以下の metadata を返せること。

- `game_id`
- `game_version`
- `ruleset_version`

platform は registry lookup 後に選ばれた game master metadata と runner 入力の metadata を照合する。
不一致は compatibility error とする。

`turn_mode` は metadata に含めない。進行形態は runtime 上の `DecisionStep` が表す。

## transport 前提

Phase 3 の標準 transport は stdio 上の JSON-RPC 2.0 + NDJSON とする。

- 1 request / response は 1 行 JSON object
- `stdout` は JSON-RPC 専用
- `stderr` は監査・デバッグ用ログとして使ってよい
- local subprocess と future external adapter で transport 実装差分があっても、以下の論理 API は不変とする

game master は trusted component であり、player AI と違って turn timeout で action を失う主体ではない。
ただし transport 断、malformed response、unexpected exit は lifecycle error として記録される。

## 起動方式

### in-process

- repo 内実装の移行期間では Go object を in-process adapter で包んでよい
- ただし platform から見えるのは後述の論理 API だけであり、具体的な package / interface 名を
  外部契約として漏らしてはならない

### local subprocess

- platform が game master 実行ファイルを起動し、stdio JSON-RPC で接続する
- sidecar 実装は `github.com/yoskeoka/ai-arena/gamemaster` package を使って transport contract を実装する
- module version 更新は、意図的に採用する ai-arena release tag を上げる形で行い、unreleased checkout 差分への依存は持ち込まない
- game master 側は起動時引数または sidecar 相当の設定から、自身の `game_version` / `ruleset_version` を確定できること
- match ごとの players と resume snapshot は、後述の初期化 API で受け取る

### future external adapter

- 将来の trusted external game backend でも、platform から見える論理 API は同一とする
- 認証・接続確立・再試行戦略は後続 phase に送る

## 論理 API

platform は game master に対して、少なくとも以下の責務を順に扱えること。

### `Metadata`

- game master が提供する metadata を返す
- 返した metadata は platform runner 入力と互換でなければならない

### `InitializeMatch`

この API は、match ごとの初期状態確定を担う。

入力:

- `players`
- `rng_seed`
- `resume_snapshot` または `null`

出力:

- `init_state`

`init_state` は player ごとの `init.state` payload を含む。fresh run でも resume run でも、
platform はこの API が返した state を各 player session へ送る。

`rng_seed` は string とする。game 固有に未使用でも受け取れてよい。seed-aware な game では human/debug-friendly
な seed 表現を受け取り、この値を `snapshot.game_state` と `exported_snapshot.public_state` に保持できなければならない。
具体的な seed encoding や内部 PRNG 選択は game 固有仕様に委ねる。fresh run で `rng_seed` が省略された場合、
初期 seed を生成して以後の snapshot/exported snapshot に保持する責務は game master 側にある。

### `NextDecisionStep`

この API は、次に platform が誰へ問い合わせるべきかを返す。

出力:

- `DecisionStep` または `null`

`null` は game master が追加の意思決定を要求せず、試合が終端へ到達したことを表す。

### `ApplyDecisionResults`

この API は、platform が集約した action status を受けて game state を進める。

入力:

- 直前に返した `DecisionStep`
- platform が集約した player ごとの `ActionStatus`

game master はここで状態遷移を進める。
simultaneous turn の game では、request 集約後に game 固有 spec で固定された phase 順へ従って progression を進める。
たとえば dungeon では action normalization -> movement resolution -> interaction resolution ->
terminal / score update -> visibility refresh の順を deterministic に適用する。

### `NormalizeAction`

- platform が集約した player response を game 固有 schema で正規化する責務
- malformed / timeout など transport 由来の `no_action` は platform が先に付与し、
  game master は game 固有の invalid action を追加で `no_action` へ落としてよい
- local subprocess adapter では `normalize_action` method として transport へ写像してよい

### `CurrentSnapshot`

- 内部向けの権威的 snapshot を返す
- `per_player.visible_state` を含め、resume source of truth として使えること

### `CurrentExportedSnapshot`

- 観戦・公開向けの exported snapshot を返す
- seed-aware な game では、再シミュレーションに使える seed を external/public exported snapshot に含めてよいのは terminal かつ `completed` の場合だけとする
- hidden information を含めるかどうかは game 固有 spec が決める

### `CurrentResult`

- 現時点の match result を返す
- `NextDecisionStep = null` 後は最終結果と一致しなければならない

### `Shutdown`

- game master の後始末を行う
- すでに終端状態であっても idempotent に呼べることが望ましい

## `DecisionStep.requests` の意味

`DecisionStep.requests` は、game master がこの step で platform に問い合わせたい player 集合そのものを表す。

- 複数 player を含む step は同時処理
- 1 player だけを含む step は逐次処理
- `DecisionMode = sequential` のとき request 対象は 1 player に限る
- `DecisionMode = simultaneous` のとき request 対象が 1 player でも、platform は同時処理 step として扱う

turn progression の細部は game master が決めるが、request 実行、timeout 判定、failure 記録、
record 生成は platform の責務である。

## action normalization / validation / apply の責務

- platform は player response を protocol 上 `accepted` または `no_action` に正規化する
- game master は game 固有 schema に照らして action を検証し、違反時は `no_action` として扱う
- game master は `ApplyDecisionResults` で `accepted` / `no_action` だけを入力として状態遷移を進める
- `failure_reason` は record / event log の監査情報として保持し、game 固有 state へ埋め込む必要はない

## skip / 強制 pass

既定動作:

- 自動 skip したい player は `DecisionStep.requests` に含めない

明示的強制 pass:

- public state 更新や観測イベント都合で、その turn に player を明示的に登場させたい場合は
  強制 pass 用 request を返してよい
- その request は game master 側 validation により `no_action` として正規化される実装でもよい

どちらの表現を使ってもよいが、同一 ruleset 内では一貫性を持たせること。

## local subprocess JSON-RPC メソッド

local subprocess adapter では以下の method 名を使う。これは transport mapping の契約であり、
上の論理 API そのものを current implementation の symbol 名へ固定する意図ではない。

- `metadata`
- `initialize_match`
- `next_decision_step`
- `normalize_action`
- `apply_decision_results`
- `current_snapshot`
- `current_exported_snapshot`
- `current_result`
- `shutdown`

これらの request / response payload は上記論理 API と 1 対 1 に対応する。
payload DTO と NDJSON framing helper の正本は `github.com/yoskeoka/ai-arena/gamemaster` package とする。

## 実装者向け最小チェックリスト

- metadata が `game_id + game_version + ruleset_version` を返せる
- `InitializeMatch` で player 初期状態を返せる
- `NextDecisionStep` と `ApplyDecisionResults` で match loop を進められる
- `CurrentSnapshot` / `CurrentExportedSnapshot` / `CurrentResult` が常に取得できる
- `Shutdown` が呼ばれても異常終了しない
- local subprocess でも in-process でも、同じ game 固有 spec に従う
