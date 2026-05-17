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
- `cmd/dungeon-gamemaster` が dungeon domain と `github.com/yoskeoka/ai-arena/gamemaster` だけで成立する portable sidecar entrypoint

## portable sidecar boundary

`cmd/dungeon-gamemaster` とその近傍 helper は、mono repo に置いているだけで
将来別 repo へ移せる前提で保つ。

- sidecar entrypoint が依存してよいのは `games/dungeon/...` と
  `github.com/yoskeoka/ai-arena/gamemaster` までとする
- platform の `internal/platform/runtime` / `session` / `registry` / `catalog` は
  ai-arena 側へ残る adapter / orchestration 実装であり、dungeon sidecar 側へ持ち込まない
- 新 repo 化で残る作業は、主に file move、module/import path 置換、repo bootstrap の最小配線で説明できる状態を維持する

## external repo bootstrap gate

`dungeon-game-ai-arena` へ dungeon game の開発場所を移す bootstrap 段階でも、
portable sidecar boundary と payload / golden contract は変えない。

- external repo 側へ持ち出す対象は、少なくとも `cmd/dungeon-gamemaster`、
  `cmd/dungeon-bot-local`、`cmd/dungeon-map-helper`、`games/dungeon/...`、
  same-golden verification に必要な `testdata/ai/dungeon/...` の portable subset、
  `e2e/golden/normalized-dungeon-result.json` とする
- external repo は sidecar SDK import に local `replace` を使わず、review 済みの ai-arena module tag を
  `go.mod` から参照する
- `ai-arena` は bootstrap 段階では dungeon verification の比較元として残り、
  external repo 側で parity が成立するまで canonical implementation を兼ねる
- bootstrap 完了条件は、現行の deterministic golden を変更せずに
  external repo 側の local run と CI e2e が同じ結果で通ることとする
- parity 差分が出た場合は、まず ruleset / payload 変更ではなく移設不備として扱う

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

## reference AI の観測境界

Phase 5 の official reference AI は、local subprocess bot と WASM bot で同じ decision layer を共有してよい。
ただし shared logic が参照してよい情報は、各 turn request の `visible_state` に含まれる public / player-visible
information だけとする。

shared decision layer は、少なくとも次の 3 責務へ分けて扱う。

- `memory`: `visible_tiles` / `known_goal` / `known_chests` と自分の過去観測から継続保持する状態更新
- `world-model`: memory に保存された既知 tile 群だけを使う path / frontier / 到達可能性 query
- `policy`: 現在の `visible_state` と world-model query を比較して `move` / `wait` を返す意思決定

この分割は hidden information を解禁するものではない。`memory` に入れてよいのは、その AI が過去 turn までに
実際に観測した情報だけである。`world-model` は memory の query layer であり、未観測 tile を補完したり
layout 全体を前提にしたりしてはならない。

reference AI が前提にしてよいもの:

- `self`
- `remaining_turns`
- `view_radius`
- `visible_tiles`
- `known_goal`
- `known_chests`
- `scores`
- 自分で保持した過去 turn の観測履歴

reference AI が前提にしてはならない hidden information:

- `full_state.tiles`
- 未発見の goal 座標
- 未発見の chest 座標や点数
- 他 player の未観測位置
- replay / debug 用に保存された generated layout 全体

視界半径や visibility shape が将来変わっても、reference AI は `visible_tiles` と `known_*` を主入力にすることで
同じ判断層を保てる形を維持する。

Policy variant を追加しても、memory update と world-model query の contract は共通のまま保つ。
例えば `balanced` と `goal-rush` が同じ turn request を受けた場合、差分として許されるのは policy の優先順位だけであり、
観測更新や hidden information の扱いは一致していなければならない。

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

## domain model の責務境界

dungeon domain は、少なくとも次の 4 層を別責務として扱う。

- `ruleset definition`
  - `map_id`
  - `max_turns`
  - `view_radius`
  - `goal_bonuses`
  - action deadline のような static rule
- `generated layout`
  - `tiles`
  - `spawn_points`
  - `goal`
  - `initial_chests`
- `match state`
  - 現在 `turn`
  - player ごとの位置、score、goal 到達 turn
  - `uncollected_chests`
  - player ごとの発見済み landmark 情報
