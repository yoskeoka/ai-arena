# Platform Service Ranking Lifecycle 仕様

## 目的

このドキュメントは、Phase 7 の online operator cycle で扱う
ranking aggregate の最小 lifecycle を定義する。

ここで固定するのは、どの completed run を ranking input とみなし、
どの単位で durable snapshot を更新し、
retry / rerun / correction とどう整合させるかである。

## この spec の責務範囲

この spec が定義するもの:

- ranking input の source-of-truth
- logical `match_id` と `official_run_id` に基づく集計単位
- first ranking aggregate の competitor identity
- durable ranking snapshot の最小 field
- incremental update と full recompute の責務分離
- CLI-first の ranking read / verify helper surface
- leaderboard family への引き継ぎ境界

この spec が定義しないもの:

- public leaderboard UI
- season / division / reward system
- game 固有 score を使う独自順位式
- public HTTP ranking API

## 参照関係

- `docs/specs/platform-service-read-model.md`: per-run read model の正本
- `docs/specs/platform-service-match-request-scheduling.md`: request / match-run-group / correction 単位の正本
- `docs/specs/platform-service-general-submission.md`: admitted AI identity の正本
- `docs/specs/platform.md`: `result-summary.json` を含む terminal artifact contract の正本
- `docs/specs/platform-service-single-worker-assumptions.md`: current non-atomic official selection / ranking refresh の前提

## Ranking Input

first ranking lifecycle の source-of-truth は、
`completed` official run が持つ `result-summary.json` と
その persisted run metadata を組み合わせた 1 件の ranking update とする。

ranking update を組み立てるときは、少なくとも次を使う。

- `result-summary.json`
  - `match_id`
  - `game_id`
  - `game_version`
  - `ruleset_version`
  - `status`
  - `placements[]`
- persisted run metadata
  - `run_id`
  - `match_id`
  - `attempt_count`
  - `official`
  - player 順序
  - `player_id`
  - `artifact_ref`

`record.json` や `history.json` は replay / audit の source-of-truth だが、
ranking update の既定入力にはしない。
ranking lifecycle は compact artifact から update を作り、
必要なときだけ recompute helper が run record と artifact locator を辿る。

## Eligible Run

ranking に寄与してよいのは、少なくとも次を満たす run だけとする。

- queue lifecycle が `completed` である
- `official` が true である
- `result-summary.json` が存在する
- `result-summary.status` が `completed` である
- `placements[]` が空でない

`failed` / `canceled` terminal は ranking input に含めない。
completed でも `official` でない rerun candidate は ranking input に含めない。

1 `match_id` に completed run が複数あっても、
同時に ranking へ寄与してよいのは `official_run_id` と一致する 1 件だけである。

## Retry / Rerun / Correction Rule

ranking lifecycle は execution fact と official adoption を分離して扱う。

- `retry`:
  failed run から新 run を append する
- `rerun`:
  completed run から new candidate run を append する
- `correction/promote`:
  same `match_id` の completed run のうち、
  どれを official とするかを切り替える

same `match_id` に official completed run が存在しない場合、
`retry` により成功した completed run は自動で official になってよい。
この場合、その run は ranking input に含まれる。

same `match_id` に official completed run が存在する場合、
`rerun` により成功した completed run は自動で ranking input に入ってはならない。
ranking へ入るのは `correction/promote` で official に切り替わった後だけである。

このため ranking aggregate は、
「completed run 全件の総和」ではなく
「各 `match_id` の official completed run だけの総和」でなければならない。

## Competitor Identity

first ranking aggregate の competitor key は `artifact_ref` とする。

理由:

- current run contract には `ai_submission_id` が残っていない
- `player_id` は match-local label であり、cross-match identity としては弱い
- `artifact_ref` は official completed run から安定して取り出せる admitted competitor identity である

snapshot は operator readability のために last seen `player_id` を持ってよいが、
cross-match aggregation key として使ってはならない。

## Aggregate Scope

durable ranking snapshot は少なくとも次の scope ごとに 1 件持つ。

