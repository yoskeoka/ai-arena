# phase8-public-state-and-reversi-visualizer-platform-connection
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

ai-arena の `0126` public spectator resource と reversi-ai-arena の `0001` Reversi replay viewer を接続する後続 work を保持する。最終的には viewer が
versioned public resource を選択して terminal replay を再生し、running match の stale response を描画せず terminal 後に polling を
止める経路を扱う。

`N/A - detail required before execution`: これは intentional parent plan である。A/B の実装 PR が merge され、実際の public
fixture/envelope version、access policy、cache/retry hint、viewer bridge と polling evidence を確認するまで implementation を開始しない。
その時点の各 repo 最新 `main` から、exact route/client symbols、cross-repository version compatibility、test lanes を持つ新しい詳細
execution plan を作成する。この parent plan 自体は code/API mutation を行わない。

## 現時点の境界

- input は A の public match metadata、terminal replay payload、final exported snapshot と B の selected viewer adapter だけとする。
- running match は A の cadence/version semantics に従い、older response を version で破棄し、terminal lifecycle 後は polling を
  stop する。
- private artifact が unavailable/missing でも public contract だけで viewer が動作する。operator route、private locator、signed
  operator URL、Reversi-specific platform endpoint は使わない。
- ai-arena と reversi-ai-arena は独立 PR とし、未 merge branch を hidden dependency にしない。

## 詳細 plan 作成の入力と受入条件

- A/B の merged SHA、public replay format/version、fixture location、visibility/access matrix、cache/retry/stale semantics を
  evidence として引用する。
- network adapter の `(NEW)/(MODIFY)/(DELETE)` map、TypeSpec/client generation impact、viewer fetch/error UX、CORS/origin
  policy、fixture-based integration E2E を exact symbols とともに定義する。
- 同一 match の public metadata/replay/final snapshot/turn/result が shared fixture と一致し、old response を描画せず、terminal
  polling を停止し、private artifact request が発生しないことを black-box acceptance にする。

## 依存

ai-arena の `0126` と reversi-ai-arena の `0001` の実装完了が必須である。`0129` の stream evaluation は本 parent を詳細化・実装して polling evidence を得た後にのみ
開始できる。
