# dungeon は手作り snapshot 起点の scenario catalog が必要

## Summary

現状の dungeon 検証は、fixed map / seeded generation をそのまま流す e2e と、
局所的な domain test の組み合わせが中心で、特定 mechanic を狙って再現する
handcrafted scenario が不足している。

- `arena_runner_test.go` の dungeon e2e は local subprocess bot を 2 体流して
  完走と artifact 出力を確認する形で、局面を細かく固定した検証にはなっていない
  (`e2e/arena_runner_test.go:105-126`)
- runner には `--snapshot-input` / `--history-input` があり、途中状態から再開する経路は
  すでに存在する
  (`e2e/arena_runner_test.go:294-360`, `e2e/arena_runner_test.go:363-405`)
- dungeon bot は現状 1 つの deterministic heuristic decision layer を持つが、
  これは「ゲームを回す reference」と「特定局面を狙って再現する fixture」の役割を兼ねていない
  (`games/dungeon/botlogic/botlogic.go:16-117`)

seed 固定のランダムマップだけでは、将来の敵・罠・消費アイテム・可視性ルール変更ごとに
「再現したい局面」を毎回 generation 側へ埋め込むことになり、テストデータの管理が重くなる。

## Why It Matters

- 機能追加ごとに「何ターン目に何を確認したいか」が変わるため、random 生成ベースだけでは
  局面の再現性と読みやすさが不足する
- seed replay は end-to-end の回帰確認には向くが、feature 単位の失敗原因を狭く切り分けにくい
- 途中ターンの selected field を確認できる scenario catalog があれば、
  long match 全体を golden 化せずに mechanic 単位で安定検証できる

## Proposed Solution

1. `games/dungeon` 向けに scenario catalog を導入する
   - 1 scenario 1 intent で、例: 同時 chest 取得、goal race、視界外からの再発見、
     dead-end 探索、残りターンぎりぎり到達など
2. handcrafted snapshot / state builder を用意し、
   random generation を経由せずに狙った局面を作れるようにする
3. scenario ごとに「何ターン回すか」と「どの中間/最終 field を見るか」を固定する
   - full `record.json` golden ではなく compact assertion を基本にする
4. fixed-seed reference AI league regression は残しつつ、
   correctness gate は scenario catalog 側へ寄せる

## Scope Boundary

- この issue は dungeon feature expansion を支える targeted test data 戦略を扱う
- reference AI の強さ評価そのものは主題ではない
- full exported snapshot golden の全面採用は前提にしない
