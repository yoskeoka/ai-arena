# Platform Service Single-Worker Assumptions 仕様

## 目的

このドキュメントは、Phase 7 時点の platform service が
single-worker / single logical queue authority 前提で成立している箇所を
明示的に記録する。

目的は 2 つある。

- current implementation がどの non-atomic path を許容しているかを後から再調査しなくてよいようにする
- multi-worker 化を別 plan で進めるとき、どこを transactional / atomic に置き換える必要があるかを最初から共有する

## この spec の責務範囲

この spec が定義するもの:

- single-worker 前提で許容している設計箇所
- その前提が破られたときの race / partial-update risk
- deferred multi-worker fix の記録方法

この spec が定義しないもの:

- multi-worker 最終アーキテクチャ
- distributed lock / DB constraint の具体実装
- fairness / throughput policy

## 基本前提

current phase の platform service は、
同じ logical queue authority を同時に mutate する worker / operator mutation actor が
1 系統だけであることを前提にしてよい。

ここでいう mutation には少なくとも次を含む。

- queued run claim
- completed run の official 判定
- `promote` による official run 切り替え
- official run に追従する ranking snapshot 更新

この前提の下では、
「read current state -> decide -> write next state」という複数段の処理を
single process 内の逐次操作として扱ってよい。

## Current Single-Worker Assumption Sites

### Official auto-promote on worker completion

completed run の auto-promote 判定は、
same `match_id` の run 群を読んで
すでに official completed run があるかを見てから
current run の `official` を決める。

current phase では、この read と write は atomic でなくてよい。

single-worker 前提が破られると、少なくとも次の race が起こりうる。

- 2 worker が同じ `match_id` 配下の別 run をほぼ同時に completed へ進める
- 双方が「まだ official がない」と観測する
- 複数 run が `official=true` になる

### Promote / correction over one match run group

`promote` は same `match_id` に属する run 群を列挙し、
target run だけ `official=true`、
他を `official=false` に更新する。

current phase では、この group update は transaction で一括更新しなくてよい。

single-worker 前提が破られると、少なくとも次の partial-update risk がある。

- loop の途中で update failure が起こる
- concurrent `promote` が別 worker / process から走る
- 一時的または永続的に official run が 0 件または複数件になる

### Official selection and ranking snapshot refresh

official run の切り替えと ranking snapshot refresh は、
同一 transaction で commit されなくてよい。

current phase では、
official selection の durable write と ranking recompute / snapshot persist の間に
短い不整合窓があってよい。

single-worker 前提が破られると、少なくとも次の状態が起こりうる。

- queue record 上の `official` は新しい値になったが ranking snapshot は旧状態
- concurrent recompute / verify が中間状態を読む

## Required Behavior Under Current Assumption

single-worker 前提でも、次は守らなければならない。

- non-atomic path は spec 上で明示する
- terminal run を ranking-side error だけで `failed` へ巻き戻してはならない
- official selection の source-of-truth は durable queue/run metadata に置く
- multi-worker 前提に変える plan では、この spec を更新しながら対象箇所を減らす

## Tracking Rule

今後、single-worker 前提でしか安全でない path を見つけたら、
「あとで全体監査する」ではなく、
見つけた時点でこの spec に追加しなければならない。

記録するときは少なくとも次を書く。

- どの mutation path か
- なぜ non-atomic か
- multi-worker で何が壊れるか
- 今 phase で defer してよい理由

## 参照関係

- `docs/specs/platform-service-match-request-scheduling.md`
- `docs/specs/platform-service-ranking-lifecycle.md`
- `docs/specs/platform-service-operator-api.md`
