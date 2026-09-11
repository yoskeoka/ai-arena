# operator-run-detail-contrast
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

`/operator/runs/{run_id}` の濃色 compact summary で、Attempt、Game、Ruleset、Output Dir、Result Summary の
label と value を反転選択なしに判読できるようにする。濃色背景内の metadata は明色で表示し、通常の明色 surface
における既存の黒系 metadata 表示は変えない。

完了は fixture local browser lane で Run Detail を表示し、濃色 summary 内の全 metadata が dark-background
専用の readable text style を使うこと、ならびに operator UI に同じ dark background と black text の組合せを
残さないことを確認した時点とする。

## 現行参照と前提

- `docs/specs/platform-service-operator-ui.md:86-91, 116-127`: run detail の compact summary と browser
  observation surface を定める。visual design を固定しないまま、operator が読める状態を observable behavior
  として補う。
- `operator-ui/src/routes/operator/CompletedDetailPanel.tsx:45-61`: `bg-ink` summary と、その内側に置かれる
  shared metadata を所有する。現在の shared metadata は dark surface を認識しない。
- `operator-ui/src/shared/ui/Meta.tsx:1-8`: label/value に `text-black/*` を固定している reusable metadata
  presentation seam である。
- `operator-ui/src/routes/operator/RunDetailPage.tsx:76-94`: Run Detail route が `CompletedDetailPanel` を
  reuse することを示す。
- `operator-ui/src/routes/operator/OperatorLayout.tsx:56-71` と `operator-ui/src/routes/operator/RunDetailPage.tsx:137-159`:
  operator UI の既存 dark surface は `text-paper` を明示しており、dark context で inherited/default black を
  用いない既存方針を示す。

## 変更対象

- `(MODIFY) docs/specs/platform-service-operator-ui.md`: run detail の compact summary 内に置く text metadata は、
  background と十分に区別できる foreground color を明示しなければならないという観測可能な accessibility
  contract を追加する。特定の色値・CSS framework・design system は規定しない。
- `(MODIFY) operator-ui/src/shared/ui/Meta.tsx`: default light-surface style を維持したまま、dark surface 用の
  opt-in presentation variant を追加する。
- `(MODIFY) operator-ui/src/routes/operator/CompletedDetailPanel.tsx`: `bg-ink` compact summary 内の five metadata
  fields に dark-surface variant を指定する。Summary/Replay Inputs の white surface の metadata は default のまま
  保つ。
- `(MODIFY) operator-ui/src/routes/operator/*.(test.)*` または既存 browser verification: dark-surface metadata
  の selected style と Run Detail route の visual readability を、既存 test seam で検証する。既存 harness が
  computed-style assertion を持たない場合は、fixture lane の screenshot/manual artifact で記録する。

## black-box contract の変更

- Run Detail の compact summary は lifecycle/action message に限らず、attempt、game identity、ruleset、output
  directory、result summary locator を通常の operator display 状態で判読可能に表示する。
- dark summary 内の foreground text は background と視覚的に区別されなければならない。white/light surface の
  metadata の visual treatment、route/API payload、polling、run follow-up action availability は変更しない。

## サブタスク・順序・依存関係

1. code より先に operator UI spec へ dark compact-summary readability contract を追加する。
2. shared metadata component に narrow な dark-surface opt-in を追加し、default を既存 light-surface semantics の
   まま維持する。
3. Run Detail/Completed Detail の `bg-ink` summary の五つの metadata field だけを opt-in させる。operator UI の
   dark background occurrences を再検索し、black foreground が残る同型箇所があれば同じ contract に合わせて修正する。
4. TypeScript/production build、fixture local browser verification、workflow lint を実行し、Run Detail screenshot
   または browser inspection で label/value の判読性を確認する。

step 1 の contract が確定後、step 2 と shared-component unit assertion は並行可能である。step 3 は step 2 の
variant に依存し、step 4 はすべての変更後に行う。

## 検証

- component/source-level assertion で dark variant が label に readable muted light color、value に readable light
  colorを適用し、default が既存 black foreground を維持することを確認する。
- `pnpm run build` を `operator-ui/` で実行する。
- `pnpm run verify:local` を `operator-ui/` で実行し、fixture-seeded Run Detail の dark compact summary を browser
  で確認する。
- `git diff --check` と `./tools/workflow-lint.sh --mode=pre-push` を実行する。

## 非目標と拒否条件

- color palette 全体の redesign、route/API contract、run lifecycle/action policy、artifact locator の内容、white
  surface の metadata theme は変更しない。
- arbitrary global CSS override または dark context の implicit inheritance に頼らない。shared metadata を dark
  surface に置く場合は explicit opt-in を使う。
