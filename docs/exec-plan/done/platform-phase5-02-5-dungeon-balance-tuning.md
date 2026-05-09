# platform-phase5-02-5-dungeon-balance-tuning
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`seeded-maze-v1` のゲーム時間と得点配分を調整し、Phase 5 dungeon が「最短到達だけを競うゲーム」に潰れず、宝箱回収にも明確な勝ち筋が残る状態へ寄せる。後続の WASM reference AI は、この更新後ルールを前提に baseline 行動を組み立てる。

depends on:

- `platform-phase5-02-dungeon-seeded-generation.md`

## Scope

- `seeded-maze-v1` の最大ターン数を 50 ターンへ引き上げる
- ゴール順位ボーナスと宝箱スコア集合の配分を再設計する
- 「1位ゴールだけでは確定勝ちにならず、宝箱回収で逆転可能」という勝敗条件を ruleset 制約として固定する
- 新しい score balance に対して local reference bot / manual helper / test の期待値を更新する

この plan では以下は扱わない。

- WASM reference AI の追加
- 迷路生成アルゴリズムそのものの変更
- モンスターや戦闘の導入
- 複数フロア化

## Spec Changes

### `docs/specs/dungeon-game.md`

- `seeded-maze-v1` の最大ターン数を 16 から 50 へ更新する
- スコア設計の目的を「ゴール最優先」から「ゴールと宝箱の両方が順位に効く競争」へ言い換える
- ゴール順位ボーナスと宝箱 score set を更新する
- 少なくとも以下の balance 条件を ruleset contract として明記する
  - 1 位が `chest_points = 0` の場合、3 位でも全 `chest_points` 合計の過半を確保すれば総合点で逆転可能
  - 宝箱を無視した最短到達と、遠回りして高得点宝箱を狙う判断の両方に合理性が残る
  - 宝箱の同時取得で等分が起きる場合も、この判定は chest 個数ではなく `chest_points` の合計値で扱う
- `remaining_turns` と scoreboard 例を新しい最大ターン数 / score table に合わせて更新する

### `docs/specs/platform.md`

- dungeon の manual verification では、到達順位だけでなく treasure-heavy 戦略が有効に働くことも確認対象に含める

## Expected Code Changes

### ruleset / scoring

- dungeon ruleset の `max_turns` 定数または設定値を更新する
- goal bonus table と chest points table を新しい配分へ差し替える
- score 計算や順位決定 helper が新しい配分を前提に動くことを確認する

### verification / helpers

- score balance を固定する unit test または golden 的検証を追加する
- manual helper や debug 出力で treasure-heavy な勝ち筋を確認しやすくする
- 既存 reference bot の期待挙動が新配分で不自然になった場合、次 plan に備えて観測結果を整理する

## Verification

- `go test ./...`
- 最大ターン数更新後も deterministic match が完走する
- 1 位が `chest_points = 0`、3 位が全 `chest_points` 合計の過半を確保したケースで逆転できることをテストで確認する
- treasure-heavy と shortest-path-heavy の両方で score 差が発生し、宝箱が無価値になっていないことを manual helper で確認する

## Sub-tasks

- [ ] balance 条件を spec に明文化する
- [ ] 最大ターン数と score table を更新する
- [ ] score 計算 / 期待順位の unit test を追加する
- [ ] manual verification 用の確認観点を更新する
- [ ] `platform-phase5-03-dungeon-wasm-reference-ai` に追加の前提差分が残っていないか確認する

## Parallelism

- unit test 追加と manual verification 観点更新は、score table が固まれば並行で進められる
- `platform-phase5-03` の plan 更新は、balance 条件の文言が固まれば並行で進められる

## Risks and Mitigations

- 宝箱点を上げすぎると、ゴール到達の達成感が薄くなる
  - mitigation: 「3 位でも宝箱過半で逆転可能」を下限条件にしつつ、1 位の価値自体は残る配分に留める
- ターン数を増やしすぎると、match が間延びして manual verification が重くなる
  - mitigation: まずは 50 へ固定し、helper と実測で過不足を確認する
- score table だけ変えて bot の期待行動が崩れると、後続の WASM baseline 設計がぶれる
  - mitigation: `platform-phase5-03` をこの plan に依存させ、reference AI の成功条件も同時に更新する

## Design Decisions

- `seeded-maze-v1` は「最速ゴール一辺倒」ではなく、treasure routing でも勝ちを狙える競争として調整する
- ターン数は最短到達の余裕と寄り道の選択肢を持てる水準まで増やす
- balance の正本は個別 bot の強さではなく ruleset contract と unit test で固定する
