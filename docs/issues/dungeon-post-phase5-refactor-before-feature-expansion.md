# dungeon は Phase 5 完了後に feature expansion 前提の refactor を入れたい

## Summary

`platform-phase5-01-dungeon-fixed-map-mvp` と
`platform-phase5-02-dungeon-seeded-generation` の流れで、`games/dungeon/dungeon.go` に
ゲーム state 型、snapshot 復元、turn progression、score 計算、discovery/fog、
pathfinding、map/ruleset generation、RNG まわりが集まってきた。

現状の実装は Phase 5 を完了するうえでは許容できるが、そのまま敵・罠・消費アイテム・
複数種の tile effect・複数 floor のような game 要素を足し始めると、turn 処理と
layout/generation 責務が 1 箇所に積み上がって変更コストが急に上がる。

このため、次の game 要素追加や game direction の検討に入る前に、
「Phase 5 完了後の整理」として dungeon 実装を分解する issue を記録しておく。

## Why It Matters

- `Apply` 相当の turn progression に要素追加の責務が集中しやすい
- `Ruleset` が static rule と generated layout の両方を抱えており、拡張点を分けにくい
- snapshot/public state/visible state と internal mutation helper が同じファイルにあり、変更影響を追いにくい
- deterministic replay を守るための順序制約と、game rule の追加が同じ場所で絡みやすい

## Follow-up Direction

Phase 5 完了後、次の game 要素追加や方向性検討と合わせて、少なくとも以下を見直す。

1. `games/dungeon` の責務分割
   - 例: contract types、match progression、layout generation、grid/pathfinding helper
2. turn progression の phase 分離
   - 例: movement、interaction、scoring、visibility refresh
3. rule config と generated layout/state の分離
   - match ごとに生成される data と、ruleset definition を分ける
4. deterministic rule の明文化
   - 単一 RNG stream、逐次処理、`map` iteration 順など runtime 依存の非決定要素を持ち込まない

## Scope Boundary

- この issue は「次の feature expansion 前に整理した方がよい実装負債」を記録するもの
- Phase 5 の active execution scope には含めない
- 実際の refactor 方針は、Phase 5 完了後に game design の検討と一緒に決める
