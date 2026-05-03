# プラットフォーム仕様

## 目的

このドキュメントは、AI Arena の Phase 2a foundation におけるプラットフォーム層の仕様を定義する。
このフェーズの目的は、ゲーム非依存の platform core を local subprocess 上で成立させ、
後続の `echo-count` fixture と `janken` 実装の土台を固めることである。

最終的な提出形式は引き続き WASM を目指すが、このフェーズでは実行基盤の切り分けを優先し、
AI 実行アダプタはローカルプロセスを使う。

## Phase 2a スコープ

このフェーズで platform が満たすべきこと:

- JSON-RPC 2.0 + NDJSON による AI 通信
- 1試合中に AI プロセスを長寿命で維持する session モデル
- `init` / `turn` / `game_over` の platform 共通メソッド
- 同時行動と順番制の両方を扱える match loop
- game metadata compatibility 判定
- local subprocess runtime adapter
- match record / event log / snapshot / exported snapshot の最小 schema

このフェーズで扱わないこと:

- WASM runtime 統合
- black-box end-to-end CLI シナリオ
- `echo-count` / `janken` のゲーム実装詳細
- resume / replay 継続実行の CLI entrypoint

## コア責務

### プラットフォームの責務

- ゲーム定義を catalog に登録し、metadata 互換性を検証する
- 試合ごとに game master と player AI session を接続する
- 各 AI runtime の起動、標準 stream 接続、終了を管理する
- 各 decision window ごとに request を送り、締切までの response を集約する
- timeout / malformed protocol / mismatched id / illegal action / late response を区別して記録する
- game master へ player action outcome を渡し、状態遷移を進める
- 最低限の match record, event log, snapshot を生成する

### プラットフォームの非責務

- ゲーム固有 action schema の意味解釈
- 対外 HTTP API や永続化 backend の具体実装
- AI ソースコードのビルド
- 試合中の AI 間通信

## 試合実行の仕組み

1試合ごとに独立した試合部屋を作る。

```text
試合部屋
  match loop
    <-> game master
    <-> ai session 1
      <-> runtime adapter 1
    <-> ai session 2
      <-> runtime adapter 2
```

- `runtime adapter`: subprocess/WASM など実行手段の差分を吸収する層
- `ai session`: protocol request/response と timeout を扱う層
- `match loop`: game master と複数 session を束ねる層
- `game master`: ゲーム固有の状態遷移と勝敗判定を持つ層

protocol と game logic は分離する。runtime adapter と session / match loop も分離する。

## ゲーム metadata 契約

### 必須 metadata

各ゲームは少なくとも以下を持つ。

- `game_id`
- `game_version`
- `ruleset_version`
- `turn_mode`

例:

```json
{
  "game_id": "janken",
  "game_version": "2.1.0",
  "ruleset_version": "regular",
  "turn_mode": "simultaneous"
}
```

### 各フィールドの責務

- `game_id`: ゲーム系列そのものの識別子。別ゲームになったら変える
- `game_version`: game master 実装と payload schema の互換性を表す semver
- `ruleset_version`: 同一 game 上での運営ルール・採点・round 数既定など、必ずしも schema 変更を伴わないルール識別子

`ruleset_version` は、たとえば以下のような用途を想定する。

- 恒常ルールの `regular`
- 期間限定イベント用の `spring-festival-2026`
- season 制の `season-3`

### 互換性判定

- platform は `game_id` 完全一致を要求する
- platform は `game_version` の major 一致を要求する
- `ruleset_version` は完全一致を要求する

したがって、以下は非互換とみなす。

- `game_id` 不一致
- `game_version` major 不一致
- `ruleset_version` 不一致

minor / patch 差分は同一 major の範囲で互換とみなす。

## AI metadata sidecar manifest

AI 実行物の横に sidecar manifest を置く既定を持つ。

- 実行物 path: `<entry>`
- sidecar path: `<entry>.arena.json`

最小 schema:

```json
{
  "ai_id": "sample-janken-bot",
  "protocol": {
    "transport": "stdio-jsonrpc-ndjson",
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular"
  },
  "runtime": {
    "kind": "local-subprocess",
    "command": ["./bot"]
  }
}
```

sidecar が存在しない場合、Phase 2a の既定は以下。

- `runtime.kind = local-subprocess`
- command は登録時に明示された実行コマンドを使う
- protocol metadata は match 側設定と同一でなければならない

## AI 実行モデル

### Phase 2a runtime

- runtime は local subprocess を使う
- platform はプロセス起動後、`stdin` / `stdout` / `stderr` を接続する
- AI は試合開始から終了まで同一プロセスで生存できる
- `stdout` は JSON-RPC response 専用
- `stderr` は自由ログであり、platform が capture する

### 起動確認

`init` 送信前に platform は少なくとも以下を確認する。

- subprocess 起動成功
- `stdin` / `stdout` / `stderr` pipe 接続成功
- `stdout` 受信ループ開始成功

これに失敗した場合、その AI は `init` 前起動失敗として扱う。

### 終了

- `completed` path では、platform は `game_over` notification を送る
- `failed` / `canceled` path では、`game_over` と shutdown cleanup を同一視しない
- すべての terminal path で、platform は session shutdown を試みる
- shutdown 猶予内に終了しなければ強制終了してよい

## 通信プロトコル

### Envelope

platform request:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-3-p2",
  "method": "turn",
  "params": {}
}
```

AI response:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-3-p2",
  "result": {}
}
```

