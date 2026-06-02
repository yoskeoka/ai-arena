# operator-ui-component-state-refactor

## Summary

`operator-ui/src/App.tsx` は、minimal landing の範囲で一気に
active/completed polling、detail polling、preset enqueue、connection hint、
panel rendering、table rendering、badge/meta helper まで抱える形になっている。
現在でも 500 行を超えており、今後 ranking/filter/view 追加が入る前に
frontend の責務分割方針を定めたうえでリファクタできる余地が大きい。

ただし現時点では UI surface 自体がまだ増える途中であり、
local/CI の e2e/integration verification も未整備であるため、
すぐに分解だけ先行すると回帰検知が弱いまま構造だけ動かすことになりやすい。

## Context

- current file:
  `operator-ui/src/App.tsx`
- related verification follow-up:
  `docs/issues/0025-operator-ui-local-and-ci-verification.md`

## Impact

- state、polling side effect、presentation が 1 file に混在し、
  小さな UI 変更でも diff scope が広がりやすい
- local connection handling や future fetch policy を触ると、
  unrelated view rendering まで同じ file でレビューする必要がある
- 今後 panel や action が増えるほど component boundary と test seam が作りにくくなる

## Proposed Solution

- UI verification lane がある程度整った時点で、operator UI の責務分割方針を先に決める
- 少なくとも次の単位を候補にする
  - data/polling state hook
  - preset queue panel
  - active/completed match list panel
  - completed detail panel
  - shared UI primitives
- fetch/polling policy と local connection normalization は view component から分離する
- refactor の開始条件として、最低限の local/CI e2e or integration coverage を
  `0025` で先に用意する

## Priority

中。
今すぐの blocker ではないが、機能が増える前か、e2e が整った直後に着手したい。