- `game_id`
- `game_version`
- `ruleset_version`

first lifecycle では season / division / queue lane / owner ごとの分割を持たない。
同じ scope に属する official completed run は、
queue order ではなく official adoption 後の durable set として 1 本の aggregate へ畳み込む。

## Durable Snapshot

ranking snapshot は per-run compact view ではなく、
official completed run を順に適用した aggregate state として保持する。

snapshot は少なくとも次を含む:

- scope:
  `game_id`、`game_version`、`ruleset_version`
- applied update bookkeeping:
  `applied_run_ids[]`
  `applied_match_ids[]`
  `last_applied_run_id`
  `last_applied_match_id`
- aggregate totals:
  `completed_matches`
- entries[]:
  - `competitor_ref`
  - `last_player_id`
  - `matches_played`
  - `first_places`
  - `placement_counts`
    - place number ごとの出現回数
  - `last_run_id`
  - `last_match_id`

first snapshot は durable object store または local filesystem 上の
stable artifact として保存してよい。
provider ごとの差は read/write adapter に閉じ込め、
operator が共有する contract は snapshot payload と stable locator に留める。

## Update Rule

incremental update は、1 件の official ranking input を current snapshot に適用して次 snapshot を得る責務である。

- scope が異なる input は別 snapshot へ入れる
- same `run_id` を 2 回適用してはならない
- same `match_id` を 2 つの run として同時適用してはならない
- 1 placement はちょうど 1 competitor entry に反映される
- unseen `competitor_ref` は新規 entry として追加してよい
- `placement_counts[place]`、`matches_played`、`first_places` を更新する
- `completed_matches` は applied `match_id` 数と一致しなければならない

first lifecycle は game 固有 score を横断比較しない。
cross-game 共通 aggregate は placement histogram ベースに留める。

## Recompute Responsibility

ranking snapshot の durable 保存と、
recompute による再構築は別責務とする。

- durable snapshot path:
  official completed run が増えるか official selection が切り替わったときに current snapshot を更新して保存する
- recompute path:
  run record list と各 `result-summary.json` から same scope の snapshot を再構築する

recompute helper は少なくとも次を確認できなければならない。

- stored snapshot が durable official run set から再構築した snapshot と一致するか
- `applied_run_ids[]` と official completed run 集合が一致するか
- `applied_match_ids[]` と official completed match 集合が一致するか
- entry totals が placement 集計と一致するか

recompute helper は correction mutation route ではない。
first milestone では read/verify 用途に留め、
snapshot repair の最終手順は後続 plan で定義する。

current phase では、
official selection と ranking snapshot persist の間に
single-worker 前提の短い不整合窓を許容してよい。
この前提を外すときは
`docs/specs/platform-service-single-worker-assumptions.md`
を更新しながら atomicity を詰めなければならない。

## Ranking Read Surface

first ranking read surface は CLI と authenticated operator HTTP read の併用とする。

- CLI `get`:
  current durable snapshot を 1 scope 分返す
- CLI `recompute`:
  run record と compact summary から rebuilt snapshot を返す
- CLI `verify`:
  stored snapshot と rebuilt snapshot の一致可否を返す
- authenticated operator HTTP `GET`:
  `game_id`、`game_version`、`ruleset_version` で 1 scope を指定し、
  current durable snapshot を返す

HTTP read surface は current durable snapshot の参照だけを責務にする。
recompute や snapshot repair を browser から直接叩く route は
この milestone では持たない。

## Leaderboard Family への引き継ぎ境界

leaderboard / standings / history view は、この ranking snapshot の上に載る read-only family として扱う。

この spec で固定するのは:

- competitor identity
- scope
- official run の採用単位
- recompute 可能な aggregate payload

この spec で固定しないのは:

- どの sort key を leaderboard の primary order にするか
- tie-break presentation
- public / authenticated route shape
- long-term history retention UI

leaderboard family の plan は、ここで定義した snapshot を source-of-truth として読めばよく、
再び `record.json` 全件走査を既定 read path にしてはならない。
