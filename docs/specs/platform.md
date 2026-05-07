# プラットフォーム仕様

## 目的

このドキュメントは、AI Arena の platform core 全体像と runner / fixture の仕様を定義する。
Phase 3 では、AI player / platform / game master が共有する共通語彙そのものは
`docs/specs/platform-common-contract.md` に切り出し、この文書では platform core の責務、
runner contract、runtime boundary、fixture appendix との境界を扱う。

最終的な提出形式は引き続き WASM を目指すが、このフェーズでは実行基盤の切り分けを優先し、
AI 実行アダプタはローカルプロセスを使う。

## Phase 2a スコープ

このフェーズで platform が満たすべきこと:

- JSON-RPC 2.0 + NDJSON による AI 通信
- 1試合中に AI player session と game master session を長寿命で維持する session モデル
- `init` / `turn` / `game_over` の platform 共通メソッド
- 同時行動と順番制の両方を扱える match loop
- game metadata compatibility 判定
- local subprocess runtime adapter
- local subprocess game master adapter
- match record / event log / snapshot / exported snapshot の最小 schema

このフェーズで扱わないこと:

- WASM runtime 統合
- black-box end-to-end CLI シナリオ
- `echo-count` / `janken` のゲーム実装詳細
- resume / replay 継続実行の CLI entrypoint

## コア責務

### プラットフォームの責務

- ゲーム定義を catalog に登録し、metadata 互換性を検証する
- 試合ごとに game master session と player AI session を接続する
- 各 AI runtime の起動、標準 stream 接続、終了を管理する
- game master runtime の起動、標準 stream 接続、終了を管理する
- 各 decision window ごとに request を送り、締切までの response を集約する
- timeout / malformed protocol / mismatched id / illegal action / late response を区別して記録する
- game master metadata / lifecycle / state exchange / turn resolution error を記録する
- game master へ player action status を渡し、状態遷移を進める
- 最低限の match record, event log, snapshot を生成する

### プラットフォームの非責務

- ゲーム固有 action schema の意味解釈
- 対外 HTTP API や永続化 backend の具体実装
- AI ソースコードのビルド
- 試合中の AI 間通信
- trusted external game backend への実ネットワーク接続実装

## 試合実行の仕組み

1試合ごとに独立した試合部屋を作る。

```text
試合部屋
  match loop
    <-> game master session
      <-> game master runtime adapter
    <-> ai session 1
      <-> ai runtime adapter 1
    <-> ai session 2
      <-> ai runtime adapter 2
```

- `runtime adapter`: subprocess/WASM など実行手段の差分を吸収する層
- `ai session`: protocol request/response と timeout を扱う層
- `game master session`: game master との標準論理 API を扱う層
- `match loop`: game master と複数 session を束ねる層
- `game master`: ゲーム固有の状態遷移と勝敗判定を持つ層

protocol と game logic は分離する。runtime adapter と session / match loop も分離する。
player AI と game master は同じ stdio JSON-RPC transport を使ってよいが、論理 API は分離する。

## 参照関係

- `docs/specs/platform-common-contract.md`: metadata / action status / failure 分類 / record core schema の正本
- `docs/specs/game-master.md`: game master 開発者向けの論理 API と transport 契約
- `docs/specs/platform-game-registry.md`: registered game の lookup key / descriptor / build/replay 入口
- `docs/specs/janken-game.md`: `janken` 固有 payload / validation / ranking
- `docs/specs/dungeon-game.md`: 本命 game が要求する platform 性質

## ゲーム metadata と ruleset の扱い

platform は game を直接 `switch` で選ばず、game registry へ lookup して起動する。
lookup key は `game_id + game_version major` とし、`ruleset_version` は lookup 後に
descriptor の build 入口へ渡して game 固有 validation を受ける。

### runner と registry の責務分離

- runner の責務:
  - CLI / artifact / replay-debug entrypoint から `game_id` と `game_version` を集める
  - `game_id + game_version major` で registry lookup を行う
  - 必要なら game master 接続方式を明示し、lookup 済み descriptor に `ruleset_version` と player list を渡して fresh run / snapshot resume / history replay を起動する
- game registry の責務:
  - registered game を descriptor 単位で保持する
  - `game_id + game_version major` lookup を提供する
  - game master 接続形態と build/replay 入口をまとめて返す
- game 固有 build 入口の責務:
  - `ruleset_version` の妥当性判定
  - fresh run / snapshot resume / history replay で使う metadata の確定
  - game 固有 snapshot/history 解釈

### 必須 metadata

