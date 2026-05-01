# platform-phase2-04-janken-integration
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`echo-count` fixture で固めた platform foundation の上に `janken` を載せ、Phase 2 の主実証ゲームとして richer integration coverage を追加する。

親 plan:

- `docs/exec-plan/done/platform-phase2-implementation.md`

depends on:

- `platform-phase2-02-fixture-e2e.md`

## Scope

- `internal/games/janken/`
- `janken` 用 sample AI または test fixture
- `janken` の CLI 実行
- `janken` e2e

この plan で担保したいのは、`echo-count` では不足する以下の責務。

- hidden action reveal
- simultaneous resolution を伴うゲーム固有ロジック
- game-specific action schema
- ranking / tie-break の richer coverage

## Spec Changes

### `docs/specs/janken-game.md`

- platform foundation 完了後の integration game としての位置付けを具体化する
- `init` / `turn` / `game_over` payload と game-specific validation を実装可能な粒度へ詰める
- score / placement / tie-break を必要なら追記する

### `docs/specs/platform.md`

- 原則として変更しない。platform contract 追加が必要な場合のみ、責務逸脱でないかを確認して最小更新に留める

## Expected Code Changes

- `internal/games/janken/`
- `testdata/ai/janken/` または同等の test fixture
- `e2e/` の `janken` coverage

## Verification

完了は CLI 実行と e2e で判定する。最低限、以下を機械的に確認できること。

- `arena-runner` から `janken` match を起動できる
- hidden action と reveal 後の解決結果が期待どおり
- illegal action や timeout が `janken` ルール上 `no_action` として処理される
- score / placement / tie-break が `janken` spec と一致する

## Sub-tasks

- [ ] Update `docs/specs/janken-game.md` for implementation detail
- [ ] Implement `janken` game master
- [ ] Add `janken` fixture AI or sample AI
- [ ] Add CLI verification path for `janken`
- [ ] Add `janken` e2e coverage

## Parallelism

- `janken` game logic と test AI 準備は並行できる
- e2e は AI fixture が揃った後に別 stream で追加できる

## Risks and Mitigations

- `janken` 着手が早すぎると platform core の不具合とゲーム不具合が混ざる
  - mitigation: `platform-phase2-02-fixture-e2e.md` 完了を前提にする
- `echo-count` と同じ assertion を重複して増やすと価値が薄い
  - mitigation: `janken` では hidden action / reveal / richer ranking に集中する
