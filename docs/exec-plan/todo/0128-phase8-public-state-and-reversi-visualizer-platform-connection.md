# phase8-public-state-and-reversi-visualizer-platform-connection
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

anonymous public API を使う ai-arena の `0126` と、game provider が別 host で提供する reversi-ai-arena の `0001` reference viewer が
完了した後の access protection work を保持する。Phase 8 は anonymous read と public API 接続までを first delivery とし、本 plan は
その後に利用者 / external visualizer を保護する必要が生じた場合だけ詳細化する。

`N/A - detail required before execution`: これは intentional parent plan である。A/B の実装 PR が merge され、実際の public
fixture/envelope version、anonymous public access、cache/retry hint、external-hosted viewer bridge を確認するまで implementation を開始しない。
その時点の各 repo 最新 `main` から、access model、exact route/client symbols、cross-repository version compatibility、test lanes を持つ
新しい詳細 execution plan を作成する。この parent plan 自体は code/API mutation を行わない。

## 現時点の境界

- input は A の public match metadata、running latest exported-state resource、terminal replay metadata/payload、final exported snapshot
  と B の external-hosted viewer adapter / client behavior だけとする。
- protected access の必要性、対象 consumer（external web / native app / CLI / direct API）、migration policy、credential / redirect /
  token / origin policy は human review で決める。anonymous Phase 8 API と同じ endpoint に adhoc な session-cookie requirement を足さない。
- private artifact が unavailable/missing でも public contract だけで viewer が動作する。operator route、private locator、signed
  operator URL、Reversi-specific platform endpoint は使わない。
- ai-arena と reversi-ai-arena は独立 PR とし、未 merge branch を hidden dependency にしない。

## 詳細 plan 作成の入力と受入条件

- ai-arena A と reversi-ai-arena B の merged SHA、public replay format/version、fixture location、anonymous access matrix、
  running-state and cache/retry/stale semantics を
  evidence として引用する。
- `(NEW)/(MODIFY)/(DELETE)` map、TypeSpec/client generation impact、external viewer / native / CLI access UX、authorization grant と
  CORS/origin policy、anonymous-to-protected migration / rollback、fixture-based integration E2E を exact symbols とともに定義する。
- 同一 match の public metadata/replay/final snapshot/turn/result が shared fixture と一致し、authorized client は policy に従い、
  anonymous client は documented migration result を受け、private artifact request が発生しないことを black-box acceptance にする。

## 依存

ai-arena の `0126` と reversi-ai-arena の `0001` の実装完了が必須である。Phase 8 の完了は本 parent の詳細化・実装を待たない。event stream の
評価は A/B の anonymous polling evidence を直接入力にでき、本 plan の protection 導入を待たない。
