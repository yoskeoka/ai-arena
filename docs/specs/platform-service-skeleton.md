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

- operator entry は CLI-first とし、`match submission` JSON を受けて service command を呼ぶ薄い adapter から始める
- service は `match submission` を受け取り、admission 済みの要求だけを queue へ入れる
- worker は queue から 1 件を lease し、runner を 1 回起動する
- runner は queue ownership を持たず、1 試合分の execution engine に留まる
- terminal persist は service skeleton 側の責務であり、runner artifact を file-backed first で残す
- reviewer / operator 向けの最小 acceptance では、single-process CLI lane で `submit -> queued-only cancel` または
  `submit -> worker run -> terminal persist` を通せればよい

初期の CLI adapter は operator input を `Match Submission` schema に decode して service command へ渡すだけに留める。
artifact locator 解決、registry lookup、sidecar manifest 互換性確認、queue write は CLI ではなく service 側の責務とする。
local CLI invocation では、relative `output_dir` を invocation base directory 基準の local path へ正規化してから service command へ渡してよい。
initial acceptance 用の single-process CLI lane は、replaceable queue store の初期実装が in-memory だけであることを前提に、
1 回の command invocation の中で queue write、queued-only cancel、または worker 実行までを閉じてよい。

## Match Submission

`match submission` は AI upload 自体ではなく、1 回の試合実行要求を表す。

最小項目:

- `submission_id`: この実行要求の一意識別子
- `match_id`: runner / artifact layout が共有する試合識別子
- `game`: `game_id` / `game_version` / `ruleset_version`
- `players[]`: 各 player の `player_id` と `artifact_ref`
- `output_dir`: terminal persist の base directory
- `attempt_count`: 将来 retry 用の拡張予約。初期 contract では `1` 固定

最小 JSON shape:

```json
{
  "submission_id": "sub-1",
  "match_id": "match-1",
  "game": {
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular"
  },
  "players": [
    {
      "player_id": "p1",
      "artifact_ref": "file:///tmp/p1.wasm"
    },
    {
      "player_id": "p2",
      "artifact_ref": "s3://bucket/p2.wasm"
    }
  ],
  "output_dir": "arena-service-output",
  "attempt_count": 1
}
```

### `artifact_ref`

- `artifact_ref` は opaque locator / URI として扱う
- 初期の file-backed first 実装では local file path または `file://` URI を受け付けてよい
- `0049` の CLI-first 実装では local filesystem で即時に解決できる locator だけを admission 対象にし、`s3://` など remote backend locator は後続 plan へ残す
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
- `game` metadata の ruleset が registry descriptor の supported ruleset と整合すること
- 各 `artifact_ref` から解決した AI manifest が存在し、`game_id` / `game_version` / `ruleset_version` が submission と互換であること
- runtime entrypoint が起動可能な最小形で存在すること
- `output_dir` が空でなく、terminal persist の base path として受け入れ可能であること

初期実装では、local locator から得た entry path に対して `<entry>.arena.json` を sidecar manifest として探す。
manifest が存在する場合は transport / metadata / runtime schema を検証し、manifest が存在しない場合は plain local entry を
`local-subprocess` fallback として扱ってよい。どちらの場合も queue write 前に runtime entrypoint の最小 startability を確認する。

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

### queue lifecycle と runner terminal status の分離

- queue lifecycle は orchestration の進捗を表し、runner の試合結果そのものとは別に扱う
- `completed` は `record.json` / `result-summary.json` / player stderr artifact の persist が完了したことを表す
- runner が `failed` や timeout reason 付き `canceled` の terminal record を返しても、artifact persist が成功した場合は queue lifecycle を `completed` に進めてよい
- `failed` は runner invocation request を組み立てられなかった場合、runner が terminal record を返せないまま落ちた場合、または terminal persist 自体が失敗した場合に使う
- queue record は terminal artifact 参照に加えて、runner が返した terminal match status と terminal error summary を保持してよい

### cancel 制約

- cancel は `queued` 中のみ許可する
- `leased` 以降は 0045 系では扱わない
- `canceled` は terminal state とし、runner 実行を開始してはならない

## 責務境界

### service の責務

- submission 受付
- admission validation の実行
- queue record の作成
- replaceable queue store 越しの enqueue / cancel orchestration
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

### worker dispatch の最小ルール

- worker claim は `queued` record だけを対象にする
- claim 後は queue record に worker identifier を残し、同じ record を別 worker が再 claim してはならない
- worker は queue record から runner invocation request を materialize し、player artifact locator を local runtime entrypoint に解決して runner を 1 回だけ起動する
- 初期実装の worker は `attempt_count=1` を前提にし、run 中断後の retry / redelivery を行わない
- runner が terminal record を返したら、worker は queue lifecycle を `running -> persisting -> completed|failed` に進める
- terminal persist 完了後の queue record には、最低でも `match_dir`、`record.json` path、`result-summary.json` path、player stderr path 群、runner terminal status を残す

`output_dir` は terminal persist artifact の保存先であり、queue store の永続化責務を意味しない。
`0049` の queue store 初期実装は in-memory のみとし、cross-process durability や queue 再起動復旧は後続 plan で扱う。

## CLI-first Acceptance

最初の operator-facing acceptance は public API ではなく CLI で確認する。

### success path

- operator は local submission JSON を渡して admission 済み queue record を作成し、そのまま worker 実行で terminal persist まで進められる
- command 終了時には queue lifecycle が `completed` であり、runner terminal status は `record.json.status` から確認できる
- artifact 確認の既定順は `result-summary.json` -> `record.json` path / stderr path 群 -> 必要なら `structured-log.ndjson` とする

### rejection path

- admission validation に失敗する submission は queue record を作成してはならない
- operator は non-zero exit と validation error を受け取り、`output_dir` に match artifact directory が生えていないことを確認できる

### queued-only cancel path

- operator は `queued` 直後の submission だけを `canceled` にできる
- `canceled` command は runner 実行を開始してはならず、terminal artifact を生成してはならない

### reviewer 向け最小 manual verification

- success path:
  `go run ./cmd/arena-service run-once --submission <echo-submission.json> --base-dir <repo-root>`
- rejection path:
  `go run ./cmd/arena-service submit --submission <invalid-submission.json> --base-dir <repo-root>`
- queued-only cancel path:
  `go run ./cmd/arena-service submit-cancel --submission <echo-submission.json> --base-dir <repo-root>`
- success path の確認は stdout queue record に含まれる `result_summary_path` / `record_path` / `player_stderr_paths` を起点に行う
- rejection / queued-only cancel では `<output_dir>/<match_id>/` が生成されていないことを確認する

## Deferred Follow-ups

- `0056-platform-online-foundation-02-01-durable-store-and-write-model.md`:
  queue lifecycle / terminal locator の durable backend、single-node cross-process queue 共有
- `0057-platform-online-foundation-02-02-result-read-model-and-operator-query.md`:
  artifact locator を含む result read model、operator-facing list/get/read API
- `0058-platform-online-foundation-02-03-replay-resume-audit-inputs.md`:
  persisted match artifact を使う replay / resume / audit input contract
- `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md`:
  matchmaking 後の submission 生成、ranking / rematch / retry policy、leased 以降の cancel / operator recovery

## Retry と Attempt Count

- `attempt_count` は queue record に持ってよいが、初期 contract では常に `1`
- 自動 retry、backoff、dead-letter queue はこの spec の対象外
- retry policy を導入するときは `failed -> queued` 再投入条件と artifact 上書き / 別 attempt directory の扱いを別 spec で固定する
