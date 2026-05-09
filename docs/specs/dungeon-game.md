# ダンジョンゲーム仕様

## 目的

このドキュメントは、Phase 5 の dungeon game 固有仕様を定義する。platform 共通契約の正本は
`docs/specs/platform-common-contract.md`、game master 開発境界の正本は
`docs/specs/game-master.md` とし、この文書では dungeon game 固有の
payload / generation / validation / state progression / scoring を固定する。

この phase で成立させたいこと:

- top-level non-internal package tree に閉じた dungeon domain
- local subprocess game master での deterministic match loop
- local subprocess reference bot による高速検証
- seed だけで初期局面を完全再現できる generated map contract
- 後続の WASM reference AI へそのまま接続できる payload 境界

## ゲーム ID とルールセット

- `game_id`: `dungeon`
- `game_version`: `1.0.0`

サポートする ruleset:

- `fixed-map-v1`
  - Phase 5-01 の固定マップ baseline
  - 既存 verification / replay 互換のため残す
- `seeded-maze-v1`
  - Phase 5-02 で追加する seed 付き自動生成 ruleset
  - 以後の dungeon 開発で主に使う ruleset

`game_version` は payload schema と game master 実装の互換性を表す。`ruleset_version` は
マップ構築規則、採点、視界半径、制限ターン数をまとめた運営ルール識別子である。

## `seeded-maze-v1` の範囲

- 1 ステージ
- 2 から 4 プレイヤー
- 同時行動
- アクションは `move` と `wait` のみ
- スコア源はゴール到達順位ボーナスと宝箱のみ
- プレイヤー同士の同居は許可
- 宝箱の同時取得は等分
- 視界制限は半径ベースの局所視界

この phase では以下を扱わない。

- モンスター、戦闘、直接妨害
- 複数フロア進行
- inventory、item use、trap
- 正式 WASM reference AI

## `rng_seed` 契約

- `rng_seed` は 32 byte の seed material を表す 64 桁 hex string とする
- fresh run で caller が `rng_seed` を省略してよい場合、dungeon game master が新しい seed を生成して match を起動し、その値を snapshot / exported snapshot / record に保存する
- replay / resume では保存済み `rng_seed` を再利用する
- game master は `rng_seed` を match 初期条件の正本として保持する
- 実装は hex string をそのまま 32 byte seed material として decode し、`math/rand/v2` の `ChaCha8` に渡す
- 同じ `game_version` / `ruleset_version` / `player_id` 順 / `rng_seed` なら、初期生成結果は必ず一致しなければならない

`rng_seed` は replay / debug / exported snapshot の持ち運びや記録を優先し、外部契約では binary ではなく
hex string で扱う。platform 側は `rng_seed` が string で保存・再投入できることだけを契約とし、
内部でどの PRNG を使うかは game 実装側の責務とする。

## 迷路生成規則

`seeded-maze-v1` は 9x9 グリッドを使う。

- 外周は常に壁
- 通路幅は 1
- floor / wall のみからなる perfect maze を 1 つ生成する
- perfect maze は「全 floor が連結」「floor 間の単純路が一意」を意味する
- 初回採用アルゴリズムは recursive backtracker とする
- 実装は `math/rand/v2` による deterministic な乱数列だけを使って生成する
- 乱数が必要な game master 処理は逐次処理を原則とし、乱数消費順が goroutine scheduling や runtime の
  非決定要素で変わらないようにする
- `map` iteration 順など language runtime 由来の非決定要素を generation / placement に持ち込まない

生成後に確定する初期局面:

- `tiles`
- `spawn_points`
- `goal`
- `initial_chests`

これらをまとめてこの ruleset の generated layout と呼ぶ。

## start / goal / chest placement

generated layout の placement は以下の順で deterministic に決める。

1. `tiles` を生成する
2. `goal` を生成 maze 上の walkable tile から 1 つ選ぶ
3. `start_anchor` を `goal` から最も遠い walkable tile として選ぶ
4. `spawn_points` を `start_anchor` からの距離が近い順に 4 つ選ぶ
5. `initial_chests` を `goal` と `spawn_points` を除く walkable tile から 3 個選ぶ