- `contract payload`
  - `visible_state`
  - `full_state`
  - `exported_snapshot.public_state`

`ruleset definition` は seed から変化しない静的ルールであり、`generated layout` は
`rng_seed` と player 順から決まる match 初期配置である。`match state` は turn ごとに更新される
進行中データであり、payload はこれらを platform / AI 向けに投影した view である。

この責務分離を守るために、dungeon domain の façade は `Match` 1 箇所へ全責務を再集約しない。
fresh run / resume の state assembly、payload 生成、layout 生成はそれぞれ別の責務として整理する。

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
- turn 解決順は deterministic contract の一部であり、同じ入力 action status 群に対して同じ phase 順で進めなければならない

1 turn の解決 phase は以下で固定する。

1. action normalization
2. movement resolution
3. interaction resolution
4. terminal / score update
5. visibility refresh

`Match.Apply` 相当の façade API は残してよいが、内部ではこの phase 順に従う orchestration として扱う。
将来の mechanic 追加でも、既存 phase の途中へ ad-hoc な条件分岐を足して順序を曖昧にしない。

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

1. action normalization で各 player の `move` / `wait` を正規化する
2. movement resolution で全 player の移動先を確定し、player 位置を一括更新する
3. interaction resolution で宝箱取得を解決する
4. terminal / score update でゴール到達 turn 記録と score 再計算を行う
5. visibility refresh で各 player の発見済み goal / chest 情報を更新する

この phase の競合解決は以下とする。

- player 同士は同じ tile に同時存在できる
- すれ違いも許可する
- body block は導入しない
- 宝箱 tile に同じ turn で複数 player が到達した場合、その宝箱点を等分する
- 宝箱はその turn の解決完了後に消滅する

将来 mechanic の差し込み位置:

- actor interaction は interaction resolution の slot に入れる
- tile effect / trap / field hazard は interaction resolution の後半 slot に入れる
- poison / buff / summon upkeep のような actor effect tick は terminal / score update の前に入れる

この追加 slot は順序だけを先に固定する。具体ルールは後続 plan で定義する。

## subsystem seam と責務位置

Phase 5-06-03 以後の dungeon expansion は、少なくとも次の subsystem seam を固定した前提で進める。

- `actors`
  - monster spawn point や固定 NPC 初期配置のような seed 由来情報は `generated layout`
  - HP、座標、aggro、alive/dead のような進行中情報は `match state`
  - actor 間の接触判定や占有判定は `interaction resolution` の先頭 slot で扱う
- `items`
  - chest 以外の floor drop、固定配置 loot、trap chest の埋め込み位置のような seed 由来情報は `generated layout`
  - 消耗済みかどうか、誰が拾ったかのような進行中情報は `match state`
  - floor 上の item 取得判定は `interaction resolution` の chest 解決と同じ deterministic slot で扱う
- `inventory`
  - inventory 自体は `match state`
  - use / drop / equip のような解決は actor/item interaction の後、combat の前に入れる
- `combat`
  - damage、knockback、death 判定のような解決は `interaction resolution` の後半 slot で扱う
  - combat result に伴う score や terminal 判定への反映は `terminal / score update` で集約する
- `effects`
  - floor hazard、poison、buff/debuff、summon upkeep のような継続 effect は `match state`
  - tick 順と寿命更新は `terminal / score update` の冒頭 slot で扱う
- `visibility` / `fov`
  - line-of-sight shape や occlusion rule は ruleset 定義
  - player ごとの既知情報と観測履歴は `match state`
  - `visible_state` / `known_*` への投影は常に `visibility refresh` の責務とし、他 subsystem から hidden information を直接返してはならない

この seam は package tree 上でも `games/dungeon` の top-level domain に留め、`internal` 依存や runner 直結 helper を逆流させない。

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
- 例: 2 人が同率 1 位なら両者に 42 点を与え、次の到達者は 3 位点 14 を得る
- ゴール未到達 player は順位ボーナス 0 点

最終順位は `goal_bonus + chest_points` の合計スコア降順で決め、こちらも
competition ranking を使う。

## baseline strategy の期待値

