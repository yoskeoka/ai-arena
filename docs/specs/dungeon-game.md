# ダンジョンゲーム MVP 仕様

## 目的

このドキュメントは、Phase 5 の最初の実装単位として導入する固定マップ dungeon MVP の
ゲーム固有仕様を定義する。platform 共通契約の正本は
`docs/specs/platform-common-contract.md`、game master 開発境界の正本は
`docs/specs/game-master.md` とし、この文書では dungeon game 固有の
payload / validation / state progression / scoring を固定する。

この MVP は以下を成立させるための最小単位である。

- top-level non-internal package tree に閉じた dungeon domain
- local subprocess game master での deterministic match loop
- local subprocess reference bot による高速検証
- 後続の seed 付き map 生成や WASM reference AI へ拡張できる payload 境界

## ゲーム ID とルールセット

- `game_id`: `dungeon`
- `game_version`: `1.0.0`
- `ruleset_version`: `fixed-map-v1`

`game_version` は payload schema と game master 実装の互換性を表す。`ruleset_version` は
固定マップ、採点、視界半径、制限ターン数をまとめた運営ルール識別子である。

## MVP の範囲

- 1 ステージ固定マップ
- 2 から 4 プレイヤー
- 同時行動
- アクションは `move` と `wait` のみ
- スコア源はゴール到達順位ボーナスと宝箱のみ
- プレイヤー同士の同居は許可
- 宝箱の同時取得は等分
- 視界制限は半径ベースの局所視界

この MVP では以下を扱わない。

- 自動マップ生成
- モンスター、戦闘、直接妨害
- 複数フロア進行
- inventory、item use、trap
- 正式 WASM reference AI

## 固定マップ

`fixed-map-v1` は以下の 9x9 マップを使う。`#` は壁、`.` は床、`A` と `B` は初期配置例、
`C` は宝箱、`G` はゴールを表す。

```text
#########
#A..#..B#
#...#...#
#.C...C.#
#.#.#.#.#
#...#...#
#.C...G.#
#.......#
#########
```

- 実装上の正本 map id は `fixed-map-v1`
- 初期配置は ruleset が持つ spawn list の先頭から player 順に割り当てる
- 余った spawn は未使用でよい
- 観戦用 exported snapshot では全体マップを公開してよい
- player 向け `visible_state` では現在視界内タイルだけを渡す

## ターン進行

- 1 match の最大ターン数は 16
- 各ターンで未ゴール player 全員に同時 request を送る
- 各 player は 1 アクションだけ返す
- game master は全 player の action status を集約後に一括で解決する
- ゴール済み player は後続ターンの request 対象から外す

試合終了条件:

- 全 player がゴール到達した
- または 16 ターンを消化した

これにより、全員ゴールで早期終了できる一方、最初の到達者だけで即終了して
順位ボーナス情報が潰れることは避ける。

## アクション schema

### `move`

```json
{
  "action": "move",
  "direction": "up"
}
```

- `direction` は `up` / `down` / `left` / `right`
- 移動先が壁またはマップ外なら invalid action とし、`no_action` と同じ扱いでその場に留まる

### `wait`

```json
{
  "action": "wait"
}
```

- その場に留まる

## 同時行動の解決規則

1. 各 player の `move` / `wait` を同時に解釈する
2. 全 player の移動先を確定する
3. player 位置を一括更新する
4. 宝箱取得を解決する
5. ゴール到達 turn を記録する

この MVP の競合解決は以下とする。

- player 同士は同じ tile に同時存在できる
- すれ違いも許可する
- body block は導入しない
- 宝箱 tile に同じ turn で複数 player が到達した場合、その宝箱点を等分する
- 宝箱はその turn の解決完了後に消滅する

## 視界制限

- 視界半径は `2`
- 視界形状は Manhattan distance ベース
- 半径外の tile は `visible_state.visible_tiles` に含めない
- 過去に見た通常床や壁の保持は AI 側責務とする
- ただし、ゴールと未取得宝箱の発見状況は `known_goal` / `known_chests` として
  game master 側でも保持してよい

この設計により、AI は完全マップを直接受け取らず、それでも landmark の再認識に
毎回の全探索を強いられない。

## スコア設計

MVP の採点は「主目的が常に強いが、寄り道最適化にも意味がある」ことを優先する。
一般的な race design の原則として、主目的の達成は副目的の総量より強くし、
副目的は最短経路の微修正や競合判断を生む重みへ抑える。

この ruleset では以下を採用する。

- ゴール順位ボーナス:
  - 1 位: 100
  - 2 位: 50
  - 3 位: 25
  - 4 位: 10
- 宝箱点:
  - 1 個あたり 12
  - 同時取得時は到達 player で等分

この配点により:

- 全宝箱合計は 36 点で、2 位が宝箱を独占しても 1 位単独到達を上回らない
- ただし 2 位同士や未ゴール同士では宝箱が十分に順位差を作る
- ルート上の寄り道判断が残り、単純最短経路固定にはなりにくい