追加制約:

- `spawn_points` は player 順に使う
- `spawn_points` の先頭は `start_anchor` 自身でよい
- `initial_chests` は同じ tile に重ならない
- `goal` は chest / spawn と重ならない
- 迷路生成と placement は同じ `rng_seed` で完全再現できなければならない

placement の細かい tie-break は実装に委ねてよいが、少なくとも以下を満たすこと。

- 同一入力に対して stable
- `goal` が到達不能にならない
- 全 chest が到達可能
- `spawn_points` が必要 player 数を満たす

## ターン進行

- `fixed-map-v1` の最大ターン数は 16
- `seeded-maze-v1` の最大ターン数は 50
- 各ターンで未ゴール player 全員に同時 request を送る
- 各 player は 1 アクションだけ返す
- game master は全 player の action status を集約後に一括で解決する
- ゴール済み player は後続ターンの request 対象から外す

試合終了条件:

- 全 player がゴール到達した
- または ruleset ごとの最大ターン数を消化した

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

この phase の競合解決は以下とする。

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

## スコア設計

`seeded-maze-v1` では、ゴール到達と宝箱回収の両方が最終順位に効く競争として扱う。
最短到達だけを唯一の正解にせず、treasure-heavy な遠回りにも明確な勝ち筋を残す。

- ゴール順位ボーナス:
  - 1 位: 42
  - 2 位: 28
  - 3 位: 14
  - 4 位: 7
- 宝箱 score set:
  - `{24, 18, 12}`
  - match ごとにこの固定集合を shuffle して `initial_chests` に割り当てる
  - スコア値そのものは乱数で拡張しない

この ruleset では:

- 全宝箱合計は 54 点で、宝箱過半は 28 点超とみなす
- 1 位が `chest_points = 0` の場合、3 位でも宝箱合計 30 点以上を確保すれば `14 + 30 = 44` 点となり逆転できる
- 同時取得で等分が起きる場合も、判定は「何個取ったか」ではなく最終 `chest_points` 合計で扱う
- 一方で、1 位 42 点は 2 位 + 12 点宝箱の 40 点を上回るため、最短ゴール一直線にも依然として合理性が残る
- score variance は chest placement と assignment で生み、score range 自体は固定する

## ゴール順位ボーナス規則

- ゴール順位は `finished_turn` の昇順で決める
- 同じ turn に到達した player は同順位
- 同順位は competition ranking とする
- 例: 2 人が同率 1 位なら両者に 100 点を与え、次の到達者は 3 位点 25 を得る
- ゴール未到達 player は順位ボーナス 0 点

最終順位は `goal_bonus + chest_points` の合計スコア降順で決め、こちらも
competition ranking を使う。

## payload schema

### `known_chests`

宝箱は位置だけでなく割当済み score も持つ。

```json
{ "x": 6, "y": 3, "points": 12 }
```

### `visible_state`

各 `turn` request の `visible_state` は少なくとも以下を持つ。

```json
{
  "turn": 4,
  "remaining_turns": 46,
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
    { "x": 2, "y": 3, "points": 12 }
  ],
  "scores": [
    { "player_id": "p1", "score": 12, "goal_bonus": 0, "chest_points": 12, "finished_turn": null },
    { "player_id": "p2", "score": 0, "goal_bonus": 0, "chest_points": 0, "finished_turn": null }
  ]
}
```

### `full_state`

`snapshot.game_state` に入る `full_state` は少なくとも以下を持つ。

