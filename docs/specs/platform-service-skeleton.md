# Platform Service Skeleton 仕様

## 目的

このドキュメントは、Phase 6 の最初の online service skeleton として、
matchmaking 後に得た `game + players` を `submission -> admission -> queue -> worker -> runner -> terminal persist`
で流す最小内部契約を定義する。

この spec は public HTTP API ではなく、service / worker / runner 間で共有する internal contract を固定する。

## この spec の責務範囲

この spec が定義するもの:

- `match submission` の最小 schema
- admission validation の責務範囲
- queue / execution lifecycle state machine
- service / worker / runner / registry / game master の責務境界
- terminal persist で最低限残す artifact

この spec が定義しないもの:

- public HTTP API
- AI upload registry の durable external contract
- matchmaking, ranking, rerun policy
- replay / resume read model の詳細
- distributed worker 運用や retry 自動化

## 参照関係

- `docs/specs/platform.md`: platform core、runner、artifact layout の正本
- `docs/specs/platform-common-contract.md`: metadata / record core schema の正本
- `docs/specs/ai-runtime.md`: AI manifest / runtime kind / stderr capture の正本
- `docs/specs/platform-game-registry.md`: registered game lookup と build 入口の正本

## サービス境界

online service skeleton は、single-match runner を 1 段外側から包む orchestration 層である。

- service は `match submission` を受け取り、admission 済みの要求だけを queue へ入れる
- worker は queue から 1 件を lease し、runner を 1 回起動する
- runner は queue ownership を持たず、1 試合分の execution engine に留まる
- terminal persist は service skeleton 側の責務であり、runner artifact を file-backed first で残す

## Match Submission

`match submission` は AI upload 自体ではなく、1 回の試合実行要求を表す。

最小項目:

- `submission_id`: この実行要求の一意識別子
- `match_id`: runner / artifact layout が共有する試合識別子
- `game`: `game_id` / `game_version` / `ruleset_version`
- `players[]`: 各 player の `player_id` と `artifact_ref`
- `output_dir`: terminal persist の base directory
- `attempt_count`: 将来 retry 用の拡張予約。初期 contract では `1` 固定

### `artifact_ref`

- `artifact_ref` は opaque locator / URI として扱う
- 初期の file-backed first 実装では local file path を使ってよい
- service / queue record は locator 文字列をそのまま保持し、path join や storage-driver 情報を queue schema に埋め込まない
- 実際の manifest 解決や runtime entrypoint 解釈は admission validation / worker 側 adapter が行う

### `players[]` の制約

- `player_id` は submission 内で一意でなければならない
- 各 player はちょうど 1 つの `artifact_ref` を持つ
- player 順序は runner にそのまま渡してよい

## Admission Validation

admission validation は queue に入る前に完了しなければならない。

### admission が確認すること

- submission schema が structurally valid であること
- `game` metadata が registry lookup 可能であること
- 各 `artifact_ref` から解決した AI manifest が存在し、`game_id` / `game_version` / `ruleset_version` が submission と互換であること
- runtime entrypoint が起動可能な最小形で存在すること
- `output_dir` が空でなく、terminal persist の base path として受け入れ可能であること

### runner dry-run の位置づけ

- admission は full match 実行を行わない
- ただし runner 側の dry-run entrypoint を使って、実際に必要な metadata 解決、manifest 解決、runtime 起動前の最小確認を共有してよい
- dry-run の責務は「queue に入れても起動前提が崩れないか」を確かめることであり、turn progression や結果生成は扱わない

## Queue / Execution Lifecycle

queue / execution lifecycle は `record.json.status` と別契約である。
`record.json.status` は 1 試合内部の実行結果を表し、queue lifecycle は service skeleton がその試合要求をどこまで進めたかを表す。

### 状態一覧

- `queued`: admission を通過し、まだ worker に lease されていない
- `leased`: worker が排他的に claim したが、runner への本実行開始前
- `running`: worker が runner 実行を開始した
- `persisting`: runner の terminal result を file-backed artifact へ保存中
- `completed`: terminal persist まで成功した
- `failed`: runner 実行または terminal persist が失敗した
- `canceled`: `queued` 中に cancel された

### 許可される遷移

- `queued -> leased`
- `queued -> canceled`
- `leased -> running`
- `leased -> failed`
- `running -> persisting`
- `running -> failed`
- `persisting -> completed`
- `persisting -> failed`

初期 contract では retry による `failed -> queued` の巻き戻しを持たない。

### cancel 制約

- cancel は `queued` 中のみ許可する
- `leased` 以降は 0045 系では扱わない
- `canceled` は terminal state とし、runner 実行を開始してはならない

## 責務境界

### service の責務

- submission 受付
- admission validation の実行
- queue record の作成
- worker claim と lifecycle 更新の orchestration
- terminal persist 成否の監査

### worker の責務

- `queued` job を 1 件 lease する
- lease 済み submission を runner invocation request へ変換する
- runner 終了後に terminal persist を呼ぶ
- lifecycle を `leased -> running -> persisting -> terminal` へ進める

### runner の責務

- 1 試合分の game master / player session lifecycle を実行する
- `record.json` source-of-truth を構築できる terminal record を返す
- queue state、worker lease、retry policy を持たない

### registry の責務

- `game_id + game_version major` lookup
- build 入口の提供
- backend 種別を runner / worker に漏らさない

### game master の責務

- game 固有 metadata、decision step、result、snapshot を返す
- queue / submission / persist policy を知らない

## Terminal Persist

terminal persist は既存 `output-dir` を使う file-backed first とする。

最低限残すもの:

- `<output-dir>/<match-id>/record.json`
- `<output-dir>/<match-id>/result-summary.json`
- `<output-dir>/<match-id>/<player-id>-stderr.log`

追加で `history.json`、`snapshot.json`、`exported-snapshot.json`、`structured-log.ndjson` を残してよいが、
service skeleton が terminal success / failure を判断する正本は `record.json` とする。

### stderr artifact

- player ごとの captured stderr は runner terminal status に関係なく保存対象に含めてよい
- `stderr` の本文は platform 共通 snapshot schema に埋め込まず、artifact file として分離する
- summary には必要なら `stderr` path 参照を含めてよい

## Retry と Attempt Count

- `attempt_count` は queue record に持ってよいが、初期 contract では常に `1`
- 自動 retry、backoff、dead-letter queue はこの spec の対象外
- retry policy を導入するときは `failed -> queued` 再投入条件と artifact 上書き / 別 attempt directory の扱いを別 spec で固定する
