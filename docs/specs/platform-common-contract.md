# プラットフォーム共通契約仕様

## 目的

このドキュメントは、AI Arena の platform / AI player / game master が共有する
共通契約の正本を定義する。Phase 3 では、個別 game の payload や game master 実装詳細とは
分離して、共通語彙・責務境界・record core schema をここに固定する。

## この spec の責務範囲

この spec が定義するもの:

- game 非依存の metadata 契約
- request / response / notification の共通メソッド契約
- decision step と action status の意味
- failure 分類
- match record / snapshot / exported snapshot の game 非依存コア schema

この spec が定義しないもの:

- JSON-RPC / NDJSON framing の細かな実装詳細
- 個別 game の payload schema
- game master 開発仕様の詳細
- registry 実装詳細

## 参照関係

- `docs/specs/platform.md`: platform core 全体像、runner contract、fixture appendix
- `docs/specs/game-master.md`: game master 開発者向けの論理 API と transport 契約
- `docs/specs/janken-game.md`: `janken` 固有 payload と validation
- repo 外へ切り出した game 固有仕様: game 開発側 repo がこの共通語彙を前提に差分だけを定義する

共通語彙の正本はこの spec とし、個別 game spec はここで定義した語彙を前提に差分だけを書く。

## sidecar SDK 候補へ載せる DTO 範囲

game master sidecar 実装者と platform adapter が共有してよい game 非依存 DTO は、
`github.com/yoskeoka/ai-arena/gamemaster` package に置く。

- external repo 側 consumer は、workspace 内の local checkout ではなく、review 済みの ai-arena module tag を
  `go.mod` から参照してこの package を取り込む
- external repo が安定依存してよい ai-arena 側 import surface は、この package に含まれる DTO / NDJSON helper までとする

- `GameMetadata`
- `DecisionMode`
- `Player`
- `InitState`
- `DecisionRequest`
- `DecisionStep`
- `ActionStatus`
- `Snapshot`
- `ExportedSnapshot`
- `MatchResult`

この spec で定義する共通語彙は、公開 sidecar SDK 候補へ載せる正本でもある。
一方で registry lookup、runtime 起動、session timeout 管理、catalog 解決の責務は
platform internal implementation に残し、上記 DTO に含めない。

## 共通 metadata 契約

各 game は少なくとも以下を持つ。

- `game_id`
- `game_version`
- `ruleset_version`

例:

```json
{
  "game_id": "janken",
  "game_version": "2.1.0",
  "ruleset_version": "regular"
}
```

### 各フィールドの責務

- `game_id`: ゲーム系列そのものの識別子。別ゲームになったら変える
- `game_version`: game master 実装と payload schema の互換性を表す semver
- `ruleset_version`: 同一 game 上での運営ルール・採点・round 数既定などを表す識別子

### 互換性判定

- platform は `game_id` 完全一致を要求する
- platform は `game_version` の major 一致を要求する
- platform は `ruleset_version` 完全一致を要求する

minor / patch 差分は同一 major の範囲で互換とみなす。

### `turn_mode` の扱い

`turn_mode` は Phase 2 では metadata に含めていたが、Phase 3 では共通 metadata から外す。

- 同時行動 / 順番制の進行形態は `DecisionStep.requests` と `DecisionStep.mode` が表す
- `turn_mode` を互換性判定の必須 key にしてはならない
- game 固有 spec が必要なら、ruleset 説明や game 固有 state に mode を載せてよい

## 共通 transport 前提

- AI との公式通信は request / response / notification モデルを前提にする
- platform は 1 試合中に同一 AI process を継続利用できる session モデルを前提にしてよい
- transport 実装が subprocess か WASM かはこの spec の責務外とする
- `stdout` は JSON-RPC response 専用であり、runtime kind に依らず JSON 以外を混在させてはならない
- `stderr` は AI の自由ログであり、platform が capture する
- `stdout` が閉じる、`stdin` に書けない、process が終了するなど transport 継続不能な事象は `runtime-stopped` に分類する

## 共通メソッド契約

標準メソッド:

- `init`: request / response
- `turn`: request / response
- `game_over`: notification 的意味を持つ request / response

### `init`

必須 `params`:

- `match_id`
- `player_id`
- `game_id`
- `game_version`
- `ruleset_version`
- `deadline_ms`
- `state`

`state` には game 固有の初期 payload を載せる。

### `turn`

必須 `params`:

- `turn`
- `visible_state`
- `legal_action_hint`
- `deadline_ms`

`visible_state` と `legal_action_hint` の shape は game 固有 spec が定義する。

### `game_over`

必須 `params`:

