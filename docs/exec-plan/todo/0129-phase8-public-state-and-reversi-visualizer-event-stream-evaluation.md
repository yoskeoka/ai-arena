# phase8-public-state-and-reversi-visualizer-event-stream-evaluation
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

snapshot polling が Phase 8 spectator latency、load、reconnect needs を満たすかを評価し、必要な場合だけ public exported-state
family の追加 transport として event stream を計画する。stream を加えない結論も、有効な完了結果である。

`N/A - detail required before execution`: これは intentional parent plan である。`0128` の public viewer connection が merge され、
polling cadence、cache/retry、viewer-side stale discard、terminal stop、provider load/latency/reconnect evidence が得られるまで
implementation を開始しない。その evidence を入力に、各 repo 最新 `main` から transport choice と exact protocol contract を含む
新しい詳細 execution plan を作成する。この parent plan 自体は code/API mutation を行わない。

## 現時点の境界

- event stream は public exported-state family の optional transport であり、`record.json.event_log`、private snapshot、history、
  stderr、bundle bytes を直接公開するものではない。
- polling を維持する場合は、spectator latency/load/reconnect requirement を満たす測定結果と、stream を再評価する threshold を
  `docs/project-plan.md` または relevant spec に残す。
- stream を追加する場合だけ、transport（SSE/WebSocket 等）、cursor、ordering、deduplication、retention、terminal close、
  reconnect、backpressure、auth/visibility を TypeSpec/behavioral spec と black-box tests で定義する。
- Reversi 固有 endpoint や viewer-only protocol を増やさず、game-specific state は A の opaque public envelope 内に閉じ込める。

## 詳細 plan 作成の入力と受入条件

- ai-arena public adapter と reversi-ai-arena viewer adapter の両方の C merged SHA、shared fixture/contract version、polling observability evidence から、latency percentile、concurrent spectator/load profile、reconnect/error
  profile、stale/terminal behavior、provider cost/operational constraints を引用する。
- polling 継続または stream 導入の比較、採否、再評価条件を human review で決め、wire source、`(NEW)/(MODIFY)/(DELETE)` map、
  cross-repository client impact を exact symbols とともに記録する。
- stream を導入するなら、public state version ordering、cursor/reconnect、terminal close、slow consumer/backpressure、retention と
  private-boundary negative test を acceptance にする。導入しないなら、polling evidence と threshold が durable documentation に
  残ることを acceptance にする。

## 依存

`0128` の詳細 plan と実装による polling evidence が必須である。Phase 8 の first delivery は polling であり、本 parent の存在は
stream 実装の承認を意味しない。
