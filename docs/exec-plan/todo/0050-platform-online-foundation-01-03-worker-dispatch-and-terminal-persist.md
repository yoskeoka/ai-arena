# platform-online-foundation-01-03-worker-dispatch-and-terminal-persist
**Execution**: Use `/execute-task` to implement this plan.

## Objective

queued match を worker が claim し、runner 実行から terminal persist まで閉じる最小 execution path を実装する。
最初のゴールは、queue record を `leased -> running -> persisting -> terminal` へ進め、
`record.json` と player stderr log を `output-dir` へ保存したうえで execution summary を閉じることに置く。

## Context

- `0048` が lifecycle / boundary を定義し、`0049` が validated submission を queue へ載せる
- 既存 runner は single-match execution と artifact 出力を担えるが、worker claim / queue lifecycle は持っていない
- retry はまだ扱わず、`attempt_count=1` 前提で最初の運用を成立させる

## Scope

- queued record の worker claim を実装する
- runner 実行 input を queue record から materialize する
- runner 実行結果を `output-dir` に保存する
- player ごとの `*-stderr.log` を保存する
- terminal summary を queue/execution record へ反映する

この plan では以下を扱わない。

- retry / redelivery / DLQ
- leased 後の cancel
- distributed lock / multi-worker fairness
- replay / resume read API

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- worker claim / lease / runner invoke / terminal persist を追記する
- `attempt_count=1` 前提と no-retry rule を明記する
- player stderr artifact naming と placement rule を明記する

### `docs/specs/platform.md`

- runner 出力 artifact を service skeleton がどう受け取って保存するかを補足する

## Expected Code Changes

- worker loop
- queue claim / lease update path
- runner invocation adapter
- terminal persist orchestration
- stderr artifact write path
- worker path の integration test

## Sub-tasks

- [ ] queued record を claim して `leased` に進める
- [ ] claim 済み record を `running` に進めて runner を起動する
- [ ] terminal `record.json` と player stderr logs を `output-dir` に保存する
- [ ] `persisting` から terminal status へ進める
- [ ] completed / failed / timeout reason 付き canceled 相当の integration test を追加する

## Parallelism

- `0048` の contract 固定後は、worker claim / persist path と `0049` の submission entry path を並行で進められる

## Dependencies

- depends on: `0048-platform-online-foundation-01-01-service-contract-and-lifecycle.md`
- blocks: `0051-platform-online-foundation-01-04-cli-proof-and-e2e-verification.md`
- informs: `0046-platform-online-foundation-02-persistence-and-read-model.md`

## Risks and Mitigations

- worker が runner の内部 lifecycle まで持ち始めると責務が崩れる
  - mitigation: worker は queue ownership と terminal persist orchestration に限定し、match loop は runner に残す
- stderr 保存を record 本体へ埋め込みすぎると artifact 境界が崩れる
  - mitigation: stderr 本文は別 file にし、record / summary には locator と summary を残す

## Design Decisions

- terminal persist は file-backed first として `output-dir` を正本の保存先に使う
- retry はまだ実装しない
