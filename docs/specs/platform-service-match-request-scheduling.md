# Platform Service Match Request / Scheduling 仕様

## 目的

このドキュメントは、Phase 7 の general operator lane で扱う
`match request` entity と、single logical queue authority 前提の
最小 scheduling contract を定義する。

ここで固定するのは、operator がどの単位で対戦要求を作るか、
そして service が preset lane / general lane をどの policy で
同じ実行 queue へ流し込むかである。

## この spec の責務範囲

この spec が定義するもの:

- `match request` の最小 identity と participant shape
- general request が参照する `game registration` / `AI submission` の整合条件
- preset lane と general lane を同じ scheduling 入口へ正規化する責務
- first scheduling policy の選択規則
- request visibility の最小 read model

この spec が定義しないもの:

- ranking 更新
- rerun / retry / cancellation policy
- multi-worker fairness の最終設計
- tournament bracket や bulk matchmaking batch

## 参照関係

- `docs/specs/platform-service-general-submission.md`: request が参照する durable identity の正本
- `docs/specs/platform-service-skeleton.md`: queue へ入る single-match execution request の正本
- `docs/specs/platform-service-operator-api.md`: operator-facing route contract の正本

## Match Request

`match request` は、operator が「どの registered game で、どの admitted AI 同士を、
どの output target で 1 試合走らせるか」を表す request entity である。

この entity 自体は queue record ではない。
service は request を validation し、scheduling policy に従って
`match submission` へ具体化して queue へ渡す。

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
- `scheduled_submission_id`
- `scheduled_match_id`

request は作成後ただちに scheduling 対象になってよい。
first contract では、accepted request は必ず 1 件の
`scheduled_submission_id` と `scheduled_match_id` を持つ。

## Validation

request を受け付けるときは、少なくとも次を同期的に確認しなければならない。

- `game_registration_id` が存在すること
- participant が 1 件以上あること
- `player_id` が request 内で一意であること
- すべての `ai_submission_id` が存在すること
- すべての participant が同じ `game_registration_id` を参照していること
- participant の admitted game metadata が request の registered game metadata と矛盾しないこと
- `output_dir` が `match submission` の terminal persist target として受け入れ可能であること

validation 成功後に scheduling された request は、
participant の `artifact_ref` を使って 1 件の `match submission` へ具体化される。

## Preset Lane との関係

preset lane は bootstrap 用の shortcut だが、
general lane と別 queue policy を持ってはならない。

service は preset enqueue のとき、少なくとも次の順序で
general lane と同じ scheduling 入口へ正規化しなければならない。

1. preset definition から `game registration` を materialize する
2. preset participant ごとに `AI submission` を materialize する
3. materialized identity から `match request` を組み立てる
4. general request と同じ scheduling policy で `match submission` を queue へ流す

このため、preset queue は dedicated 実行 queue を持たず、
general request と同じ queue authority を共有する。

## First Scheduling Policy

first scheduling policy は single logical queue authority 前提の FIFO とする。

- scheduling の source-of-truth は service process が受け付けた request 順とする
- request は accepted 後、ただちに 1 件の `match submission` へ具体化して queue へ入れてよい
- queue claim 順は既存の execution queue policy に委ねてよい
- preset request と manual request は source による優先度差を持たない
- fairness、quota、parallel lane 分離、reservation は後続 plan へ送る

この milestone では、`match request` を「queue 前の operator-facing unit」として固定し、
queue 側は従来どおり single-match execution request を扱い続ける。

## Visibility

operator は accepted request を list できなければならない。

最小の visibility item は次を含む:

- `request_id`
- `game_registration_id`
- `game`
- `participants[]`
- `source`
- `source_id`
- `scheduled_submission_id`
- `scheduled_match_id`
- 現在観測できる queue lifecycle

queue lifecycle は request entity 自体の mutable state として永続化してもよいし、
request が参照する scheduled submission から read 時に導出してもよい。

## Deferred Follow-Ups

- pending / deferred request backlog
- request cancellation after scheduling
- per-game / per-owner quota
- scheduling fairness and multi-worker dispatch policy
