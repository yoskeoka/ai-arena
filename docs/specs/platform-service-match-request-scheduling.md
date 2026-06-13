# Platform Service Match Request / Scheduling 仕様

## 目的

このドキュメントは、Phase 7 の general operator lane で扱う
`match request` entity と、single logical queue authority 前提の
最小 scheduling contract を定義する。

ここで固定するのは、operator がどの単位で対戦要求を作るか、
そして service が preset lane / general lane という異なる operator entry を
どの policy で同じ実行 queue へ正規化するかである。

## この spec の責務範囲

この spec が定義するもの:

- `match request` の最小 identity と participant shape
- logical `match_id` と per-attempt `run_id` の責務分離
- general request が参照する `game registration` / `AI submission` の整合条件
- preset lane と general lane を同じ scheduling 入口と queue authority へ正規化する責務
- first scheduling policy の選択規則
- retry / rerun / correction の match-run-group 境界
- request visibility の最小 read model

この spec が定義しないもの:

- ranking 集計式の詳細
- public dispute workflow
- self-service user appeal flow
- multi-worker fairness の最終設計
- tournament bracket や bulk matchmaking batch

## 参照関係

- `docs/specs/platform-service-general-submission.md`: request が参照する durable identity の正本
- `docs/specs/platform-service-skeleton.md`: queue へ入る single-run execution request の正本
- `docs/specs/platform-service-operator-api.md`: operator-facing route contract の正本
- `docs/specs/platform-service-ranking-lifecycle.md`: official run と ranking correction の正本

## Match Request

`match request` は、operator が「どの registered game で、どの admitted AI 同士を、
どの output target で 1 論理対戦を走らせるか」を表す request entity である。

この entity 自体は queue record ではない。
service は request を validation し、scheduling policy に従って
1 件の initial run を queue へ渡す。

### 最小項目

- `request_id`
- `game_registration_id`
- `game`
  - request 作成時点の registered game metadata snapshot
- `participants[]`
  - `player_id`
  - `ai_submission_id`
- `output_dir`
- `source`
  - `manual`
  - `preset`
- `source_id`
  - preset 由来なら `preset_id`
- `match_id`
  - logical match identity
- `latest_run_id`
- `official_run_id`

request は作成後ただちに scheduling 対象になってよい。
accepted request は、作成直後から少なくとも 1 件の run を持たなければならない。
first run は `match_id` に属する attempt `1` として queue へ入る。

## Match ID And Run ID

`match_id` は logical match identity であり、
operator が「同じ対戦のやり直し」とみなす run 群を束ねる。

`run_id` は 1 回の実行 attempt identity であり、
queue record、artifact persist、detail view、retry/rerun target の最小単位になる。

1 `match_id` に対して複数の `run_id` が属してよい。
この集合を `match run group` と呼ぶ。

first lifecycle では、少なくとも次を満たさなければならない。

- 1 `run_id` はちょうど 1 `match_id` に属する
- 1 `match_id` の run は `attempt_count` 昇順に並べられる
- `official_run_id` は 0 件または 1 件だけ持てる
- `official_run_id` を持つ場合、その run は `completed` でなければならない
- completed / failed run を in-place で `queued` へ戻して再利用してはならない

queue に入る write model は `run_id` 単位で append-only に保存する。

## Validation

request を受け付けるときは、少なくとも次を同期的に確認しなければならない。

- `game_registration_id` が存在すること
- participant が 1 件以上あること
- `player_id` が request 内で一意であること
- すべての `ai_submission_id` が存在すること
- すべての participant が同じ `game_registration_id` を参照していること
- participant の admitted game metadata が request の registered game metadata と矛盾しないこと
- `output_dir` が run の terminal persist target として受け入れ可能であること

validation 成功後に scheduling された request は、
participant の `artifact_ref` を使って 1 件の initial run へ具体化される。

## Preset Lane との関係

preset lane は bootstrap 用の shortcut だが、
general lane と別 queue policy や dedicated queue を持ってはならない。

service は preset enqueue のとき、少なくとも次の順序で
general lane と同じ scheduling 入口へ正規化しなければならない。

1. preset definition から `game registration` を materialize する
2. preset participant ごとに `AI submission` を materialize する
3. materialized identity から `match request` を組み立てる
4. general request と同じ scheduling policy で initial run を queue へ流す

このため、preset queue は queue implementation を別に持つのではなく、
general request と同じ single logical queue authority を共有する。
違いは queue 前の operator entry にあり、queue へ入る時点では
どちらも同じ logical `match_id` + first `run_id` contract に正規化される。

## First Scheduling Policy

first scheduling policy は single logical queue authority 前提の FIFO とする。

- scheduling の source-of-truth は service process が受け付けた request 順とする
- request は accepted 後、ただちに 1 件の initial run へ具体化して queue へ入れてよい
- queue claim 順は既存の execution queue policy に委ねてよい
- preset request と manual request は source による優先度差を持たない
- fairness、quota、parallel lane 分離、reservation は後続 plan へ送る

current durable queue backend は lane ごとの分離列を持たず、
scheduled 後の `run` だけを保存してよい。
`source` や `source_id` の違いを durable に持つ必要がある場合は、
queue record ではなく request visibility 側で扱ってよい。

## Retry / Rerun / Correction

request 作成後の follow-up 操作は、initial scheduling とは別 lifecycle として扱う。

- `queued cancel`:
  latest queued run だけを `canceled` へ進める
- `retry`:
  `failed` run だけを対象に、same `match_id` へ新しい `run_id` を append する
- `rerun`:
  `completed` run だけを対象に、same `match_id` へ新しい `run_id` を append する
- `correction/promote`:
  same `match_id` に属する completed run のうち、
  どれを `official_run_id` とみなすかを切り替える

`retry` と `rerun` は同じ enqueue mechanics を共有してよいが、
operator-facing intent を混同してはならない。

- `retry` は execution failure recovery である
- `rerun` は operator judgment / platform bug investigation / correction candidate 生成である

`retry` が `completed` を対象にしてはならない。
`rerun` が `failed` を対象にしてはならない。

same `match_id` に already completed な official run が存在しない場合、
`retry` により成功した completed run は自動で `official_run_id` になってよい。

same `match_id` に already completed な official run が存在する場合、
`rerun` により成功した completed run は自動で official になってはならない。
official の切り替えは `correction/promote` の明示操作でだけ行う。

## Visibility

operator は accepted request を list できなければならない。

最小の visibility item は次を含む:

- `request_id`
- `game_registration_id`
- `game`
- `participants[]`
- `source`
- `source_id`
- `match_id`
- `latest_run_id`
- `official_run_id`
- 現在観測できる latest run の queue lifecycle

request visibility は request entity 自体を永続化してもよいし、
request metadata と queue run 群から read 時に導出してもよい。

`official_run_id` は durable source-of-truth を持たなければならない。
process-local request cache だけに置いてはならない。

## Deferred Follow-Ups

- pending / deferred request backlog
- per-game / per-owner quota
- scheduling fairness and multi-worker dispatch policy
