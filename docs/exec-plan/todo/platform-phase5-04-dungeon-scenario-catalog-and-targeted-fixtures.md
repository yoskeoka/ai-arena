# platform-phase5-04-dungeon-scenario-catalog-and-targeted-fixtures
**Execution**: Use `/execute-task` to implement this plan.

## Objective

seed replay だけに頼らず、handcrafted snapshot/state から始める targeted scenario test
を dungeon に導入する。機能追加時に「狙った局面を短く再現し、必要なターンだけ回して、
途中結果まで確認する」検証基盤を先に整える。

depends on:

- `docs/exec-plan/done/platform-phase5-03-dungeon-wasm-reference-ai.md`
- `docs/issues/dungeon-needs-handcrafted-scenario-catalog-for-targeted-tests.md`

## Scope

- dungeon scenario catalog の最小構造を定義する
- handcrafted snapshot / state builder を追加する
- scenario ごとに intermediate/final assertion を持つ test を追加する
- fixed-seed reference AI regression と targeted scenario correctness を分離する

この plan では以下は扱わない。

- dungeon feature expansion 自体
- full `record.json` / full exported snapshot golden の大量追加
- reference AI の大規模 refactor

## Spec Changes

### `docs/specs/dungeon-game.md`

- targeted scenario test の目的と、seed replay との役割分担を追記する
- dungeon mechanic ごとに scenario fixture を持てることを明記する

### `docs/specs/platform.md`

- `--snapshot-input` / `--history-input` を使った targeted verification の想定ユースケースを補足する
- compact assertion と full replay artifact の使い分けを補足する

## Expected Code Changes

### scenario representation

- `games/dungeon/testdata` または同等の場所に scenario catalog を追加する
- 1 scenario 1 mechanic の命名と metadata を定める

### fixture/state builder

- random generation を経由せずに dungeon state を組み立てる helper を追加する
- 必要なら exported snapshot から internal state へ戻す test-only helper を追加する

### tests

- 同時 chest 取得、goal race、視界再発見、残りターンぎりぎり到達など、
  代表 scenario の targeted test を追加する
- intermediate turn の selected fields を確認する test を追加する
- fixed-seed reference AI regression は compact result assertion に寄せる

## Verification

- `go test ./games/dungeon ./e2e/...`
- scenario catalog の代表ケースが generation 非依存で再現できる
- feature 追加時に seed replay をいじらず scenario を増やせる

## Sub-tasks

- [ ] scenario catalog の schema と配置を決める
- [ ] handcrafted state builder を追加する
- [ ] representative scenario test を追加する
- [ ] fixed-seed regression と targeted correctness の test 層を分離する
- [ ] docs/specs に役割分担を追記する

## Risks and Mitigations

- fixture が内部実装へ密結合して壊れやすくなる
  - mitigation: scenario input はできるだけ public/exported snapshot 形に寄せる
- scenario 数が増えすぎて保守コストが上がる
  - mitigation: 1 scenario 1 intent を守り、full match golden を避ける
- seed replay と scenario test の責務が曖昧になる
  - mitigation: correctness gate は targeted scenario、league regression は compact result と明文化する

## Design Decisions

- dungeon の correctness は random generation 再現性だけで担保しない
- handcrafted scenario は短く意図の明確な局面を優先する
- intermediate assertion を first-class に扱い、最終順位だけの golden へ寄せすぎない