各ゲームは少なくとも以下を持つ。

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
- `ruleset_version`: 同一 game 上での運営ルール・採点・round 数既定など、必ずしも schema 変更を伴わないルール識別子

`ruleset_version` は、たとえば以下のような用途を想定する。

- 恒常ルールの `regular`
- 期間限定イベント用の `spring-festival-2026`
- season 制の `season-3`

### 互換性判定

- registry lookup は `game_id` 完全一致と `game_version` major 一致を要求する
- lookup 後の metadata 確定フェーズでは `ruleset_version` 完全一致を要求する

したがって、以下は非互換とみなす。

- `game_id` 不一致
- `game_version` major 不一致
- `ruleset_version` 不一致

minor / patch 差分は同一 major の範囲で互換とみなす。

`ruleset_version` の一致判定は registry key ではなく、各 game の build 入口が返す
metadata を runner / catalog が検証する段階で行う。

### `turn_mode` の再分類

Phase 2 では `turn_mode` を metadata に含めていたが、Phase 3 では互換性 metadata から外す。

- 同時行動 / 順番制の表現は `DecisionStep.mode` と `DecisionStep.requests` に寄せる
- game 固有 spec が mode を説明したい場合は、ruleset の説明や game state に載せてよい
- runner は `turn_mode` を注入せず、game master が返す decision step をそのまま実行する

## game master runtime boundary

platform は game master を Go object へ直結してもよいが、標準形は game master session を介した
runtime boundary として扱う。

- match loop の主導権は platform に残す
- turn progression の細部は game master が `DecisionStep.requests` で表す
- game master は request 対象 player 集合を明示する
- `DecisionStep.mode = sequential` のとき request 対象は 1 player だけでなければならない
- `DecisionStep.mode = simultaneous` のとき request 対象は複数 player でも 1 player でもよいが、
  platform はその step を同時処理 step として扱う
- sequential game で自動 skip したい player は request に含めないことで表現できる
- public state 更新や観測イベントの都合で強制 pass を明示したい場合、game master は
  `no_action` へ正規化される request を送ってよい

### game master session の最小論理 API

platform は game master session に対して少なくとも以下を呼べること。

- `InitializeMatch`
- `NextDecisionStep`
- `NormalizeAction`
- `ApplyDecisionResults`
- `CurrentSnapshot`
- `CurrentExportedSnapshot`
- `CurrentResult`
- `Shutdown`

in-process 実装も local subprocess 実装も、この論理 API に写像できなければならない。

### local subprocess と future external adapter

- Phase 3 で必須なのは local subprocess adapter である
- trusted external game backend は将来 adapter 差し替えで載せる前提とし、この段階では実ネットワーク接続を持たない
- platform から見える論理 API と metadata / lifecycle / snapshot / result 契約は、local subprocess と future external adapter の間で不変とする
- runner が任意の game master mode を切り替える機能は持たない。接続形態は registry に登録された game ごとに固定する

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

- `game_over` は request / response とする
- AI は、終了前 cleanup が完了したあとに ACK response を返す
- `params` には少なくとも `final_visible_state` と `summary` を含める
- `params.shutdown_after_ms` には、platform がその環境で最大何 ms 待つかを入れる
- AI は `shutdown_after_ms` を超えて cleanup を継続する前提を持ってはならない

最小 request:

```json
{
  "jsonrpc": "2.0",
  "id": "game-over",
  "method": "game_over",
  "params": {
    "final_visible_state": {},
    "summary": {},
    "shutdown_after_ms": 3000
  }
}
```

最小 ACK:

```json
{
  "jsonrpc": "2.0",
  "id": "game-over",
  "result": {
    "ack": true
  }
}
```

platform 側待機上限:

- platform は環境変数 `AI_ARENA_GAME_OVER_TIMEOUT` を読む
- 未設定時の既定は 3 秒
- local / CI では 3 秒を使う
- online 環境では暫定的に 30 秒を設定してよい
- `game_over` ACK が期限までに返らなければ shutdown failure として記録し、その後の process cleanup へ進む
- `shutdown_after_ms` 超過後に AI が `stderr` やその他出力を続けても、platform はそれを拾えることを保証しない
- したがって `shutdown_after_ms` 超過後の出力可視性は未定義とする。実装や環境次第で一部拾えることはあっても、contract 上は保証しない

## Failure 分類

failure 分類の正本は `docs/specs/platform-common-contract.md` とする。ここでは platform core 上の扱いだけを補足する。

platform は action そのものと failure reason を分離して記録する。

### action status

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

