# dungeon reference AI は memory/world-model と policy を分けたい

## Summary

現状の dungeon reference AI は `games/dungeon/botlogic/botlogic.go` の `Bot` に
既知 tile/chest/goal の保持、観測更新、frontier 選択、最短経路探索、行動決定が
まとまっている。

- `Bot` が `knownTiles` / `knownGoal` / `knownChests` / `exploreTarget` を直接持つ
  (`games/dungeon/botlogic/botlogic.go:16-22`)
- `Decide` が observe, treasure/goal prioritization, frontier 探索, fallback wait を
  1 箇所で束ねている
  (`games/dungeon/botlogic/botlogic.go:93-117`)
- `observe`, `chooseChestAction`, `chooseFrontierTarget`, `shortestKnownPath` も同ファイルにあり、
  state update と action policy が分離されていない
  (`games/dungeon/botlogic/botlogic.go:119-260`)

Phase 5 の baseline としては十分だが、この形のまま敵・罠・不完全観測・
将来の inventory/consumable を足すと、「何を知っているか」と「どう動くか」が
同時に肥大化して変更コストが急に上がる。

## Why It Matters

- hidden information や uncertainty の扱いを入れ始めると、観測更新と policy の責務が衝突しやすい
- memory update を独立させておかないと、reference AI を複数戦略へ派生させるたびに
  観測処理を重複しやすい
- scenario catalog で途中局面から bot を回す場合も、world-model を明示的に扱える方が
  fixture 設計と debug がしやすい

## Proposed Solution

1. reference AI を少なくとも次の責務へ分ける
   - observation / memory update
   - world-model query / path utility
   - action policy
2. `balanced`, `goal-rush`, `treasure-biased` のような複数 policy を
   同一 memory/world-model 上へ載せ替えられる形にする
3. scenario test では policy 単体、memory update 単体、統合挙動をそれぞれ確認できるようにする

## Scope Boundary

- この issue は dungeon reference AI の将来拡張に向けた構造整理を扱う
- Phase 5 の baseline heuristic を直ちに高性能化することは目的にしない
- 多言語 bot support や learned policy の導入は対象外
