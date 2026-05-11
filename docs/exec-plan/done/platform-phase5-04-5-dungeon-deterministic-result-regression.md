# platform-phase5-04-5-dungeon-deterministic-result-regression
**Execution**: Use `/execute-task` to implement this plan.

## Objective

同一 `game_id`、同一 `game_version`、同一 `ruleset_version`、同一 deterministic AI Player、
同一 player 順、同一 `rng_seed` で dungeon match を複数回実行したとき、同一の result が
得られることを継続的に確認する。

この plan の目的は「その result が強いか/正しいか」を固定することではなく、
ai-arena の決定性 contract を回帰検証できるようにすることにある。

depends on:

- `docs/exec-plan/done/platform-phase5-03-dungeon-wasm-reference-ai.md`

## Scope

- deterministic 条件下で dungeon result が再現することを確認する regression test を追加する
- 比較対象となる result の正規化 shape を定義する
- deterministic guarantee の前提条件を spec に明文化する
- golden failure / update の運用ルールを定義する

この plan では以下は扱わない。

- game balance の良し悪し評価
- どの AI が勝つべきかという golden 固定
- non-deterministic AI Player や将来の stochastic policy の挙動保証
- cross-version 間での result 一致保証

## Spec Changes

### `docs/specs/platform.md`

- same `game_id`, same `game_version`, same `ruleset_version`, same deterministic
  AI Player, same player order, same `rng_seed` のときに deterministic
  result regression を確認する方針を追記する
- 再現性保証の対象外を明記する
  - `game_id` / `game_version` / `ruleset_version` が異なる場合
  - PR 作成者が deterministic AI Player 実装を変更した場合
  - player 順が異なる場合
  - AI Player が内部で非決定的乱数を使う場合
  - runtime/transport contract を破る場合

### `docs/specs/dungeon-game.md`

- dungeon regression では「同一条件なら同一 result」を first-class に確認することを追記する
- balance や strategy の評価と、determinism の確認を分けることを明記する
- deterministic result golden の failure/update rule を追記する

## Expected Code Changes

### deterministic fixture setup

- fixed-seed かつ deterministic に行動する dungeon AI Player fixture を選定する
- 同一 fixture path と同一 player 順で、同じ deterministic AI Player を比較していることを
  test comment と spec wording で明示する
- local subprocess path と、必要なら WASM path でも同じ contract を確認できるようにする

### normalized result comparison

- result 全体ではなく、決定性確認に必要な normalized result shape を定義する
- 例:
  - `result.placements`
  - per-player `score`, `goal_bonus`, `chest_points`, `finished_turn`
  - remaining chests の `(x, y, points)`
  - selected public-state fields
    - `turn` は terminal public-state 由来の値を正本として扱う
- `match_id` は test harness 側で固定して比較対象へ含めてもよいが、
  deterministic contract の本質条件としては扱わない

### golden operation rules

- regression test は、同一条件で再実行した normalized result が一致しないと失敗とする
- 次の場合は golden 更新ではなく test failure として扱う
  - 同一 `game_id` / `game_version` / `ruleset_version`
  - 同一 deterministic AI Player 実装
  - 同一 player order
  - 同一 `rng_seed`
  - にもかかわらず normalized result が変わった場合
- 次の場合は、意図された変更として golden 更新を許可する
  - `game_version` または `ruleset_version` を意図的に更新した場合
  - PR 作成者が deterministic AI Player 実装を意図的に変更した場合
  - normalized result shape 自体を意図的に見直した場合
- golden 更新時は、どの条件が変わったため更新可能なのかを PR と spec/plan で明示する

### regression tests

- 同一条件で 2 回以上 match を回し、normalized result が一致することを確認する
- 必要なら `record.json` 全体ではなく compact result 抽出関数を比較に使う

## Verification

- `go test ./e2e/... ./games/dungeon/...`
- 同一 `game_id` / `game_version` / `ruleset_version` / 同一 deterministic AI Player /
  同一 player order / 同一 `rng_seed` で
  複数回実行した result が一致する
- seed、AI logic、player 順、`game_version`、`ruleset_version` を変えたときは、
  この test が「一致しないと失敗」にはならない

## Sub-tasks

- [ ] deterministic guarantee の spec wording を追加する
- [ ] deterministic dungeon fixture/player の前提を固定する
- [ ] normalized result extractor を追加する
- [ ] same-condition rerun regression test を追加する
- [ ] local subprocess と WASM の coverage 境界を決める
- [ ] golden failure / update rule を spec と test comment に落とし込む

## Risks and Mitigations

- `match_id` や artifact path のような本質でない差分で test が壊れる
  - mitigation: normalized result shape を比較し、run-specific field を除外する
- compact result に寄せすぎて本当に確認したい determinism が抜ける
  - mitigation: ranking/score/finished-turn/public-state など、game outcome に効く field を含める
- determinism test と strategy regression test の責務が混ざる
  - mitigation: この plan は「同一条件で同一結果」を確認することだけに限定する

## Design Decisions

- determinism regression は「expected winner を固定する test」ではない
- guarantee の軸は same `game_id` + same `game_version` + same `ruleset_version` +
  same deterministic AI + same player order + same seed に限定する
- outcome comparison は full record ではなく normalized result を基本とする