- `match_id`
- `final_visible_state`
- `summary`
- `shutdown_after_ms`

`summary` は game 固有の最終結果要約を載せてよいが、少なくとも record の `result` と矛盾してはならない。

## Decision Step 契約

game master は次に必要な意思決定単位を `DecisionStep` として platform に返す。

最小項目:

- `turn`
- `mode`
- `requests`

`mode` は以下を持つ。

- `simultaneous`
- `sequential`

`requests` は player ごとの `turn` request payload を表す。

### simultaneous

- platform は対象 player 全員へ同一 step の request を送る
- 全員の response か timeout を待つ
- 収集結果をまとめて game master に渡す

### sequential

- 1 step は 1 player 分の request だけを持つ
- platform は response を受けたら直ちに game master へ反映する
- 次 step の生成責務は game master にある

## Action Status 契約

platform が game master に渡す action status は以下のいずれかである。

- `accepted`: game master に渡せる action を受理した
- `no_action`: game master に渡す action が存在しない

最小 schema:

```json
{
  "player_id": "p1",
  "action_status": "accepted",
  "action": {}
}
```

または:

```json
{
  "player_id": "p2",
  "action_status": "no_action",
  "failure_reason": "invalid-timeout"
}
```

制約:

- `accepted` のとき `failure_reason` は空とする
- `no_action` のときだけ `failure_reason` を持ってよい
- game 固有 validator が action を棄却した場合も、game master に渡る値は `no_action` とする

## Failure 分類

failure 分類は action そのものと分離して記録する。

- `invalid-timeout`: 締切までに response が来なかった
- `invalid-protocol-malformed`: JSON 破損、JSON-RPC envelope 不正、複数行 JSON など
- `invalid-protocol-mismatched-id`: request と異なる `id` の response
- `invalid-protocol-late-response`: timeout 後に届いた旧 response
- `invalid-illegal-action`: protocol 上は正常だが game 固有 validation に失敗
- `runtime-stopped`: process exit、stdin write failure、stdout close などで session transport が継続不能になった

late response は current pending request の action status を書き換えず、event log にのみ残す。

## Record Core Schema

### Match Status

platform は少なくとも以下の lifecycle status を扱う。

- `starting`
- `initializing`
- `running`
- `finishing`
- `completed`
- `failed`
- `canceled`

`completed` / `failed` / `canceled` は terminal status である。

### Match Record

match record の最小共通項目:

- `match_id`
- `game`
- `players`
- `status`
- `result`
- `event_log`
- `snapshot`
- `exported_snapshot`

`result` の最小共通部分は `placements` とする。

```json
{
  "placements": [
    {"player_id": "p1", "place": 1},
    {"player_id": "p2", "place": 2}
  ]
}
```

### Event

event 共通項目:

- `seq`
- `kind`
- `turn`
- `player_id`
- `payload`

foundation で最低限必要な `kind`:

- `match_started`
- `session_initialized`
- `turn_requested`
- `turn_result`
- `turn_timeout`
- `protocol_error`
- `late_response_ignored`
- `runtime_exited`
- `game_over_sent`
- `session_shutdown_started`
- `session_shutdown_completed`
- `session_shutdown_failed`
- `match_failed`
- `match_canceled`
- `match_completed`

### Snapshot

snapshot の最小共通項目:

- `match_id`
- `game_id`
- `game_version`
- `ruleset_version`
- `turn`
- `status`
- `game_state`
- `per_player`

`per_player` の最小共通項目:

- `visible_state`
- `last_action_status`
- `stderr_bytes`

### Exported Snapshot

exported snapshot の最小共通項目:

- `match_id`
- `game_id`
- `game_version`
- `ruleset_version`
- `turn`
- `status`
- `public_state`
- `players`

`players` の最小共通項目:

- `player_id`
- `last_action_status`

## Snapshot / Exported Snapshot の責務境界

- `snapshot` は内部向けの現在状態であり、game master の内部状態保存を許す
- `exported_snapshot` は観戦・debug 向けの公開 shape であり、内部状態をそのまま露出しない
- replay/debug の source of truth は persisted final `record.json` とし、`snapshot.json` と `history.json` は derived artifact とする

## platform と game 固有仕様の分担

platform 共通契約が担うもの:

- metadata 互換性判定
- request lifecycle
- action status と failure 分類
- match record / snapshot / exported snapshot の game 非依存コア

game 固有 spec が担うもの:

- `state` / `visible_state` / `legal_action_hint` / `summary` の payload shape
- action validation 規則
- `accepted` / `no_action` を受け取った後の勝敗・状態遷移
- 観戦向け `public_state` の game 固有意味
