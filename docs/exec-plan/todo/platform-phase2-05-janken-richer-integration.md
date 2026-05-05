# platform-phase2-05-janken-richer-integration
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`janken` を `echo-count` fixture の次段として本実装し、Phase 2 で不足している hidden action reveal、simultaneous resolution、game-specific action schema、ranking / tie-break を platform 上で実検証できる状態にする。

depends on:

- `platform-phase2-04-janken-integration.md`

## Scope

- `internal/games/janken/` の game master 実装
- `janken` metadata / ruleset selection
- `janken` 用 sample AI と fixture
- `arena-runner` からの `janken` 実行
- `janken` unit test / e2e / replay-debug coverage

この plan の非目標:

- ダンジョンゲーム固有仕様
- 観戦 UI 実装
- WASM 実行基盤への拡張

## Spec Changes

### `docs/specs/janken-game.md`

- `init` / `turn` / `game_over` payload を code-level shape と整合させる
- hidden action reveal と `self_history` / `public_history` の更新タイミングを確定する
- `no_action` / invalid action / timeout を順位へ反映する集計規則を確定する

### `docs/specs/platform.md`

- `janken` を fixture ではない richer integration game として runner 入力例、artifact 例、replay/debug 適用範囲へ接続する
- `echo-count` appendix で閉じる責務と `janken` verification に委ねる責務の境界を再確認する

## Expected Code Changes

- `internal/games/janken/`
- `internal/platform/replay/`
- `cmd/arena-runner/`
- `testdata/ai/janken/` または同等の fixture
- `e2e/`

## Verification

完了は unit test と `arena-runner` 実行で判定する。最低限、以下を機械的に確認できること。

- hidden action が当該ラウンド解決前に他プレイヤーへ漏れない
- simultaneous round resolution と `public_history` 更新が spec 通りに進む
- timeout / invalid action が `no_action` として順位へ反映される
- tie-break を含む最終 placement が spec 通りに計算される
- `janken` match record / snapshot / history から replay-debug が破綻しない

## Sub-tasks

- [ ] Update `docs/specs/janken-game.md` and `docs/specs/platform.md` for implementation-level contract
- [ ] Implement `internal/games/janken` metadata and master lifecycle
- [ ] Add unit coverage for action normalization, round resolution, reveal timing, and placements
- [ ] Wire `arena-runner` to execute `janken`
- [ ] Add `janken` sample AI / fixtures
- [ ] Add e2e and replay-debug coverage for `janken`

## Parallelism

- unit test 用の game logic 実装と sample AI 準備は並行できる
- runner wiring 後は e2e と replay-debug coverage を別 stream で増やせる

## Risks and Mitigations

- `echo-count` と `janken` の責務が再び混ざると verification の意味が薄れる
  - mitigation: fixture coverage と richer integration coverage を spec 冒頭で固定してから code を進める
- hidden action の表現を急ぐと replay/snapshot export と食い違う
  - mitigation: round resolution と exported state schema を unit test で先に固定する
- sample AI を厚くしすぎると game master 検証より fixture 作り込みに寄る
  - mitigation: 最小 AI で protocol / timing / ranking を確認する用途に限定する
