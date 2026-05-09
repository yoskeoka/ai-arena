# platform-phase5-05-dungeon-reference-ai-memory-policy-split
**Execution**: Use `/execute-task` to implement this plan.

## Objective

dungeon reference AI を、memory/world-model と policy を分けた構造へ整理する。
今後の feature expansion に備え、観測更新・世界理解・意思決定を独立に進化させられる
baseline architecture を作る。

depends on:

- `docs/exec-plan/done/platform-phase5-03-dungeon-wasm-reference-ai.md`
- `docs/exec-plan/todo/platform-phase5-04-dungeon-scenario-catalog-and-targeted-fixtures.md`
- `docs/issues/dungeon-reference-ai-should-separate-memory-and-policy.md`

## Scope

- current reference AI の責務分解
- memory update / world query / policy interface の導入
- 少なくとも 2 種類の policy を同じ memory/world-model で差し替えられる構造へ整理
- scenario catalog を使った AI component test の追加

この plan では以下は扱わない。

- learned policy や search-heavy 最適化
- 多言語 reference AI の再設計
- dungeon rules 自体の追加

## Spec Changes

### `docs/specs/dungeon-game.md`

- reference AI が hidden information をどう扱うかの責務境界を補足する
- shared decision layer の中で memory と policy を分ける前提を記録する

### `docs/specs/ai-runtime.md`

- subprocess bot と WASM bot が同一 world-model / policy contract を共有できることを補足する

## Expected Code Changes

### reference AI structure

- `games/dungeon` または `games/dungeon/botlogic` 配下で
  memory/world-model/policy の package or file 分割を行う
- observation update を pure function or narrow state transition へ寄せる
- policy は current balanced heuristic を 1 実装としてぶら下げる

### policy variants

- `balanced` に加えて `goal-rush` または `treasure-biased` のような比較用 policy を追加する
- fixed-seed compact regression で policy ごとの差が観測できるようにする

### tests

- memory update 単体
- world-model query/path utility 単体
- policy decision 単体
- 統合動作

## Verification

- `go test ./games/dungeon/... ./e2e/...`
- subprocess bot と WASM bot が同一 policy contract を使って動作する
- scenario catalog 上で memory update と policy の責務が独立に検証できる

## Sub-tasks

- [ ] 現行 bot の責務分解点を整理する
- [ ] memory/world-model/policy interface を導入する
- [ ] balanced policy を新構造へ移す
- [ ] 比較用 policy を 1 つ追加する
- [ ] scenario catalog ベースの component/integration test を追加する

## Risks and Mitigations

- 分割だけ先行して baseline bot の挙動が崩れる
  - mitigation: current heuristic を golden reference として parity を確認しながら移す
- abstraction を早く作りすぎて feature 要件とずれる
  - mitigation: monsters/traps/inventory を見越しつつ、現時点では 3 責務に限定する
- policy variant 導入が強さ比較へ脱線する
  - mitigation: 目的を architecture verification と regression diversity に限定する

## Design Decisions

- reference AI の次段階は「強さ」より先に「責務分離」を優先する
- memory/world-model は複数 policy の共通基盤とする
- scenario catalog を AI refactor の安全網として先に用意した上で分割する