```json
{
  "map_id": "seeded-maze-v1",
  "rng_seed": "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
  "turn": 4,
  "max_turns": 50,
  "tiles": [
    "#########",
    "#...#...#"
  ],
  "spawn_points": [
    { "x": 1, "y": 1 },
    { "x": 2, "y": 1 }
  ],
  "goal": { "x": 6, "y": 6 },
  "initial_chests": [
    { "x": 6, "y": 3, "points": 24 },
    { "x": 2, "y": 6, "points": 12 },
    { "x": 5, "y": 5, "points": 12 }
  ],
  "players": [
    { "player_id": "p1", "x": 2, "y": 3, "score": 12, "goal_bonus": 0, "chest_points": 12, "finished_turn": null }
  ],
  "uncollected_chests": [
    { "x": 6, "y": 3, "points": 24 }
  ],
  "discovery": {
    "p1": {
      "known_goal": { "x": 6, "y": 6 },
      "known_chests": [{ "x": 6, "y": 3, "points": 24 }]
    }
  }
}
```

`full_state` は resume source of truth なので、少なくとも以下を復元できなければならない。

- `map_id`
- `rng_seed`
- generated `tiles`
- generated `spawn_points`
- generated `goal`
- generated `initial_chests`
- 現在 turn
- 各 player の位置、得点、ゴール到達 turn
- 未取得宝箱
- player ごとの発見済み landmark 情報

### `exported_snapshot`

`exported_snapshot.public_state` は観戦・デバッグ向けに、少なくとも以下を持つ。
`rng_seed` を含めてよいのは terminal exported snapshot のうち最終 status が `completed` の場合だけとする。
途中実行中はもちろん、terminal でも `failed` / `canceled` の external/public exported snapshot では `rng_seed` を含めない。

```json
{
  "map_id": "seeded-maze-v1",
  "rng_seed": "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
  "turn": 4,
  "max_turns": 50,
  "tiles": [
    "#########",
    "#...#...#"
  ],
  "spawn_points": [
    { "x": 1, "y": 1 },
    { "x": 2, "y": 1 }
  ],
  "goal": { "x": 6, "y": 6 },
  "initial_chests": [
    { "x": 6, "y": 3, "points": 24 },
    { "x": 2, "y": 6, "points": 12 },
    { "x": 5, "y": 5, "points": 12 }
  ],
  "players": [
    { "player_id": "p1", "x": 2, "y": 3, "score": 12, "goal_bonus": 0, "chest_points": 12, "finished_turn": null }
  ],
  "uncollected_chests": [
    { "x": 6, "y": 3, "points": 24 }
  ]
}
```

`exported_snapshot` は hidden information を残さず、観戦に必要な全体状態を含める。
この phase では generated layout と chest score assignment は debug / replay のため公開してよいが、
`rng_seed` 自体の公開は terminal かつ `completed` の場合に限る。

## deterministic 再現条件

以下が一致すれば、match 開始時点の generated layout は一致しなければならない。

- `game_id`
- `game_version`
- `ruleset_version`
- `rng_seed`
- player 順

また、以下の運用条件を満たす。

- `full_state` には常に `rng_seed` と generated layout の両方を保持する
- `exported_snapshot` は generated layout を保持してよく、`rng_seed` は terminal かつ `completed` の公開時にだけ含める
- generated layout は「seed から再生成した結果と保存状態が一致しているか」を resume / debug で検証できる形で残す
- replay / exported snapshot の利用者は `rng_seed` だけで初期局面を再構成できる

## local reference bot

Phase 5 の reference bot は Go subprocess で動かす。

- 現在見えている tile を基に walkable area を記憶する
- 未取得宝箱が見えていれば、それを優先候補にする
- そうでなければ既知ゴールへ向かう
- どちらも未発見なら frontier 探索を行う

`seeded-maze-v1` の baseline は、到達順位だけでなく宝箱による逆転余地も前提に判断する。
少なくとも「宝箱 12 点だけでは chestless 1 位を捲れないが、18 点や 24 点、または複数宝箱の合計では逆転可能」
という score gap を利用できることを確認対象に含める。

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
- 迷路生成規則
- start / goal / chest placement
- 宝箱 score assignment
- 視界半径
- 宝箱競合、ゴール順位、最終順位の規則
