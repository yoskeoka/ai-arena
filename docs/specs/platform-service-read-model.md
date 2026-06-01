# Platform Service Read Model 仕様

## 目的

このドキュメントは、online service skeleton の operator-facing query surface として、
durable write model と terminal artifact locator から組み立てる最小 read model contract を定義する。

この spec は public HTTP API を定義しない。
CLI-first の list/get/read 入口から観測できる field と、
compact view と detail view の責務分離だけを固定する。

## この spec の責務範囲

この spec が定義するもの:

- operator が `list` で確認する最小 result row
- operator が `get` で確認する最小 detail view
- read model 上の service lifecycle state と runner terminal status の見せ分け
- compact result summary と artifact locator detail の既定読取順
- CLI-first の `list` / `get` / `read` query contract
- replay / resume / audit input locator group の detail exposure と local verification

この spec が定義しないもの:

- public HTTP API
- spectator 向け public state delivery
- ranking / matchmaking 集計
- retention / pagination / free-text search

## 参照関係

- `docs/specs/platform-service-skeleton.md`: submission / lifecycle / worker orchestration の正本
- `docs/specs/platform-service-persistence.md`: durable write model と artifact locator 保存単位の正本
- `docs/specs/platform.md`: runner artifact layout と compact artifact 読取順の正本

## Query Surface

operator-facing query surface は submission identity を主キーにした
`list` / `get` / `read` の 3 入口から始める。

- `list`:
  queue に存在する record を operator 向け compact row として返す
- `get`:
  1 submission の detail view と artifact locator 群を返す
- `read`:
  `get` が返す locator を起点に、selected artifact をそのまま読む補助入口とする

initial milestone では CLI adapter がこの query surface を expose し、
store backend や transport 差分を operator へ漏らさない。

## Lifecycle と Terminal Status の分離

read model は service lifecycle state と runner terminal status を別 field として扱う。

- `lifecycle_state`:
  `queued|leased|running|persisting|completed|failed|canceled`
- `terminal_status`:
  terminal artifact が存在するときだけ観測できる runner match status

`completed` は artifact persist 完了を意味し、
runner terminal status が `completed` / `failed` / `canceled` のいずれであっても取りうる。

`failed` lifecycle は、runner terminal record を保存できない失敗を表す。
この場合 read model は terminal artifact locator を持たなくてよい。

## Compact List View

`list` は operator の既定確認導線として compact row を返す。

1 row は少なくとも次を含む:

- submission identity:
  `submission_id`、`match_id`
- game metadata:
  `game_id`、`game_version`、`ruleset_version`
- service progress:
  `lifecycle_state`
- worker ownership:
  current `worker_id` when leased or later
- terminal summary:
  `terminal_status`、terminal error summary
- compact result:
  `turn`、placements
- artifact summary locator:
  `result_summary_path`

terminal artifact がまだない row では、
`terminal_status`、`turn`、placements、`result_summary_path` は空でもよい。

compact result の source-of-truth は `result-summary.json` とする。
write model が持つ terminal status / error summary は、
artifact 未作成または compact artifact 未読取のときの fallback として使ってよい。

## Match Detail View

`get` は 1 submission の detail view を返す。

detail view は少なくとも次を含む:

- `list` row と同じ compact field
- submitted players:
  player 順序、`player_id`、`artifact_ref`
- artifact locator group:
  `match_dir`、`record_path`、`result_summary_path`、player stderr locator 群
- delegated artifact access metadata:
  artifact kind ごとの短寿命 download URL または同等 token、expiry、issuer
- optional decoded compact summary:
  `result_summary`
- replay / resume / audit input group:
  `record_path` を source-of-truth とし、
  `snapshot_path`、`history_path`、`exported_snapshot_path`、
  game metadata、player 順序 / `artifact_ref`、artifact consistency verification を返してよい

initial milestone の detail は `record.json` や stderr 本文を必須で inline しない。
operator はまず compact summary と locator を受け取り、
必要な場合だけ `read` で個別 artifact を読む。
remote object storage lane では、detail view が object bytes の代わりに delegated artifact access metadata を返してよい。
この metadata は request 時に派生させる derived field であり、永続 write model の一部ではない。

## Artifact Read Contract

`read` は detail view から辿れる selected artifact をそのまま返す。

initial milestone では少なくとも次を対象にしてよい:

- `result-summary`
- `record`
- `snapshot`
- `history`
- `exported-snapshot`
- `stderr:<player_id>`

既定読取順は次の通りとする:

1. `result-summary.json`
2. `record.json` と player stderr locator
3. 必要なら `structured-log.ndjson`

`record.json.event_log` や `history.json` は source-of-truth / replay 入力として保持するが、
通常の operator 結果確認では最初の入口にしない。

remote object storage lane では、`read` は backend proxy を必須にしない。
artifact locator から provider-generated の delegated download URL / token を発行し、
client は object backend から直接取得してよい。

## Replay / Resume / Audit Input Detail

operator-facing detail view は、persisted match から replay / resume / audit 入力を辿る最小 surface を
同じ `get` response に載せてよい。

- replay source-of-truth:
  `record_path`
- resume helper:
  `snapshot_path`
- target-turn replay helper:
  `history_path`
- audit / public-state helper:
  `exported_snapshot_path`
- metadata join:
  game metadata、submitted player 順序 / `artifact_ref`
- local verification:
  `record.json` と persisted submission metadata の整合、
  `snapshot.json` / `history.json` / `exported-snapshot.json` が
  `record.json` 由来の derived artifact と一致するか

remote object storage lane では、detail view が locator と join metadata を返せればよい。
artifact 本文の照合や decode は、少なくとも local filesystem lane で verification できればよい。

## CLI-first Contract

initial CLI adapter は public API の代わりにこの read model を expose する。

- `list`:
  compact row array を JSON で返す
- `get --submission-id <id>`:
  1 detail view を JSON で返す
- `read --submission-id <id> --artifact <kind>`:
  selected artifact content を返す

JSON field 名は write model の既存 snake_case に合わせてよい。
CLI は local filesystem artifact lane から始めてよく、
remote object storage locator の取得方法や認可は後続 plan へ残す。

## Deferred Follow-Ups

- public HTTP query API
- pagination / filtering / sort options
- remote object storage artifact fetch