Phase 5 の dungeon reference AI は、毎回最適手を出す solver ではなく、継続的に完走しつつ
1 位または 2 位争いへ絡める baseline を目指す。

この ruleset での baseline 戦略は、少なくとも次の性質を持つことを期待する。

- 24 点や 18 点 chest のような高価値 chest を、残りターンと goal 余裕を見ながら拾いにいける
- 12 点 chest や遠すぎる detour を無条件で追わず、終盤は goal 到達へ pivot できる
- goal 未発見時は frontier 探索を継続し、goal 発見後は chest detour と finish pace を比較して進路を選べる
- `scores` を使って、自分が無得点 finish では逆転されやすい局面と、すでに十分な加点を持つ局面を雑にでも区別できる

具体的な heuristic は実装に委ねるが、少なくとも「常に最短 goal だけを目指す」または
「見えた chest を点数や残りターンを無視して必ず追う」のどちらか一方に固定しないこと。

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
    { "x": 2, "y": 6, "points": 18 },
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

`full_state` は external contract 上は 1 つの JSON payload だが、domain 内部では
「ruleset definition + generated layout + mutable match state」を束ねた snapshot view として扱う。
resume では payload decode 後に、少なくとも以下の順で整合確認する。

1. `game_version` / `ruleset_version` から static rule を選ぶ
2. `rng_seed` から generated layout を再構築する
3. payload に入っていた `tiles` / `spawn_points` / `goal` / `initial_chests` が generated layout と一致することを確認する
4. その上で turn / player state / discovery / uncollected chests を mutable match state として復元する

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

`exported_snapshot.public_state` も domain 内部の state 所有そのものではなく、generated layout と
現在の match state を観戦向けに束ねた contract payload として扱う。将来 actor / item / effect が増えても、
新要素はまず `generated layout` 由来か `match state` 由来かを決めてから payload へ載せる。

### `result-summary.json`

`result-summary.json` は dungeon の local verification と AI Agent 実装時に最初に読む compact derived artifact とする。
少なくとも以下を持つ。

- match / game / ruleset の識別子
- final `status` と順位
- player ごとの `score` / `goal_bonus` / `chest_points` / `finished_turn`
- `public_state` から取り出した `map_id` / `turn` / `max_turns` / `goal`
- 残宝箱一覧、および count / total points
- `record.json` / `exported-snapshot.json` / `snapshot.json` / `structured-log.ndjson` / `history.json` への artifact path 参照

この summary は `exported_snapshot.public_state` の縮小コピーではなく、通常の確認導線に必要な field だけを抜き出した compact view とする。
詳細な因果追跡や full replay が必要になった場合だけ `record.json.event_log` / `history.json` / `structured-log.ndjson` へ降りる。

## targeted scenario verification

dungeon の correctness gate は seed replay だけに寄せず、handcrafted scenario catalog を first-class に持つ。

- scenario catalog は 1 scenario 1 intent を守り、同時 chest 取得、goal race、視界再発見、残りターンぎりぎり到達のような mechanic 単位で増やす
- scenario input は random generation を経由しない hand-crafted `full_state` 相当を基本にし、ruleset 固有の tile / spawn / goal / chest 配置と turn / score / discovery を短く固定できるようにする
- targeted scenario test では必要 turn だけ進め、中間 turn の `known_goal`、`known_chests`、score breakdown、`finished_turn` など selected field を確認してよい
- fixed-seed reference AI regression は別レイヤとして残してよいが、correctness の主責務は scenario catalog 側に置く
- full `record.json` golden や full exported snapshot golden は必須にせず、compact assertion と scenario intent の読みやすさを優先する

verification layer の役割分担:

- scenario catalog
  - mechanic 単位の correctness gate
  - subsystem seam 追加時も、中間 turn の selected field と hidden-information boundary を確認する第一ゲート
- deterministic result regression
  - same-condition rerun で public outcome が drift していないかを見る gate
  - selected public-state field、順位、score breakdown、残差分だけを比較し、subsystem 内部の全 state dump は比較しない
- replay / resume verification
  - `snapshot.json` / `history.json` / `record.json` から source-of-truth state を再構築できるかを見る gate
  - scenario catalog の代替ではなく、debug 再現性と source-of-truth contract の gate として扱う

