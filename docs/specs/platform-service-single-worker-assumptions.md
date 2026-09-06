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

## Phase 7 運用トポロジ

Phase 7 の最小 deploy unit は、HTTP service と同居する 1 worker process とする。
metadata / queue authority は Neon Postgres、artifact payload は R2、operator UI は Pages が担う。
Pages には R2 credential を配布してはならず、artifact access は service が発行する限定的な
download metadata を通す。worker を別 deploy unit に分離すること、複数 worker、distributed
fairness はこの phase の範囲外である。

同じ Postgres queue を使う service は 1 worker だけを起動しなければならない。起動時に
既存の生存 worker が観測される構成を許可する場合は fail closed とし、operator が lease expiry
を待つか明示的に復旧させるまで、2 本目の worker が run を実行してはならない。

## Lease、回復、shutdown

worker が claim した run には worker identity、lease deadline、最後の heartbeat を durable に
記録する。worker は run の実行中も heartbeat を更新し、queue lag と worker heartbeat age は
operator が観測できなければならない。

lease deadline を過ぎた `leased`、`running`、`persisting` run は startup と通常 poll の前に
recovery 対象となる。recovery は run を `queued` に戻して lease owner/deadline を消去し、同じ
submission snapshot を再実行する。terminal record、未期限 lease、または cancel 済み record を
変更してはならない。これにより process crash 後の run が永久に in-flight で残らない。

shutdown は新規 claim を止め、HTTP server の drain と現在の run の context cancellation を
bounded deadline 内で行う。deadline 内に終了できない run は lease expiry recovery に委ねる。
graceful shutdown の成功を、terminal artifact の完成を伴わない `completed` として記録してはならない。

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
- `docs/specs/index.md`
