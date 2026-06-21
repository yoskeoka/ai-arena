# local-operator-completed-detail-missing-record-artifact

## Summary

local operator auth / verification 手順で preset queue から completed まで進んだ run に対し、
Completed Matches 一覧は `completed` として表示される一方、
Completed Detail を開くと `record.json` 欠損で panel 全体が error になる。

観測された error 例:

```text
service: get artifact object s3://ai-arena-local/tmp/operator-ui-browser-postgres/run-22b3f8de-ed5e-4ed4-8216-b0c03a171919/record.json: file does not exist
```

同じ run は Completed Matches 側では次のように成功扱いで見えていた。

- `lifecycle_state=completed`
- `terminal_status=completed`
- `turn=n/a`
- `worker=cli-worker`

この現象は、`result-summary.json` 欠損を degraded path に逃がしたあとも、
detail 側が replay input 構築で `record.json` を必読しているため継続している。

## Reproduction

ユーザー報告の local 手順:

```sh
make up
make migrate
make local-dummy-fixture
make start-backend-local
make start-frontend-local
```

前提:

- `.env` は main worktree から持ち込み済み
- local object storage は SeaweedFS
- backend は local auth / local object storage 前提

再現時の流れ:

1. operator login を通す
2. preset queue から match を enqueue する
3. run は Completed Matches に `completed` として現れる
4. Completed Detail を開くと `record.json` 読み取り失敗で error panel になる

## Current Understanding

- compact list view は `result-summary` と queue record の要約だけで成立するため、
  `record.json` が欠けても completed row 自体は出せる
- detail view は `internal/platform/service/query.go` から
  `buildReplayResumeAuditInputs()` を呼び、
  `internal/platform/service/replay_inputs.go` 内で `record.json` を source-of-truth として読む
- その `loadRecordFromLocator()` が失敗すると、detail response 全体が error になる
- したがって今回の本筋は「detail の degraded path 不足」だけでなく、
  そもそも completed run なのに `record.json` が object storage から読めない理由の切り分けにある

## Investigation Targets

次セッションでは、degraded path 追加だけで終わらせず、
`record.json` 欠損の根本原因を以下の観点で切り分ける。

1. 生成確認
   - local completed run 実行時に `record.json` 自体が runner output に生成されているか
   - `result-summary.json` はあるのに `record.json` だけ欠けるのか、両方とも欠けるのか
2. persist 経路確認
   - worker / persist 処理が `record.json` locator を queue record / DB に保存しているか
   - saved locator が `match_dir` や `result-summary` の artifact ref と整合しているか
3. object storage 確認
   - SeaweedFS へ put 自体が失敗していないか
   - bucket / object key / prefix の構築が `record.json` だけずれていないか
4. read path 確認
   - detail response が参照している locator と実在 object key が一致しているか
   - `s3://ai-arena-local/tmp/operator-ui-browser-postgres/run-.../record.json` という key shape が
     local persist 側の実際の書き込み先と一致しているか
5. state 整合性確認
   - `completed` へ遷移する条件が `record.json` persist 完了を本当に保証しているか
   - `result-summary` だけ persist できた partial success を completed と誤分類していないか

## Proposed Direction

- まずは local one-run の artifact 実体、DB 保存 locator、SeaweedFS object key を横並びで確認し、
  `record.json` 欠損が
  1. 未生成
  2. 未保存
  3. key 構築ミス
  4. completed 判定の不整合
  のどれかを確定する
- 根本原因が特定できるまでは、
  「detail 全体を error にしない」応急処置と
  「completed 判定の正当性」修正は分けて扱う

## Why It Matters

- `completed` 表示と detail 読み取り結果が矛盾しており、operator が local verification を信頼できない
- spec 上は `completed` が `record.json` / `result-summary.json` persist 完了を意味するため、
  実装がそれを満たしていないなら lifecycle contract の破れになる
- local SeaweedFS lane の問題を放置すると、
  staging / production の object storage path でも同種の locator 不整合を見逃しやすい
