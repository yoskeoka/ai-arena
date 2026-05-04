# arena-runner-artifact-io-contract
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`arena-runner` の artifact input/output 契約を整理し、`record` を source of truth とする既存 model を保ったまま、human-operated local debug を楽にするための標準 artifact layout を既定導線として固定する。

この plan では、既存の replay/debug 機能を壊さずに以下を揃える。

- `record` / `history` / `snapshot` / `exported snapshot` の責務境界
- input/output flag naming の一貫性
- default 値付き `--output-dir` を使った標準 artifact 配置
- local verification で「まず何を見ればよいか」が CLI と examples から自然に分かる状態

親 plan:

- `docs/exec-plan/done/platform-phase2-implementation.md`

depends on:

- `platform-phase2-03-replay-debug.md`

## Scope

- `arena-runner` の artifact CLI contract 再設計
- default 値付き `--output-dir` の導入
- source-of-truth record と derived artifact の標準出力レイアウト定義
- replay/debug input flag の naming / help / examples 整理
- `docs/specs/platform.md` の artifact hierarchy / CLI examples 更新
- black-box verification と local human verification の更新

この plan では以下は扱わない。

- DB / object storage など本番 persistence backend
- replay/debug の新しい意味論追加
- AI process memory continuity の保証
- 観戦 UI replay player
- game 固有 spec の変更

## Design Direction

この plan では、既存の「persisted final record が source of truth、history / snapshot は derived entrypoint」という model を維持したまま、runner の surface を `output-dir` 中心に再編する。

固定する契約:

- persisted final match record は引き続き source of truth とする
- `history` / `snapshot` / `exported snapshot` は derived artifact とする
- replay/debug の第一入口は `record` とする
- local 人間運用では default 値を持つ `--output-dir <dir>` を基本導線とし、未指定時も `arena-runner` からの相対既定パスに標準レイアウトで artifact を配置する
- structured log は従来どおり `stdout` に出し続け、標準 artifact として `structured-log.ndjson` にも同等内容を保存する
- 個別 output flag は compatibility / explicit extra-output path として整理し、`--output-dir` と同時指定された場合は競合ではなく追加で両方へ出力する

標準レイアウト案:

```text
<output-dir>/
  record.json
  structured-log.ndjson
  snapshot.json
  exported-snapshot.json
  event-log.json
```

ここでの責務:

- `record.json`: source-of-truth final match-record artifact
- `structured-log.ndjson`: `stdout` に流す structured log と同等内容の保存先
- `snapshot.json`: debug 用の derived snapshot
- `exported-snapshot.json`: 公開/debug 用の exported snapshot
- `event-log.json`: record の `event_log` から抽出した derived history artifact

## Spec Changes

### `docs/specs/platform.md`

- `arena-runner` artifact contract を default 値付き `output-dir` 前提で更新する
- source-of-truth artifact と derived artifact の hierarchy を CLI naming と examples まで含めて明文化する
- `record` を replay/debug の primary entrypoint として再確認する
- `history-input` / `snapshot-input` の位置付けを「補助 entrypoint」として明記する
- `target-turn` の命名見直しを含め、replay/resume boundary の意味が help text だけでも読めるようにする
- `output-dir` の既定相対パス、明示指定時の切り替え、`stdout` / `output-dir` / 個別 output flags の併用ルールを定義する
- local debug 用 examples を追加し、「record を保存して、その record から event-log / snapshot を辿る」導線を明記する

### `docs/specs/janken-game.md`

- 原則変更しない
- `arena-runner` examples が `janken` verification に触れる場合のみ、platform contract 参照の文脈調整に留める

## Expected Code Changes

- `cmd/arena-runner/main.go`
- `internal/platform/replay/` または artifact helper 周辺
- artifact path / extraction helper
- runner usage/help text
- black-box test fixture / golden data

必要なら以下も含める。

- `record` から `event_log` / `snapshot` / `exported_snapshot` を抽出して `output-dir` に配置する adapter
- old flag 名から新 contract への compatibility layer

## Verification

完了は CLI 実行と機械 verification で判定する。最低限、以下を確認できること。

- `--output-dir` 未指定時でも、`arena-runner` からの相対既定パスに標準 artifact 一式が期待どおりに生成される
- `--output-dir <dir>` 指定で、標準 artifact の出力先だけを期待どおりに切り替えられる
- `record.json` が source-of-truth final record として replay/debug に再入力できる
- `event-log.json` / `snapshot.json` / `exported-snapshot.json` が `record.json` と整合した derived artifact として出力される
- `structured-log.ndjson` が `stdout` に流れる structured log と同等内容を保持し、進行中観測ログの保存先として継続して読める
- replay/debug の通常導線が `record` 起点であることを help / examples / tests が示している
- compatibility path として残す個別 flags が `output-dir` と同時指定されても競合せず、標準 artifact に加えて個別 path にも期待どおり出力される
- local human verification として、1 回の runner 実行後に artifact directory を見るだけで次の debug 操作に進める

## Sub-tasks

- [ ] Update `docs/specs/platform.md` to define `output-dir`-centered artifact contract and naming
- [ ] Decide final flag surface, including `record` primary entrypoint naming and replay boundary naming
- [ ] Implement defaulted `--output-dir` artifact layout and extraction flow from final record
- [ ] Keep or adapt per-file output flags as compatibility/extra-output paths with explicit dual-output rules
- [ ] Update runner help text and representative CLI examples for local human verification
- [ ] Add black-box verification for fresh run, record replay, history replay, and snapshot start under the new layout

## Parallelism

- spec 更新と final flag surface 決定は blocking
- defaulted `output-dir` artifact writer と compatibility flag 整理は contract 固定後に並行で進められる
- black-box verification 追加と CLI examples 整理は実装後に並行で進められる

## Design Decisions

- 新規 ADR は原則不要
- ただし `output-dir` を Phase 2 の恒久 primary UX として固定し、個別 flags を将来的に縮退対象へ置く判断まで含める場合は `docs/design-decisions/adr.md` 追記を検討する

## Risks and Mitigations

- `output-dir` 導入で CLI surface が広がりすぎる
  - mitigation: source-of-truth を `record` 1 個に固定し、その他は derived artifact として扱う
- `stdout` / 既定 `output-dir` / 個別 flags の同時存在で挙動が読みにくくなる
  - mitigation: spec と help に「標準出力は維持しつつ、artifact は常に既定 `output-dir` に保存され、個別 flags は追加出力先として併用できる」ことを明記する
- `history` が source-of-truth のように誤読される
  - mitigation: examples と help を `record` 起点に寄せ、`history-input` は補助導線に限定する
- replay/debug 実装そのものの意味論変更まで巻き込むと scope creep する
  - mitigation: この plan では artifact contract と operator UX に限定し、continuation semantics は既存 spec を維持する
