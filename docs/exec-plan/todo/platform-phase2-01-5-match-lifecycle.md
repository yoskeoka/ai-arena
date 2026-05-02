# platform-phase2-01-5-match-lifecycle
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`platform-phase2-02-fixture-e2e.md` と `platform-phase2-03-replay-debug.md` で black-box behavior を固定する前に、platform foundation の match lifecycle と session/runtime 境界を、実運用の失敗・キャンセル・逐次進行に耐えられる形へ整える。

この plan は `platform-phase2-01-foundation.md` の後続であり、PR #108 の foundation 実装を前提にする。

## Why Now

`phase2-02` は fixture e2e により runner の外部挙動を固定する。現状の `internal/platform/match.Run` は foundation としては動くが、以下の暫定実装を e2e で固定すると、後続の正しい lifecycle 実装が不要に壊しにくくなる。

- `Run` が init / decision loop / game_over / snapshot / record 作成を1つの直線的な関数に持っている
- 途中失敗時に partial record、event log、stderr summary、session shutdown を一貫して残す責務がない
- sequential mode が「事前に渡された request list を順に処理する」形で、game master が各 player の結果を反映してから次 request を生成する契約を十分に表せていない
- runtime / session の失敗を match status と lifecycle event に写像する境界が薄い

先に match lifecycle の骨格を整えてから e2e を載せることで、fixture e2e は暫定実装ではなく維持したい platform contract を固定できる。

## Scope

- `match.Run` の lifecycle phase / step 分解
- sequential contract の修正
- 失敗時 partial record / event log / snapshot の最小生成
- session shutdown / game_over / runtime failure の lifecycle event 記録
- 現実の複雑性で破綻しやすい foundation 実装のうち、`phase2-02` 前に直すべきものの整理

この plan では以下は扱わない。

- `echo-count` fixture game 実装
- `arena-runner` の black-box e2e
- replay / resume CLI
- WASM runtime 統合
- 本番用 scheduler / worker pool / queueing system
- 完全な actor model 化

## Findings From Foundation Review

### Must Fix Before `phase2-02`

- `internal/platform/match.Run`
  - lifecycle state が暗黙で、`completed` / `failed` / `canceled` を一貫して扱えない
  - `return Record{}, err` が多く、途中失敗時の record と event log が失われる
  - cleanup が `Run` の正常終了 path に寄っており、失敗時の `game_over` / shutdown / stderr capture が曖昧
  - sequential mode の契約が game master 主導の逐次進行を十分に表せていない

- `internal/platform/game.Master`
  - `NextDecision` が `DecisionWindow` 全体を返すため、sequential の「1 player の結果を反映して次 player の request を決める」モデルを自然に表現しづらい
  - match completion と failure/cancel の区別が interface 上にない

- `internal/platform/session.Session`
  - request 送信失敗、transport close、malformed response、mismatched id などがすべて似た `Result` に潰れやすい
  - late response は扱えるが、match lifecycle 上の event として記録する境界がまだ弱い

- `internal/platform/runtime.Adapter`
  - subprocess の exit / crash / forced kill が match lifecycle event へ伝播する contract が弱い
  - stderr capture はあるが、失敗時 record に必ず反映する責務が match 側にない

### Defer Beyond This Plan

- runtime adapter の pool 化、queue 化、backpressure の本格設計
- stderr 本文の保存 backend や公開 API
- WASM 固有の resource limit 実装
- replay / resume 用の history storage 最適化

## Design Direction

### Recommended Approach

大きな actor model には進まず、match runner 内部に明示的な lifecycle phase と step boundary を導入する。

理由:

- Phase 2 の主目的は platform core の検証であり、まだ本番 scheduler を作る段階ではない
- Go 採用 ADR は goroutine / context による締切管理を利点としているが、ここで過剰な concurrency framework を導入する必要はない
- `Correctness over Speed` に従い、e2e 前に失敗時の record / cleanup / event を正しく残せる構造を優先する

### Rejected For This Plan

- 全 player session を actor として常駐させ、match loop と channel で完全分離する
  - 将来の負荷対策には候補だが、Phase 2 fixture 前の修正としては過剰
- 現状の直線的 `Run` を維持して `phase2-02` の e2e で守る
  - e2e が暫定構造を固定し、後続 refactor のコストを上げる

## Spec Changes

### `docs/specs/platform.md`

- match lifecycle phase を定義する
  - `starting`
  - `initializing`
  - `running`
  - `finishing`
  - `completed`
  - `failed`
  - `canceled`