## ゴール順位ボーナス規則

- ゴール順位は `finished_turn` の昇順で決める
- 同じ turn に到達した player は同順位
- 同順位は competition ranking とする
- 例: 2 人が同率 1 位なら両者に 100 点を与え、次の到達者は 3 位点 25 を得る
- ゴール未到達 player は順位ボーナス 0 点

最終順位は `goal_bonus + chest_points` の合計スコア降順で決め、こちらも
competition ranking を使う。

## `visible_state`

各 `turn` request の `visible_state` は少なくとも以下を持つ。

```json
{
  "turn": 4,
  "remaining_turns": 12,
  "view_radius": 2,
  "self": {
    "player_id": "p1",
    "x": 2,
    "y": 3,
    "score": 12,
    "goal_bonus": 0,
    "chest_points": 12,
    "finished_turn": null
  },
  "visible_tiles": [
    { "x": 1, "y": 3, "tile": "floor" },
    { "x": 2, "y": 3, "tile": "chest" }
  ],
  "known_goal": { "x": 6, "y": 6 },
  "known_chests": [
    { "x": 2, "y": 3 }
  ],
  "scores": [
    { "player_id": "p1", "score": 12, "goal_bonus": 0, "chest_points": 12, "finished_turn": null },
    { "player_id": "p2", "score": 0, "goal_bonus": 0, "chest_points": 0, "finished_turn": null }
  ]
}
```

フィールド要件:

- `turn`: 次に解決する turn 番号
- `remaining_turns`: `max_turns - turn + 1`
- `self`: 自 player の位置と累積得点
- `visible_tiles`: 現在視界内だけの tile list
- `known_goal`: この player が一度でも視認したゴール位置。未発見なら `null`
- `known_chests`: この player が把握している未取得宝箱位置
- `scores`: 全 player の公開スコア状況

`legal_action_hint` は `move` / `wait` の object schema を返す。

## `full_state`

`snapshot.game_state` に入る `full_state` は少なくとも以下を持つ。

```json
{
  "map_id": "fixed-map-v1",
  "rng_seed": 0,
  "turn": 4,
  "max_turns": 16,
  "goal": { "x": 6, "y": 6 },
  "players": [
    { "player_id": "p1", "x": 2, "y": 3, "score": 12, "goal_bonus": 0, "chest_points": 12, "finished_turn": null }
  ],
  "uncollected_chests": [
    { "x": 6, "y": 3 }
  ],
  "discovery": {
    "p1": {
      "known_goal": { "x": 6, "y": 6 },
      "known_chests": [{ "x": 6, "y": 3 }]
    }
  }
}
```

`full_state` は resume source of truth なので、少なくとも以下を復元できなければならない。

- マップ識別子
- `rng_seed`
- 現在 turn
- 各 player の位置、得点、ゴール到達 turn
- 未取得宝箱
- player ごとの発見済み landmark 情報

## `exported_snapshot`

`exported_snapshot.public_state` は観戦・デバッグ向けに、少なくとも以下を持つ。

```json
{
  "map_id": "fixed-map-v1",
  "rng_seed": 0,
  "turn": 4,
  "max_turns": 16,
  "tiles": [
    "#########",
    "#...#...#"
  ],
  "goal": { "x": 6, "y": 6 },
  "players": [
    { "player_id": "p1", "x": 2, "y": 3, "score": 12, "goal_bonus": 0, "chest_points": 12, "finished_turn": null }
  ],
  "uncollected_chests": [
    { "x": 6, "y": 3 }
  ]
}
```

`exported_snapshot` は hidden information を残さず、観戦に必要な全体状態を含める。
この MVP では fixed map 自体は秘匿対象ではないため、全体 tile layout を含めてよい。

## 初期化と deterministic 性

- `rng_seed` は match 初期化時に受け取り、`full_state` と `exported_snapshot` に保持する
- `fixed-map-v1` では map 生成に乱数を使わないため、同じ action 列なら常に同じ結果になる
- 後続 phase で seed 付き map 生成へ拡張しても、snapshot shape は変えない

## local reference bot

Phase 5 MVP の reference bot は Go subprocess で動かす。

- 現在見えている tile を基に walkable area を記憶する
- 未取得宝箱が見えていれば近いものを優先する
- そうでなければ既知ゴールへ向かう
- どちらも未発見なら frontier 探索を行う

これは開発用の最速検証経路であり、正式提出経路は引き続き WASM を正本とする。

## 共通契約との境界

この game が platform 共通契約に依存する部分:

- `init` / `turn` / `game_over` の共通メソッド境界
- `DecisionStep` による同時行動解決
- `accepted` / `no_action` と failure 分類
- record / snapshot / exported snapshot の共通 envelope
- `docs/specs/game-master.md` が定める game master session と runtime boundary

この game 側で固定する部分:

- `visible_state` / `full_state` / `exported_snapshot` の payload shape
- `move` / `wait` action schema
- 視界半径
- 宝箱競合、ゴール順位、最終順位の規則
