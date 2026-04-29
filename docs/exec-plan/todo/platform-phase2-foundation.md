# platform-phase2-foundation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/specs/platform.md` の Phase 2 foundation を実装し、`echo-count` や `janken` の前提になる platform core を unit test 中心で成立させる。

親 plan:

- `platform-phase2-implementation.md`。このファイルは `docs/exec-plan/todo/` または `docs/exec-plan/done/` に存在しうる

## Scope

- `docs/specs/platform.md` の foundation scope 更新
- JSON-RPC 2.0 + NDJSON protocol
- `game_id` / `game_version` / `ruleset_version` metadata 契約
- local subprocess runtime adapter
- AI session (`init` / `turn` / `game_over`)
- simultaneous / sequential を扱う match loop
- match record / event log / snapshot/exported snapshot の最小形

この plan では以下は扱わない。

- `echo-count` fixture game 実装
- fixture AI 群
- black-box e2e
- `janken` 実装
- start-from-snapshot / resume-from-history-and-continue の CLI entrypoint

## Spec Changes

### `docs/specs/platform.md`

- Phase 2a を local subprocess foundation として明記する
- match record / event log / snapshot / exported snapshot の最小 schema を定義する
- `accepted` / `no_action` と failure reason の分離を定義する
- `invalid-timeout`
- `invalid-protocol-malformed`
- `invalid-protocol-mismatched-id`
- `invalid-illegal-action`
- `invalid-protocol-late-response`
  の最低分類を定義する
- `game_id` / `game_version major` / `ruleset_version` の責務差分を定義する
- AI metadata の sidecar manifest 既定を定義する

### `docs/specs/janken-game.md`

- この plan では変更しない。`janken` 側の責務は `platform-phase2-janken-integration.md` へ送る

## Expected Code Changes

- `go.mod`
- `cmd/arena-runner/` の最小 bootstrap
- `internal/platform/protocol/`
- `internal/platform/catalog/`
- `internal/platform/runtime/`
- `internal/platform/session/`
- `internal/platform/match/`
- `internal/platform/game/`

package 分割は execution で調整してよいが、以下は維持する。

- protocol と game logic を分離する
- runtime adapter と session / match loop を分離する
- platform failure と game validation failure を record 上で区別する

## Verification

完了は unit test で判定する。最低限、以下を機械的に確認できること。

- `go test ./...` が foundation package 群で通る
- valid request/response encode-decode
- NDJSON framing
- malformed JSON / invalid envelope / mismatched id 判定
- metadata validation と `game_version major` compatibility 判定
- subprocess 起動 / stream 接続 / stderr capture / shutdown timeout
- `init` ACK / `turn` timeout / `game_over` notification
- late response が後続 turn と混線しない
- simultaneous / sequential scheduler
- record / event log / snapshot の最小整合

## Sub-tasks

- [ ] Update `docs/specs/platform.md` for Phase 2 foundation
- [ ] Define metadata and compatibility rules
- [ ] [parallel] Implement protocol package and tests
- [ ] [parallel] Define runtime / session / match interfaces
- [ ] [depends on: Implement protocol package and tests, Define runtime / session / match interfaces] Implement local subprocess runtime adapter
- [ ] [depends on: Implement local subprocess runtime adapter] Implement AI session
- [ ] [depends on: Implement AI session] Implement match loop and game master contract
- [ ] [depends on: Implement match loop and game master contract] Implement record / event log / snapshot foundation
- [ ] Run unit test verification and capture the package-level results

## Parallelism

- protocol package と runtime/session interface 定義は独立して進められる
- record foundation は match contract 固定後に game 実装と切り離して進められる

## Risks and Mitigations

- foundation に replay まで入れると責務が膨らむ
  - mitigation: replay/debug は別 plan に送る
- `janken` 実装まで同時に入れると platform 不具合の切り分けが難しくなる
  - mitigation: game 非依存 core の unit verification を先に閉じる
