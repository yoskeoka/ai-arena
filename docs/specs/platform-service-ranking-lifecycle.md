# Platform Service Ranking Lifecycle 仕様

## 目的

このドキュメントは、Phase 7 の online operator cycle で扱う
ranking aggregate の最小 lifecycle を定義する。

ここで固定するのは、どの completed match を ranking input とみなし、
どの単位で durable snapshot を更新し、
どこまでを recompute 可能な read model として持つかである。

## この spec の責務範囲

この spec が定義するもの:

- ranking input の source-of-truth
- first ranking aggregate の集計単位と competitor identity
- durable ranking snapshot の最小 field
- incremental update と full recompute の責務分離
- CLI-first の ranking read / verify helper surface
- leaderboard family への引き継ぎ境界

この spec が定義しないもの:

- public leaderboard UI
- season / division / reward system
- rerun / retry / cancellation による ranking correction の最終規則
- game 固有 score を使う独自順位式
- public HTTP ranking API

## 参照関係

- `docs/specs/platform-service-read-model.md`: per-match read model の正本
- `docs/specs/platform-service-match-request-scheduling.md`: ranking 前段の request / scheduling 単位の正本
- `docs/specs/platform-service-general-submission.md`: admitted AI identity の正本
- `docs/specs/platform.md`: `result-summary.json` を含む terminal artifact contract の正本

## Ranking Input

first ranking lifecycle の source-of-truth は、
`completed` submission が持つ `result-summary.json` と
その submission metadata を組み合わせた 1 件の ranking update とする。

ranking update を組み立てるときは、少なくとも次を使う。

- `result-summary.json`
  - `match_id`
  - `game_id`
  - `game_version`
  - `ruleset_version`
  - `status`
  - `placements[]`
- persisted submission metadata
  - `submission_id`
  - player 順序
  - `player_id`
  - `artifact_ref`
  - `attempt_count`

`record.json` や `history.json` は replay / audit の source-of-truth だが、
ranking update の既定入力にはしない。
ranking lifecycle は compact artifact から update を作り、
必要なときだけ recompute helper が queue record と artifact locator を辿る。

## Eligible Match

ranking に寄与してよいのは、少なくとも次を満たす submission だけとする。

- queue lifecycle が `completed` である
- `result-summary.json` が存在する
- `result-summary.status` が `completed` である
- `placements[]` が空でない

`failed` / `canceled` terminal は ranking input に含めない。
execution retry / rerun / operator correction の扱いは後続 plan で定義する。
この milestone では、1 `submission_id` は at-most-once で ranking aggregate に適用される。

## Competitor Identity

first ranking aggregate の competitor key は `artifact_ref` とする。

理由:

- current queue submission / terminal artifact contract には `ai_submission_id` が残っていない
- `player_id` は match-local label であり、cross-match identity としては弱い
- `artifact_ref` は current completed submission から安定して取り出せる admitted competitor identity である

snapshot は operator readability のために last seen `player_id` を持ってよいが、
cross-match aggregation key として使ってはならない。

後続で durable match record が `ai_submission_id` を直接保持するようになれば、
competitor key を差し替える migration を別 plan で扱う。

## Aggregate Scope

durable ranking snapshot は少なくとも次の scope ごとに 1 件持つ。

- `game_id`
- `game_version`
- `ruleset_version`

first lifecycle では season / division / queue lane / owner ごとの分割を持たない。
同じ scope に属する completed submission は、queue order に従って 1 本の aggregate へ畳み込む。

## Durable Snapshot

ranking snapshot は per-match compact view ではなく、
completed submissions を順に適用した aggregate state として保持する。

snapshot は少なくとも次を含む:

- scope:
  `game_id`、`game_version`、`ruleset_version`
- applied update bookkeeping:
  `applied_submission_ids[]`
  `last_applied_submission_id`
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
  - `last_submission_id`
  - `last_match_id`

first snapshot は durable object store または local filesystem 上の
stable artifact として保存してよい。
provider ごとの差は read/write adapter に閉じ込め、
operator が共有する contract は snapshot payload と stable locator に留める。

## Update Rule

incremental update は、1 件の ranking input を current snapshot に適用して次 snapshot を得る責務である。

- scope が異なる input は別 snapshot へ入れる
- same `submission_id` を 2 回適用してはならない
- 1 placement はちょうど 1 competitor entry に反映される
- unseen `competitor_ref` は新規 entry として追加してよい
- `placement_counts[place]`、`matches_played`、`first_places` を更新する
- `completed_matches` は applied submission 数と一致しなければならない

first lifecycle は game 固有 score を横断比較しない。
cross-game 共通 aggregate は placement histogram ベースに留める。

## Recompute Responsibility

ranking snapshot の durable 保存と、
recompute による再構築は別責務とする。

- durable snapshot path:
  queue が `completed` へ進む前に current snapshot を更新して保存する
- recompute path:
  queue record list と各 `result-summary.json` から same scope の snapshot を再構築する

recompute helper は少なくとも次を確認できなければならない。

- stored snapshot が queue/source-of-truth から再構築した snapshot と一致するか
- `applied_submission_ids[]` と completed submission 集合が一致するか
- entry totals が placement 集計と一致するか

recompute helper は operator correction の mutation route ではない。
first milestone では read/verify 用途に留め、
snapshot repair の最終手順は後続 plan で定義する。

## Ranking Read Surface

first ranking read surface は CLI-first とする。

- `get`:
  current durable snapshot を 1 scope 分返す
- `recompute`:
  queue record と compact summary から rebuilt snapshot を返す
- `verify`:
  stored snapshot と rebuilt snapshot の一致可否を返す

この surface は ranking aggregate 自体を返す。
public HTTP API、operator UI、pagination、history diff viewer は後続へ送る。

## Leaderboard Family への引き継ぎ境界

leaderboard / standings / history view は、この ranking snapshot の上に載る read-only family として扱う。

この spec で固定するのは:

- competitor identity
- scope
- incremental update の単位
- recompute 可能な aggregate payload

この spec で固定しないのは:

- どの sort key を leaderboard の primary order にするか
- tie-break presentation
- public / authenticated route shape
- long-term history retention UI

leaderboard family の plan は、ここで定義した snapshot を source-of-truth として読めばよく、
再び `record.json` 全件走査を既定 read path にしてはならない。
