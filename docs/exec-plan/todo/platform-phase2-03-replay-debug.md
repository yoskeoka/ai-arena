# platform-phase2-03-replay-debug
**Execution**: Use `/execute-task` to implement this plan.

## Objective

turn 境界 snapshot と event history を使って、debug 用の `start-from-snapshot` と `resume-from-history-and-continue` を CLI と e2e で成立させる。

親 plan:

- `docs/exec-plan/done/platform-phase2-implementation.md`

depends on:

- `platform-phase2-02-fixture-e2e.md`
- `platform-phase2-02-5-arena-runner-log-persist-split.md`

## Scope

- persisted final match-record artifact input as the source of truth
- snapshot file input extracted from a persisted artifact or hand-crafted for debug
- history file input extracted from a persisted artifact
- target turn 指定による replay/resume
- debug 用 entrypoint と verification

この plan の非目標:

- AI process memory を含む完全 continuation
- 本番障害時の in-flight 復旧
- 観戦 UI replay player

## Spec Changes

### `docs/specs/platform.md`

- `start-from-snapshot` の入力 shape と再現範囲を定義する
- `resume-from-history-and-continue` の入力 shape と再現範囲を定義する
- replay/debug CLI の source of truth は persisted final match-record artifact であり、snapshot/history file input はそこから抽出した debug entrypoint として扱うことを定義する
- runner log stream は replay/debug input として読まないことを定義する
- AI memory continuity 非保証を明記する
- snapshot と exported snapshot の違いを再確認できるようにする

## Expected Code Changes

- `cmd/arena-runner/` の snapshot/history CLI option
- `internal/platform/match/` または replay 専用 package
- replay 用 test fixture data
- `e2e/` の replay coverage

## Verification

完了は CLI 実行と e2e で判定する。最低限、以下を機械的に確認できること。

- snapshot file から `echo-count` match を開始できる
- persisted final match-record artifact から snapshot/history を辿って replay/debug を開始できる
- history file と target turn から replay した後、その続きだけ新しい AI process で進行できる
- history に記録済みの選択は再問い合わせしない
- snapshot / history から再開しても、非目標である AI memory continuity は保証しないことが runner 挙動と spec で整合している

## Sub-tasks

- [ ] Update `docs/specs/platform.md` replay/debug sections
- [ ] Add runner snapshot input path
- [ ] Add runner history replay and resume path
- [ ] Add unit coverage for snapshot restore and history replay
- [ ] Add CLI / e2e verification for snapshot start
- [ ] Add CLI / e2e verification for history resume

## Parallelism

- snapshot restore と history replay の unit coverage は別 stream にできる
- CLI/e2e は fixture record data の準備後に並行で増やせる

## Risks and Mitigations

- replay が foundation 実装に食い込むと plan 境界が崩れる
  - mitigation: foundation では schema まで、entrypoint と resume UX はこの plan に限定する
- full recovery を期待させると責務過大になる
  - mitigation: debug 用機能であることを spec と verification の両方で固定する