`arena-runner` が出す structured log は、上記 event を NDJSON の逐次 stream に写した観測用チャネルである。
最小共通項目は以下とする。

- `match_id`
- `seq`
- `kind`
- `turn`
- `player_id` if applicable
- `payload`

structured log は replay/debug の入力 source of truth ではない。
replay/debug plan が読むのは persisted final match-record artifact であり、log stream は進行中観測と将来のログ分析に使う。

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
- 最後に得た action status
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

## `arena-runner` CLI

Phase 2a の black-box verification は `arena-runner` を入口にする。
この CLI は単発 match を起動し、観測用 structured log stream と persisted final record artifact を分離して扱う。

最小 contract:

- `--game <game-id>`
- `--game-version <game-version>`
- `--ruleset <ruleset-version>`
- `--player player_id=entry-path`
- `--match-id <id>` は省略可能
- `--output-dir <dir>` は標準 artifact layout の base directory を指定する。省略時は `arena-runner-output` を使う
- `--log-output <target>` は file path または `stdout` を受け付け、省略時は `stdout` を使う
- `--persist-record <target>` は source-of-truth final match-record artifact の追加出力先として file path または `stdout` を受け付ける
- `--exported-snapshot-output <target>` は derived exported snapshot の追加出力先として file path または `stdout` を受け付ける
- `--match-timeout <duration>` は省略可能で、指定時はその duration 経過で match を `canceled` として打ち切る
- `--stderr-limit-bytes <n>` は省略時に既定値を使ってよい
- `--snapshot-input <path>` は hand-crafted snapshot または persisted final record から抽出した `snapshot.json` を受け付ける
- `--history-input <path>` は persisted final record の `event_log` を抽出した `history.json` を受け付ける
- `--record-input <path>` は source-of-truth persisted final match-record artifact を受け付ける
- `--target-turn <n>` は `--history-input` または `--record-input` と組み合わせて使う replay / resume の turn 境界を指定する

`echo-count` は platform fixture 用の最小 game であり、`janken` は richer integration 用の game として同じ runner contract に乗る。
runner が担保するのはゲーム非依存の起動・artifact・replay/debug entrypoint と registry lookup までであり、
hidden action reveal、simultaneous resolution、game-specific action schema、ranking / tie-break の正しさと
game 固有 snapshot/history 解釈は descriptor 配下の build/replay 入口と `janken` 側 spec / verification で担保する。

artifact hierarchy:

- source of truth は persisted final `record.json` 1 個に固定する
- `history.json` / `snapshot.json` / `exported-snapshot.json` は `record.json` から導出できる derived artifact とする
- structured log は進行中観測用の stream / archive であり、replay/debug の source of truth にはしない
- replay/debug の primary entrypoint は常に `--record-input <path>` とする
- `--history-input <path>` は `record.json` から切り出した `history.json` を直接使いたい場合の補助 entrypoint とする
- `--snapshot-input <path>` は hand-crafted debug 開始点も許す補助 entrypoint とする

標準 artifact layout:

```text
<output-dir>/
  <match-id>/
    record.json
    structured-log.ndjson
    snapshot.json
    exported-snapshot.json
    history.json
```

- `--output-dir` が指すのは base path であり、runner はその直下に `match-id` ごとの subdirectory を切る
- `--output-dir` に空文字や無効 path は許可しない。存在しない path は runner が作成を試み、作成できない場合は session 起動前に失敗させる
- `record.json` は platform record を加工せずそのまま露出した source-of-truth final match-record artifact とする
- `history.json` は `record.json.event_log` をそのまま JSON array として抜き出した file format とし、`--history-input` にそのまま再投入できる
- `snapshot.json` は `record.json.snapshot` をそのまま抜き出した derived snapshot とする
- `exported-snapshot.json` は `record.json.exported_snapshot` をそのまま抜き出した derived exported snapshot とする
- `structured-log.ndjson` は `stdout` に流れる structured log と同じ NDJSON record を保存する

出力:

- structured log の既定出力先は `stdout` とする
- `structured-log.ndjson` は `stdout` の置き換えではなく複製であり、標準 artifact layout に常に保存する
- `--log-output <target>` が file path の場合、structured log はその file にも NDJSON で追加出力する
- structured log は NDJSON で 1 レコード 1 行とし、少なくとも `match_started` / per-event / `terminal_snapshot` / `terminal_exported_snapshot` / `terminal_summary` を出す
- `terminal_summary` は少なくとも `status` を持ち、`completed` では最終 `result`、`failed` / `canceled` では failure summary を含められる
- `record.json` は fresh run / replay/debug のどちらでも標準 artifact layout に常に保存する
- `--persist-record <target>` が file path の場合、標準 `record.json` に加えて同じ final record をその file にも追加出力する
- `--persist-record stdout` の場合だけ、利用者が明示的に mixed `stdout` を選んだものとして structured log と final record の混在を許容する
- `--exported-snapshot-output <target>` 指定時は、標準 `exported-snapshot.json` に加えて selected debug entrypoint に対応する exported snapshot をその target にも追加出力する。fresh run では terminal exported snapshot、resume 開始時は continuation 前 exported snapshot を使う
- 起動前 metadata 不整合などで match を開始できない場合も、stderr に説明を出して非 0 終了する

AI metadata 読み取り:

- `--player` で指定した実行物の横に `<entry>.arena.json` があれば、それを優先して読む
- sidecar がある場合、`runtime.command` を実行コマンドとし、`protocol.game_id` / `game_version` / `ruleset_version` を compatibility 判定に使う
- sidecar がない場合、entry-path 自体を実行コマンドとし、protocol metadata は match 側設定と同一でなければならない

起動前 compatibility:

- runner は registry lookup で解決した game master metadata と各 AI metadata の `game_id` 完全一致を要求する
- runner は registry lookup で解決した game master metadata と各 AI metadata の `game_version` major 一致を要求する
- runner は build 後に確定した `ruleset_version` 完全一致を要求する
- どれか 1 つでも不一致なら match loop を開始しない

runner の非責務:

- turn 数を決めること
- decision mode を metadata から注入すること
- per-turn deadline を決めること

これらは game 側の ruleset / decision step contract に属する。runner は `game_id` と
`game_version` で registry lookup した descriptor に `ruleset_version` を渡して対象 game を起動するだけで、
match の進行条件そのものは game master が定義する。

replay/debug で読むべき source of truth は structured log stream ではなく persisted final `record.json` である。
必要に応じて `snapshot.json` / `history.json` をその artifact から抽出して使う。
通常の replay/debug entrypoint は `--record-input <path>` を優先し、hand-crafted 編集を前提にしてよいのは snapshot だけとする。

replay/debug entrypoint:

- `start-from-snapshot` は `--snapshot-input <path>` を使い、その snapshot を初期局面として新しい AI process で続きを実行する
- `resume-from-history-and-continue` は `--history-input <path>` または `--record-input <path>` と `--target-turn <n>` を使い、target turn 境界までの履歴を replay した後、その続きだけ新しい AI process で実行する
- `--record-input <path>` 指定時は persisted final record の metadata / snapshot / history を source of truth とし、未指定の `--game` / `--game-version` / `--ruleset` をそこから補える
- `--history-input <path>` は `history.json` を直接与えたい場合の補助 entrypoint であり、通常は `--record-input <path>` を優先する
- hand-crafted snapshot file は debug entrypoint として許可するが、AI process memory continuity は保証しない
- history replay は記録済み choice / timeout / protocol-failure を再問い合わせせず target turn 境界まで再現するが、AI process memory continuity や in-flight transport state の復元はしない
- replay/debug path も fresh run と同じ runner log contract に従うが、log stream 自体は replay source of truth とみなさない

local debug の既定導線:

```sh
go run ./cmd/arena-runner \
  --game echo-count \
  --game-version 2.0.0 \
  --ruleset phase2-simultaneous-3turn \
  --match-id local-debug \
  --player p1=./testdata/ai/echo/echo-ai \
  --player p2=./testdata/ai/echo/echo-ai
```

- この実行では structured log は `stdout` に流れ続け、artifact は `./arena-runner-output/local-debug/` に保存される
- 次の debug 操作は `record.json` を第一入口として始める

```sh
go run ./cmd/arena-runner \
  --record-input ./arena-runner-output/local-debug/record.json \
  --target-turn 2 \
  --match-id local-debug-resume \
  --player p1=./testdata/ai/echo/echo-ai \
  --player p2=./testdata/ai/echo/echo-ai
```

- `history.json` を直接使うのは `record.json` を介さず replay 境界だけ差し替えて試したい場合に限る

## `echo-count` fixture appendix

`echo-count` は platform 検証用 fixture であり、独立ゲーム仕様ではない。
目的は deterministic な payload で session / match / record の責務を black-box に閉じることにある。

Phase 3 の runtime boundary 検証では、同じ挙動を持つ fixture を 2 つの registered game として使い分ける。

- `echo-count`: in-process game master
- `echo-count-subprocess`: local subprocess game master

