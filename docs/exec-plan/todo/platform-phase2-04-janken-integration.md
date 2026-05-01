# platform-phase2-04-janken-integration
**Execution**: Use `/execute-task` to implement this plan.

## Objective

親 plan の Task 12 に合わせて、`janken` richer integration へ進む入口だけを作る。

この plan では `janken` の空 package と test skeleton、責務境界の明文化、そして本実装用 follow-up plan の作成までを扱う。`janken` の中身や platform 検証のための作りこみは、この plan では実装せず別 plan に送る。

親 plan:

- `docs/exec-plan/done/platform-phase2-implementation.md`

depends on:

- `platform-phase2-02-fixture-e2e.md`

## Scope

- `internal/games/janken/`
- `janken` game master の空 package
- `janken` test skeleton
- `janken` richer integration 本実装用の follow-up exec plan

この plan で担保したいのは、`echo-count` では不足する以下の責務。

- hidden action reveal
- simultaneous resolution を伴うゲーム固有ロジック
- game-specific action schema
- ranking / tie-break の richer coverage

この plan では以下は扱わない。

- `janken` game logic の本実装
- `janken` 用 sample AI や test fixture の作りこみ
- `janken` の CLI 実行
- `janken` e2e
- hidden action / reveal / ranking の platform 上での実検証

## Spec Changes

### `docs/specs/janken-game.md`

- platform foundation 完了後の richer integration game であることを再確認する
- この plan では skeleton と責務境界だけを整え、本実装と richer verification は follow-up plan に送ることを明記する
- `echo-count` で担保済みの責務と、`janken` に残す責務を区別する

### `docs/specs/platform.md`

- 原則として変更しない。platform contract 追加が必要な場合のみ、責務逸脱でないかを確認して最小更新に留める

## Expected Code Changes

- `internal/games/janken/`
- `internal/games/janken/*_test.go` の skeleton
- `docs/exec-plan/todo/` の `janken` 本実装用 follow-up plan

## Verification

完了は docs review と skeleton の機械確認で判定する。最低限、以下を確認できること。

- `internal/games/janken/` に空 package と test skeleton があり、`go test ./...` を壊さない
- `echo-count` で担保済みの責務と `janken` に残す責務が docs 上で区別されている
- `janken` の中身、sample AI、CLI、e2e、richer verification は別 follow-up plan で扱うことが明記されている

## Sub-tasks

- [ ] Update `docs/specs/janken-game.md` to clarify skeleton-only scope and remaining `janken` responsibilities
- [ ] Add empty `internal/games/janken/` package
- [ ] Add `janken` test skeleton without committing game logic
- [ ] Write a separate follow-up exec plan for `janken` implementation and richer platform verification

## Parallelism

- spec 上の責務整理と skeleton package 追加は並行できる
- follow-up plan 作成は skeleton 追加と独立して進められる

## Risks and Mitigations

- この plan の段階で `janken` 本実装まで進めると、親 plan の Task 12 を超えて scope creep する
  - mitigation: skeleton と follow-up plan 作成までに止める
- `echo-count` と `janken` の責務境界が曖昧だと、後続 plan の verification が重複する
  - mitigation: `echo-count` で閉じた責務と `janken` に残す責務を docs で先に固定する
