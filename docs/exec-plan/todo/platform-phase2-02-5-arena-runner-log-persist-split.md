# platform-phase2-02-5-arena-runner-log-persist-split
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`arena-runner` の観測用逐次出力と、後段で保存・再利用する match record artifact を分離する。

この plan では、試合進行中に追える structured log と、replay/debug の入力として扱える永続化データの責務境界を先に固定する。これにより `platform-phase2-03-replay-debug.md` は「どの file を replay/debug 入力として読むか」が曖昧なまま CLI を増やさずに済む。

親 plan:

- `docs/exec-plan/done/platform-phase2-implementation.md`

depends on:

- `platform-phase2-02-fixture-e2e.md`

should be completed before:

- `platform-phase2-03-replay-debug.md`

## Why Now

`platform-phase2-02-fixture-e2e.md` で `arena-runner` の happy-path / failure-path は black-box で成立したが、出力契約はまだ「終了時に final record 全体を 1 つの JSON として `stdout` に出す」に寄っている。

このまま `platform-phase2-03-replay-debug.md` に進むと、以下が混線する。

- 進行中観測のための log
- replay/debug 入力として残す snapshot / history を含む persist artifact
- CLI の標準出力として人や他ツールが受け取る stream

先に runner 出力面を整理しておくことで、replay/debug plan は「保存済み artifact をどう読むか」に集中できる。

## Scope

- `arena-runner` の output contract 整理
- structured log の逐次出力
- final match record の persist 先分離
- terminal status 時の snapshot / exported snapshot / failure summary の log exposure
- local CLI 向け persist file option と black-box verification

この plan では以下は扱わない。

- turn ごとの incremental persistence
- history からの resume / replay 実行
- DB / object storage など本番用 persistence backend
- AI process memory を含む完全 continuation
- 観戦 UI 向け replay player

## Design Direction

### Options Considered

#### Option A: final record は `stdout` のまま、逐次 log を `stderr` に足す

利点:

- 既存 CLI の machine-readable surface を壊しにくい
- persist file を必須にしなくてよい

欠点:

- `stdout` 単発巨大 JSON と「保存対象 artifact」の責務が分離されない
- `platform-phase2-03-replay-debug.md` が `stdout` dump 前提のまま進み、保存境界が曖昧なままになる

#### Option B: structured log を stream として出し、final record は file persist を主契約にする

利点:

- 観測用 stream と replay/debug 用 artifact を明確に分離できる
- 中断・失敗時も log と persisted artifact の責務を独立に扱える
- 将来 DB / object storage へ差し替える時も contract を保ちやすい

欠点:

- CLI option と verification の更新が必要
- 既存の「stdout をそのまま final JSON として受ける」使い方は見直しが必要

#### Option C: `stdout` を NDJSON 多重 stream にして log と final record を同居させる

利点:

- 出力先を 1 本にまとめられる
- pipe で収集しやすい

欠点:

- 観測 log と persist artifact の責務分離が不十分
- consumer 側が record kind で demux し続ける必要があり、今回の issue の目的に逆行する

### Recommended Approach

この plan では **Option B** を採用する。

理由:

- この issue の主題は「巨大単発 JSON をやめる」ことではなく、「log と persist data の責務を分ける」ことにある
- `platform-phase2-03-replay-debug.md` は persisted snapshot/history を読む plan なので、入力元を file artifact として先に固定した方が整合的
- Phase 2 の現在地では storage backend を増やすより、local CLI で file persist を選べるようにする方が最小コストで効果が高い

## Spec Changes

### `docs/specs/platform.md`

- `arena-runner` の output contract を更新する
- structured log の目的を「進行中観測」として定義する
- final match record の目的を「persist artifact」として定義する
- `stdout` / `stderr` / file output の責務分担を明記する
- structured log record の最低 shape を定義する
  - `match_id`
  - `seq` または stable ordering field
  - `kind`
  - `turn`
  - `player_id` if applicable
  - JSON payload
- match start / event / terminal snapshot / terminal exported snapshot / terminal summary の log record kinds を定義する
- `failed` / `canceled` 時も、可能なら terminal snapshot / exported snapshot を log に出すことを定義する
- final persisted record に snapshot / exported snapshot / event log / stderr summary が含まれることを再確認する
- replay/debug plan が読むのは log stream ではなく persisted artifact であることを明記する

## Expected Code Changes

- `cmd/arena-runner/` の output flag と writer 構成
- structured log emitter
- final record persist writer
- CLI 実行例と verification 用 testdata / golden assertion の整理

必要なら以下も含める。

- runner event を log record へ写像する小さな adapter
- file persist 未指定時の扱いを明示する usage / error path

## Verification

完了は CLI 実行と black-box verification で判定する。最低限、以下を機械的に確認できること。

- match 開始時に metadata を含む structured log が出る
- 試合進行中 event が逐次 log として観測できる
- `completed` / `failed` / `canceled` の各 terminal path で terminal summary log が出る
- terminal 時に snapshot / exported snapshot を log と persisted record の両方で辿れる
- final match record を file に保存できる
- replay/debug 向け入力として読むべき artifact が persisted file だと分かる CLI/example になっている
- log stream と persisted record が同じ出力先に混在しない

## Sub-tasks

- [ ] Update `docs/specs/platform.md` for runner log/persist contract
- [ ] Design CLI flags and default behavior for persisted record output
- [ ] Implement structured log emission for match start, per-event progress, and terminal summaries
- [ ] Implement final record persistence to a separate file target
- [ ] Add black-box verification for completed, failed, and canceled output behavior
- [ ] Capture representative `arena-runner` command examples for log observation and artifact persistence
- [ ] Update follow-up plan dependencies so replay/debug reads persisted artifacts instead of assuming stdout-only output

## Parallelism

- spec 更新と CLI surface 設計は先に固める必要があり blocking
- structured log emission と persist writer は contract 固定後に並行で進められる
- completed / failed / canceled の black-box verification は output contract 実装後に分担できる

## Risks and Mitigations

- Risk: local CLI の既存利用が `stdout` final JSON 前提で壊れる
  - Mitigation: spec と usage で migration path を明示し、persist file を runner の正式 artifact として固定する
- Risk: log record kinds を増やしすぎて replay/debug 本体より重くなる
  - Mitigation: Phase 2 では start / event / terminal summary / terminal snapshot 系の最小集合に留める
- Risk: replay/debug plan と保存データ shape の責務が再び混線する
  - Mitigation: この plan では output contract と local file persist までに限定し、resume semantics は `platform-phase2-03-replay-debug.md` に残す