これは product 向け mode 切替ではなく、e2e で境界差分を明示的に踏むための fixture 分離である。

### metadata

`echo-count` は以下の ruleset を持つ。

```json
{
  "game_id": "echo-count",
  "game_version": "2.0.0",
  "ruleset_version": "phase2-simultaneous-3turn"
}
```

```json
{
  "game_id": "echo-count",
  "game_version": "2.0.0",
  "ruleset_version": "phase2-sequential-3turn"
}
```

```json
{
  "game_id": "echo-count",
  "game_version": "2.0.0",
  "ruleset_version": "phase2-simultaneous-2turn"
}
```

同時行動 / 順番制の違いは ruleset の意味と decision step の返し方で表現する。turn 数も ruleset に含める。

### ルール

- platform は各 decision window ごとに期待値 `expected` を player へ渡す
- AI は `{"echo": <expected>}` を返す
- 値が一致すれば `accepted`
- schema 不正または期待値不一致なら `invalid-illegal-action` を記録し、game master へは `no_action` として渡す
- timeout / malformed / mismatched id / late response / runtime stop も game master へは `no_action` として渡す
- `accepted` は 1 点、`no_action` は 0 点
- 最終順位は score 降順。同点は同順位

### ruleset 別進行

`phase2-simultaneous-3turn`:

- 同一 turn で全 player に同じ `expected` を送る
- 全員の結果が揃ってから score と public state を進める
- 3 turns で終了する

`phase2-sequential-3turn`:

- 1 turn の中で player 順に個別 request を送る
- 各 response を直ちに反映してから次 player の request を作る
- turn 完了後に score と public state を確定する
- 3 turns で終了する

`phase2-simultaneous-2turn`:

- 同一 turn で全 player に同じ `expected` を送る
- 全員の結果が揃ってから score と public state を進める
- 2 turns で終了する

全 ruleset で per-turn deadline は game 側仕様として 100ms を使う。
期待値列は deterministic に `1, 2, 3, ...` とする。

### payload 形

`init.params.state`:

```json
{
  "mode": "simultaneous",
  "turns": 3,
  "player_order": ["p1", "p2"]
}
```

`turn.params.visible_state`:

```json
{
  "turn": 1,
  "expected": 1,
  "score": {
    "p1": 0,
    "p2": 0
  }
}
```

`turn.params.legal_action_hint`:

```json
{
  "type": "object",
  "required": ["echo"]
}
```

`turn.result`:

```json
{
  "echo": 1
}
```

`game_over.params.summary`:

```json
{
  "placements": [
    {"player_id": "p1", "place": 1},
    {"player_id": "p2", "place": 2}
  ],
  "score": {
    "p1": 3,
    "p2": 2
  }
}
```

### record 上の扱い

- game master が受ける入力は `accepted` または `no_action` だけ
- `failure_reason` は platform record 側にのみ残す
- `invalid-illegal-action` は game validator が返した理由として記録する
- `accepted` では `failure_reason` を空にする

代表例:

```json
{
  "player_id": "p1",
  "action_status": "accepted",
  "action": {"echo": 2}
}
```

```json
{
  "player_id": "p2",
  "action_status": "no_action",
  "failure_reason": "invalid-timeout"
}
```

```json
{
  "player_id": "p2",
  "action_status": "no_action",
  "failure_reason": "invalid-illegal-action"
}
```

late response は当該 turn の action status を遡って変更せず、`late_response_ignored` event にだけ残す。

```json
{
  "kind": "late_response_ignored",
  "turn": 1,
  "player_id": "p2",
  "payload": {"response_id": "turn-1-p2"}
}
```

init failure と shutdown failure は lifecycle event として残す。

```json
{
  "kind": "runtime_exited",
  "turn": 0,
  "player_id": "p1",
  "payload": {"stage": "init"}
}
```

```json
{
  "kind": "session_shutdown_failed",
  "turn": 0,
  "player_id": "p2",
  "payload": {"stage": "close", "error": "context deadline exceeded"}
}
```

### snapshot / exported snapshot

`snapshot.game_state` は少なくとも以下を持つ。

```json
{
  "mode": "simultaneous",
  "turn": 3,
  "expected": 3,
  "score": {
    "p1": 3,
    "p2": 2
  }
}
```

`exported_snapshot.public_state` は内部状態をそのまま出さず、観戦に必要な公開情報だけを持つ。

```json
{
  "mode": "simultaneous",
  "resolved_turns": 3,
  "score": {
    "p1": 3,
    "p2": 2
  }
}
```
