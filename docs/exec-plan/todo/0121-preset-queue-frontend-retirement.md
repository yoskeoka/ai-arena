# preset-queue-frontend-retirement
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

operator が registered game、bot、match request、ranking を使う運営導線へ移行したため、overview から
server-known preset を one click で enqueue する UI を撤去する。完了時、`/operator` は active/completed run と
run detail を引き続き表示するが、preset catalog、queue action、preset 固有 error/state は表示も送信もしない。

この plan は frontend-only である。`POST /api/v1/preset-matches`、preset config、service の preset materialization
は後続 plan まで保持し、既存 client/server compatibility をこの段階で変更しない。

## Existing References

- `docs/specs/platform-service-operator-ui.md:12-27, 55-96, 274-283`: overview と preset queue の現在 contract。
- `operator-ui/src/routes/operator/OperatorPage.tsx:1-34` と
  `operator-ui/src/routes/operator/useOperatorPageState.ts:128-172`: overview composition と preset mutation state。
- `operator-ui/src/routes/operator/PresetQueuePanel.tsx:1-35` と `operator-ui/src/presets.ts:1-13`:
  static catalog と visual action。
- `operator-ui/src/lib/operatorApiClient.ts:252-266`: handwritten browser adapter の preset call。
- `operator-ui/tests/operator-ui.spec.js:29-70`: fixture lane の preset click assertion。
- `docs/development/operator-ui-local-verification.md:156-169`: stable test-id inventory。

## Code Change Map

- `(MODIFY) docs/specs/platform-service-operator-ui.md`: overview を active/completed run と detail の read/follow-up
  surface として定義し直し、preset queue interaction、preset catalog state、preset test-id を除く。
- `(DELETE) operator-ui/src/routes/operator/PresetQueuePanel.tsx`: preset button/panel を削除する。
- `(DELETE) operator-ui/src/presets.ts`: browser-owned static preset catalog を削除する。
- `(MODIFY) operator-ui/src/routes/operator/OperatorPage.tsx` と
  `(MODIFY) operator-ui/src/routes/operator/useOperatorPageState.ts`: preset props、mutation state、enqueue callback を
  削除し、active/completed/detail polling を保つ。
- `(MODIFY) operator-ui/src/lib/operatorApiClient.ts`: handwritten `enqueuePreset` adapter と不要になった imports を除く。
  generated client は API retirement まで変更しない。
- `(MODIFY) operator-ui/src/routes/operator/OperatorHeader.tsx`、
  `(MODIFY) operator-ui/tests/operator-ui.spec.js`、
  `(MODIFY) docs/development/operator-ui-local-verification.md`: preset を operator surface/fixture coverage/stable
  selector として扱う記述・assertion を削除し、残る overview observation を確認する。

## Black-box Contract

- `/operator` overview は preset action を提供せず、browser load 時に preset enqueue mutation を行わない。
- active/completed run list、run detail、cancel/retry/rerun/promote 等の既存 run follow-up surface は維持する。
- backend preset endpoint の availability は本 plan の observable frontend contract に含めない。

## Dependencies and Order

1. spec を先に更新し、frontend の撤去境界を固定する。
2. panel/catalog/state/handwritten adapter と browser assertions を一緒に削除する。
3. fixture backend や generated TypeSpec output を、この frontend-only PR では API compatibility のため残す。
4. `0122-wasi-bundle-admission-e2e-regression.md` が general lane の replacement regression を固定した後だけ、
   `0123-preset-match-api-retirement.md` を開始する。

## Verification

- `pnpm run verify:local` で `/operator` の active/completed/detail fixture observation が通り、preset selector/call
  へ依存しない。
- `pnpm build`、`make lint`、`git diff --check`、
  `./tools/workflow-lint.sh --mode=pre-push` を通す。
- browser network assertion または adapter unit coverage で、overview load/click が
  `/api/v1/preset-matches` を呼ばないことを確認する。

