# platform-phase5-04-5-dungeon-deterministic-result-regression
**Execution**: Use `/execute-task` to implement this plan.

## Objective

同一 `game master version`、同一 deterministic AI Player、同一 `rng_seed` で dungeon match を
複数回実行したとき、同一の result が得られることを継続的に確認する。

この plan の目的は「その result が強いか/正しいか」を固定することではなく、
ai-arena の決定性 contract を回帰検証できるようにすることにある。

depends on:

- `docs/exec-plan/done/platform-phase5-03-dungeon-wasm-reference-ai.md`

## Scope

- deterministic 条件下で dungeon result が再現することを確認する regression test を追加する
- 比較対象となる result の正規化 shape を定義する
- deterministic guarantee の前提条件を spec に明文化する

この plan では以下は扱わない。

- game balance の良し悪し評価
- どの AI が勝つべきかという golden 固定
- non-deterministic AI Player や将来の stochastic policy の挙動保証
- cross-version 間での result 一致保証

## Spec Changes

### `docs/specs/platform.md`

- same game master version, same AI artifact/input, same `rng_seed` のときに
  deterministic result regression を確認する方針を追記する
- 再現性保証の対象外を明記する
  - game master version が異なる場合
  - AI Player が内部で非決定的乱数を使う場合
  - runtime/transport contract を破る場合

### `docs/specs/dungeon-game.md`

- dungeon regression では「同一条件なら同一 result」を first-class に確認することを追記する
- balance や strategy の評価と、determinism の確認を分けることを明記する

## Expected Code Changes

### deterministic fixture setup

- fixed-seed かつ deterministic に行動する dungeon AI Player fixture を選定する
- local subprocess path と、必要なら WASM path でも同じ contract を確認できるようにする

### normalized result comparison

- result 全体ではなく、決定性確認に必要な normalized result shape を定義する
- 例:
  - `match_id` を除いた placements
  - per-player `score`, `goal_bonus`, `chest_points`, `finished_turn`
  - remaining chests
  - selected public-state fields

### regression tests

- 同一条件で 2 回以上 match を回し、normalized result が一致することを確認する
- 必要なら `record.json` 全体ではなく compact result 抽出関数を比較に使う

## Verification

- `go test ./e2e/... ./games/dungeon/...`
- 同一 seed / 同一 deterministic AI Player / 同一 game master version で
  複数回実行した result が一致する
- seed や AI を変えたときは、この test が「一致しないと失敗」にはならない

## Sub-tasks

- [ ] deterministic guarantee の spec wording を追加する
- [ ] deterministic dungeon fixture/player の前提を固定する
- [ ] normalized result extractor を追加する
- [ ] same-condition rerun regression test を追加する
- [ ] local subprocess と WASM の coverage 境界を決める

## Risks and Mitigations

- `match_id` や artifact path のような本質でない差分で test が壊れる
  - mitigation: normalized result shape を比較し、run-specific field を除外する
- compact result に寄せすぎて本当に確認したい determinism が抜ける
  - mitigation: ranking/score/finished-turn/public-state など、game outcome に効く field を含める
- determinism test と strategy regression test の責務が混ざる
  - mitigation: この plan は「同一条件で同一結果」を確認することだけに限定する

## Design Decisions

- determinism regression は「expected winner を固定する test」ではない
- guarantee の軸は same version + same deterministic AI + same seed に限定する
- outcome comparison は full record ではなく normalized result を基本とする