error response:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-3-p2",
  "error": {
    "code": -32000,
    "message": "illegal action"
  }
}
```

### NDJSON framing

- 1メッセージは1行の JSON object とする
- 行終端は `\n`、受信側は `\r\n` も許可してよい
- 複数行 JSON は無効
- `stdout` に JSON 以外の行を混在させてはならない

### 標準メソッド

- `init`: request / response
- `turn`: request / response
- `game_over`: notification

### `init`

必須 `params`:

- `match_id`
- `player_id`
- `game_id`
- `game_version`
- `ruleset_version`
- `deadline_ms`
- `state`

`init` response は protocol-ready ACK として扱う。ゲーム固有 readiness の詳細は各ゲーム仕様が定義する。

### `turn`

必須 `params`:

- `turn`
- `visible_state`
- `legal_action_hint`
- `deadline_ms`

`turn` response の `result` shape はゲーム固有である。

### `game_over`

- `id` を持たない notification とする
- AI response は不要
- `params` には少なくとも `final_visible_state` と `summary` を含められる
- 必要なら `shutdown_after_ms` を含めてよい

## Failure reason 分類

platform は action そのものと failure reason を分離して記録する。

### action outcome

- `accepted`: game master に渡せる action を受理した
- `no_action`: game master に渡す action が存在しない

### failure reason

- `invalid-timeout`: 締切までに response が来なかった
- `invalid-protocol-malformed`: JSON 破損、JSON-RPC envelope 不正、複数行 JSON など
- `invalid-protocol-mismatched-id`: request と異なる `id` の response
- `invalid-illegal-action`: protocol 上は正常だがゲーム仕様上の action validation に失敗
- `invalid-protocol-late-response`: timeout 後に届いた旧 response
- `runtime-stopped`: AI process exit、stdin write failure、stdout close などで session transport が継続不能になった

`accepted` と `failure reason` は同居しない。`accepted` のとき `failure reason` は空とする。
`no_action` のときだけ `failure reason` を持ってよい。

## Match loop 契約

### decision step

game master は次に必要な意思決定 step を platform に返す。
1 step は「この単位の request collection が終わったら game master に反映してよい」境界を表す。
decision step は以下を持つ。

- `turn`
- `mode`: `simultaneous` または `sequential`
- `requests`: player ごとの `turn` request payload

### simultaneous

- platform は対象プレイヤー全員へ同一 turn step の request を送る
- 全員の response か timeout を待つ
- 収集結果をまとめて game master に渡す

### sequential

- game master は sequential progression を step 単位で返す
- 1つの sequential step は1プレイヤー分の request だけを持つ
- platform は response を受けたら直ちに game master へ反映し、その後の次 step は game master が決める

### late response

timeout 済み request の response が後から届いても、後続 turn の response と混線させてはならない。
platform はそれを `invalid-protocol-late-response` として event log にのみ残し、現在の pending request には紐付けない。

## Record / Event / Snapshot

### match lifecycle phase

platform は少なくとも以下の lifecycle phase を内部で扱う。

- `starting`
- `initializing`
- `running`
- `finishing`
- `completed`
- `failed`
- `canceled`

`completed` / `failed` / `canceled` は terminal phase であり、match record の `status` と整合していなければならない。

### match record

match record は、1試合の最終結果をあとから参照するための最小記録である。
少なくとも以下の意味を持つ。

- `match_id`: 試合単位の識別子
- `game`: どのゲーム・どの互換系・どの ruleset で行われた試合か
- `players`: 参加プレイヤーと、その試合で使った AI 識別子の対応
- `status`: 試合の終了状態。少なくとも `completed` / `failed` / `canceled` を扱う
- `result`: 最終順位やゲーム固有の最終結果
- `event_log`: lifecycle event を含む時系列記録
- `snapshot`: terminal 時点で materialize した内部 snapshot
- `exported_snapshot`: terminal 時点で materialize した公開向け snapshot

`failed` / `canceled` の場合も、platform は可能な範囲で partial `event_log` / `snapshot` / `exported_snapshot` を残す。
また、各 player の stderr byte summary は terminal status に関係なく record / snapshot へ反映する。

`result.placements` は、順位確定済みのプレイヤー一覧を表す。
各要素は `player_id` と `place` を持ち、同順位を許すなら複数プレイヤーが同じ `place` を持ってよい。

最小 schema:

```json
{
  "match_id": "match-001",
  "game": {
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "phase2"
  },
  "players": [
    {"player_id": "p1", "ai_id": "bot-a"},
    {"player_id": "p2", "ai_id": "bot-b"}
  ],
  "status": "completed",
  "result": {
    "placements": [
      {"player_id": "p1", "place": 1},
      {"player_id": "p2", "place": 2}
    ]
  }
}
```

### event log

最小 event 共通項目:

- `seq`: その match 内で単調増加する event 順序番号
- `kind`: event の種別
- `turn`: どの turn / decision window に紐づく event か。試合全体 event では `0` を使ってよい
- `player_id`: 特定プレイヤーに紐づく event の場合だけ持つ
- `payload`: event ごとの補足データ。shape は `kind` ごとに異なる

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

`game_over_sent` は `completed` path の final notification event であり、`failed` / `canceled` path の cleanup event とは別に扱う。

### snapshot

snapshot は match 実行中または終了時点の現在状態を表す内部向け構造である。
最小で以下を持つ。

- `match_id`: どの試合の状態か
- `turn`: その snapshot が表す最新 turn
- `status`: 実行中か完了済みか、失敗/キャンセル済みかなどの内部状態
- `game_state`: game master が保持している内部状態のうち、platform が保存対象とする部分
- `per_player`: プレイヤーごとの直近状態

`per_player` には少なくとも以下を含められる。

- 最後に見せた `visible_state`
- 最後に得た action outcome
- その時点までに capture 済みの `stderr` byte 数

ここでの `stderr` byte 数は、保存済みログ量や上限消費量を追跡するためのメタデータであり、
`stderr` の本文そのものを snapshot に含めることは foundation の対象外とする。

### exported snapshot

exported snapshot は観戦・debug 用に外へ出せる shape で、内部 snapshot をそのまま露出しない。
foundation では最小で以下を持つ。

- `match_id`: どの試合の公開状態か
- `turn`: 観戦側が見ている最新 turn
- `status`: 公開上の試合状態
- `public_state`: 観戦や debug に出してよい公開状態
- `players`: プレイヤーごとの公開向け状態一覧