- match record の `status` が `completed` だけでなく `failed` / `canceled` を取りうることを定義する
- failure 時も partial event log / snapshot / stderr byte summary を残すことを定義する
- sequential mode を、game master が各 player の outcome を反映してから次 request を生成する契約として定義する
- lifecycle event kind を追加または整理する
  - `match_failed`
  - `match_canceled`
  - `session_shutdown_started`
  - `session_shutdown_completed`
  - `session_shutdown_failed`
  - `runtime_exited`
- `game_over` notification は正常完了 path の final notification であり、failure / canceled path では cleanup event と区別して扱うことを明記する

## Expected Code Changes

### `internal/platform/game/`

- `Master` interface を sequential-friendly に修正する
- 候補:
  - `NextDecision(ctx)` は次に必要な decision step を返す
  - `ApplyDecision(ctx, step, outcome)` は1 step ごとに状態を反映する
  - simultaneous は同一 step group として複数 request を返せる
- match completion / failure / canceled を record 作成に必要な形で取得できるようにする

### `internal/platform/match/`

- `Runner.Run` を lifecycle orchestration に限定し、以下の helper に分解する
  - `start`
  - `initializeSessions`
  - `runDecisionLoop`
  - `runSimultaneousStep`
  - `runSequentialStep`
  - `finish`
  - `fail`
  - `shutdownSessions`
  - `buildRecord`
- `Runner` に current phase / status / terminal error / cancellation reason を持たせる
- すべての terminal path で record を返せるようにする
- `defer` で session cleanup と final record materialization を保証する
- failure/cancel の event log と snapshot を unit test で確認する

### `internal/platform/session/`

- `Result` に session/transport の失敗種別をより明確に載せる
- `GameOver` / `Close` の結果を match lifecycle event に残せる shape にする
- malformed / mismatched / timeout / late response の event 境界を match 側が安定して扱えるようにする

### `internal/platform/runtime/`

- subprocess exit / crash / forced kill を session/match に伝えられる error shape を整理する
- cleanup 時の forced kill と通常の crash を区別できる test を追加する

## Verification

- `go test ./...`
- `go test -race ./internal/platform/match ./internal/platform/session ./internal/platform/runtime`
- unit test で以下を確認する
  - normal completed path が `completed` record を返す
  - init failure でも `failed` record と partial event log が返る
  - decision loop failure でも snapshot / stderr byte summary が残る
  - context cancellation が `canceled` record として残る
  - sequential mode で game master が player outcome 反映後に次 request を生成する
  - simultaneous mode で request collection と record mutation が分離され、race detector が通る
  - game_over failure / shutdown failure が event log に残る

## Sub-tasks

- [ ] Update `docs/specs/platform.md` for match lifecycle phases, terminal statuses, and sequential contract
- [ ] Redesign `internal/platform/game.Master` around decision steps that support true sequential progression
- [ ] Refactor `internal/platform/match.Run` into lifecycle phase / step helpers
- [ ] Add partial record generation for failed and canceled matches
- [ ] Add cleanup event handling for game_over and session shutdown outcomes
- [ ] Tighten session/runtime failure propagation into match lifecycle events
- [ ] Add unit tests for completed, failed, canceled, sequential, simultaneous, and cleanup-failure paths
- [ ] Run `go test ./...` and targeted `go test -race`

## Parallelism

- `docs/specs/platform.md` lifecycle update and code interface design should happen first and are blocking
- session/runtime failure-shape tests can be implemented in parallel after the lifecycle event contract is fixed
- match runner normal/failure/cancel tests can be split by path after `Runner` phase structure exists

## Risks and Mitigations

- Risk: 大きな actor model に寄せすぎて `phase2-02` 前の plan が膨らむ
  - Mitigation: この plan は runner 内部の lifecycle 明示化と cleanup/record 保証に限定する
- Risk: game master interface 変更が `phase2-02` の fixture 実装と衝突する
  - Mitigation: fixture e2e 前にこの plan を完了し、fixture は新 contract の最初の利用者にする
- Risk: failure record の shape を細かく決めすぎて replay/debug の自由度を下げる
  - Mitigation: この plan では terminal status, event log, snapshot, stderr byte summary の存在だけを固定し、storage/replay 詳細は `phase2-03` に残す

## Dependencies

- Depends on `platform-phase2-01-foundation.md` being completed and merged. In the current workflow this is represented by PR #108.
- Should be completed before `platform-phase2-02-fixture-e2e.md`.
