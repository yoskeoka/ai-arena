# Platform Service Persistence 仕様

## 目的

このドキュメントは、online service skeleton が queue lifecycle と terminal artifact locator を
durable に保持する write model contract を定義する。

この spec の対象は service orchestration が読む write path である。
operator-facing query API や replay / resume 用の read model 詳細は別 plan で扱う。

## この spec の責務範囲

この spec が定義するもの:

- 1 submission あたりに durable に保持する最小保存単位
- queue lifecycle と claim ownership の durable contract
- terminal artifact source-of-truth と locator summary の責務分離
- first infra target における metadata backend と artifact backend の分離

この spec が定義しないもの:

- operator-facing list/get/read API
- replay / resume / audit 用の派生 read model
- retention、archive、compaction policy
- distributed fairness、multi-region queue、dead-letter queue

## 参照関係

- `docs/specs/platform-service-skeleton.md`: submission / admission / queue lifecycle の正本
- `docs/specs/platform.md`: runner、artifact layout、platform core 責務の正本

## 保存単位

durable write model の最小保存単位は、1 回の `match submission` に対応する 1 lifecycle record とする。

1 lifecycle record には少なくとも次を保持する:

- submission identity:
  `submission_id`、`match_id`
- compatibility metadata:
  `game_id`、`game_version`、`ruleset_version`
- submitted players:
  player 順序、`player_id`、`artifact_ref`
- artifact base locator:
  `output_dir`
- orchestration lifecycle:
  `queued|leased|running|persisting|completed|failed|canceled`
- queue ordering identity:
  queued 順序を再現し、claim 順を安定させるための durable ordering key
- lease metadata:
  現在の worker identity
- terminal summary:
  `match_dir`、`record` locator、`result-summary` locator、player stderr locator 群、terminal match status、terminal error summary

`attempt_count` は record に含めてよいが、この milestone では `1` 固定のままとする。

## 状態遷移と durability

- `enqueue` は validation 済み submission を queued record として保存しなければならない
- `claim` は queued record のうち最も早いものを 1 worker だけへ lease しなければならない
- `claim` 後の worker identity は、以後の lifecycle update より前に durable に観測できなければならない
- `update` は `queued -> leased -> running -> persisting -> terminal` の許可遷移だけを保存する
- `cancel` は queued record に対してだけ許可し、cancel 後は worker claim できてはならない
- terminal artifact persist が成功したとき、write model は terminal artifact locator と terminal match summary を保持しなければならない

single logical queue authority の最初の到達点は明確に絞る。
複数 process が同じ backend を共有しても、
1 queued record が同時に複数 worker へ lease されないことだけをまず保証する。
multi-node fairness や retry redelivery はこの spec の対象外とする。

## Artifact Source-Of-Truth との境界

- `record.json`、`result-summary.json`、`snapshot.json`、`history.json`、stderr log、本体 executable payload は artifact backend の責務とする
- durable write model は、artifact 本体を複製せず locator と最小 terminal summary だけを保持する
- operator / replay / resume は後続 read model で locator を使って artifact backend を読む
- write model 単体で game 固有 world state の完全再構築責務を持たない

## Backend Split

first deploy target は `Cloudflare Pages + Render + Neon Postgres + Cloudflare R2` とする。

- `Neon Postgres`:
  submission metadata、queue ordering、lease metadata、terminal locator summary
- `Cloudflare R2`:
  `record.json`、`result-summary.json`、`snapshot.json`、`history.json`、stderr artifact、AI / game master payload 本体

local CLI と CI では、同じ Postgres write contract を Docker ベースの local database で検証してよい。
artifact lane は引き続き local filesystem を default としてよい。

## Schema Management Contract

- durable write model の desired schema は repo 内の SQL file 群を source of truth とする
- migration SQL は desired schema と current DB schema の差分から生成する review/apply artifact とし、
  migration file 自体を唯一の schema 定義として扱わない
- service runtime は startup 時に DDL を実行してはならない
- Postgres backend を使う service / worker / test harness は、
  起動前に target DB へ schema apply が完了していることを前提にする
- review/deploy lane は versioned migration SQL を生成して review 対象にし、
  test/bootstrap lane は空 DB へ desired schema を直接 apply して初期化してよい

## Query Artifact Boundary

- handwritten SQL schema と handwritten query SQL は人間が review する source artifact とする
- generated query code は typed adapter artifact とし、query source を置き換える責務を持たない
- durable queue backend の既存抽象境界は維持し、generated query code の採用は Postgres backend 内部に閉じ込める
- `0057` / `0058` 以降の operator/read path 拡張も、同じ schema source と query package 境界を再利用してよい

## Deferred Follow-Ups

- result list / match detail / locator read API
- replay / resume / audit input builder
- retention / archive policy
- multi-node retry / fairness / lease recovery policy