## local verification の既定導線

- まず `result-summary.json` を見て順位、score breakdown、残宝箱、goal 到達状況を確認する
- layout や終局 public state をもう少し見たいときだけ `exported-snapshot.json` を開く
- per-turn causal trace、failure reason 詳細、debug replay が必要なときだけ `structured-log.ndjson` / `record.json` / `history.json` を読む

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

同一 `game_id` / `game_version` / `ruleset_version` / deterministic AI 実装 / player 順 / `rng_seed` で
match を複数回 fresh run した場合、Phase 5 dungeon は generated layout だけでなく final result も
一致しなければならない。ここでいう deterministic AI 実装には、local subprocess / WASM の実行形態差ではなく、
同じ decision layer と同じ policy 入力から同じ action 列を返すことを期待する。

subsystem 拡張後も、少なくとも以下を deterministic contract に含める。

- single RNG ownership
  - match 中に乱数を消費してよいのは dungeon domain が所有する単一 stream だけとする
  - actor spawn、item roll、combat variation、effect proc のような subsystem 追加でも、独自 RNG や runtime 依存 seed を持ち込まない
- fixed phase order
  - action normalization -> movement resolution -> interaction resolution -> terminal / score update -> visibility refresh
  - subsystem hook はこの 5 phase の内側 slot へだけ差し込んでよく、新しい top-level phase を ad-hoc に増やさない
- stable iteration order
  - player 順は `playerOrder` を正本とする
  - chest / item / actor / effect の複数要素処理は stable な slice 順に正規化してから走査し、Go の `map` iteration 順へ依存しない
  - tie-break が必要な場合は `player_id`、座標、または spec で定義した key 順を使う
- replay / resume source of truth
  - `full_state` は generated layout、`rng_seed`、および進行中 subsystem state の正本である
  - replay/debug helper は `record.json` / `history.json` / `snapshot.json` から復元してよいが、runtime-specific cache や log timestamp を state の正本にしてはならない

### deterministic result regression shape

same-condition regression では full `record.json` をそのまま golden 化せず、`result-summary.json` と
terminal public state から導ける compact な normalized result shape を比較する。少なくとも以下を含める。

- `placements`
- player ごとの `score` / `goal_bonus` / `chest_points` / `finished_turn`
- `map_id` / `turn` / `max_turns` のような selected public-state field
  - `turn` は terminal public state 由来の値を正本とし、summary top-level field と不一致なら test failure とみなしてよい
- `remaining_chests` の `(x, y, points)` と、その count / total points

以下のような run-specific field は比較対象に含めない。

- `match_id`
- artifact path
- log sequence や timestamp のような harness 依存値

### deterministic result golden の運用

- same-condition regression test は、同一条件で再実行した normalized result が一致しない場合に失敗しなければならない
- この failure は「勝者が変わった」「score breakdown が変わった」「残宝箱が変わった」などを含む deterministic drift として扱い、まず golden 更新ではなく実装修正対象とみなす
- golden 更新を許可するのは、`game_version` または `ruleset_version` の意図的更新、deterministic AI 実装の意図的変更、normalized result shape 自体の見直しを行った場合に限る
- golden 更新時は、何が変わったため更新可能なのかを PR と spec/plan に明示する
- たとえば `remaining_chests` の座標まで deterministic contract に含めるよう shape を強めた場合や、どの summary field を public-state の正本として比較するかを見直した場合は、その shape review を更新理由として明示してよい

Phase 5 時点の必須 coverage は Go local-subprocess reference bot path に置く。WASM path は同じ decision layer を
使ってよいが、この phase では runtime parity まで同じ regression test へ含めることを必須にしない。

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

## 将来拡張の state slot

Phase 5 後続で actor / item / effect / combat を追加するときも、責務位置は先に固定する。

- actor spawn point や固定 obstacle のように seed から決まるものは `generated layout`
- monster HP、inventory、temporary effect、floor 上の消耗済み object のように turn で変化するものは `match state`
- AI へ見せる要約や spectator 向け集約は `contract payload`

この段階では slot 位置だけを固定し、具体的な field 追加は後続 plan で扱う。
