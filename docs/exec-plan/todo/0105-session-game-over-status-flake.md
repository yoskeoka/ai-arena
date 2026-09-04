# session-game-over-status-flake
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0020-session-game-over-status-flake.md`

## Objective

subprocess AI が `game_over` への完全な JSON-RPC response を stdout に書き終えた直後に終了しても、platform はその response を `accepted` として観測するようにする。既に output された response を runtime shutdown が先に pipe を閉じたことで失わせ、`no_action` / `runtime-stopped` と誤分類してはならない。

完了境界は、late response を破棄して次の request を受理する既存動作を保持しつつ、最後の `game_over` acknowledgement と即時 subprocess exit の組合せを反復実行しても安定して `accepted` になることとする。process が response を出さずに終了した場合は、従来どおり `runtime-stopped` である。WASM/WASI の transport semantics や game/match lifecycle API は変更しない。

## Cause and Existing Implementation References

- `docs/issues/0020-session-game-over-status-flake.md`
  - CI の 2 回の failure で `TestSessionInitTurnTimeoutGameOverAndLateResponse` が `GameOver status = "no_action"` になった記録、lines 1-44
- `internal/platform/runtime/local_subprocess.go`
  - `startLocalSubprocess` が `readStdout` と同時に `cmd.Wait()` を開始し、`Wait()` の pipe close が decoder より先行し得る、lines 23-71
  - `Close` が stdin close -> grace -> interrupt/kill と `done` を待つ shutdown ownership、lines 88-120
- `internal/platform/session/session.go`
  - `Session.call` は incoming channel close を `runtime-stopped`、matching complete response を `accepted` にする、lines 107-169
- `internal/platform/session/session_test.go`
  - native subprocess fixture は `game_over` response を出した直後に `os.Exit(0)` し、今回の race を露出する、lines 17-87, 203-225
  - WASM/WASI counterpart と early runtime-stop coverage、lines 89-201
- `docs/specs/platform-common-contract.md`
  - stdout close/process exit の `runtime-stopped` 分類と `game_over` request/response contract、lines 95-146
- `docs/specs/platform.md`
  - completed path の `game_over`、すべての terminal path の shutdown contract、lines 341-357

## Black-box Specification Changes

`docs/specs/platform-common-contract.md` の common transport contract を次のように明文化する。

- request に対応する完全な JSON-RPC response が transport close より前に stdout へ出力済みなら、platform は close を理由にその response を破棄せず、通常の response matching/validation に渡す。
- 完全な対応 response を観測する前に stdout close、stdin write failure、または process exit により transport が継続不能になった場合だけ `runtime-stopped` とする。
- この優先順は subprocess と WASM/WASI を含む runtime kind に共通であり、timeout 済み response の late-response 分類と current pending request への非混線は維持する。

wire field、TypeSpec、game 固有 payload、match lifecycle status の変更はない。

## Code Change Map

- `docs/specs/platform-common-contract.md` (MODIFY)
  - emitted response と subsequent runtime close の観測優先順、および `runtime-stopped` の境界を追加する。
- `internal/platform/runtime/local_subprocess.go` (MODIFY)
  - stdout decoder の drain 完了を `cmd.Wait()` による pipe close より先にし、child exit 後もすでに書かれた complete response を incoming channel へ届けてから terminal close を publish する。
  - existing `Close` deadline、exit error、stdin/interrupt/kill handling を保持する。
- `internal/platform/runtime/runtime_test.go` (MODIFY)
  - child が response を stdout に出して即時終了する regression fixture を追加し、response が channel close より前に観測できることを反復確認する。
- `internal/platform/session/session_test.go` (MODIFY)
  - native `Init -> timeout -> late response ignored -> next turn -> game_over` scenario を、final acknowledgement 後の即時 exit でも `accepted` とする end-to-end regression として維持し、failure 時に status/reason を出力する。
  - WASM/WASI counterpart と runtime-stop-before-response coverage を変更しないことを確認する。
- `docs/issues/0020-session-game-over-status-flake.md` (DELETE)
  - implementation verification と PR preparation 後、resolved local issue と matching execution plan を execution PR で削除する。plan PR では残す。
- `docs/exec-plan/todo/0105-session-game-over-status-flake.md` (DELETE)
  - implementation verification と PR preparation 後、matching execution plan を execution PR で削除する。plan PR では残す。

## Sub-tasks

- [ ] spec-first で、complete response と subsequent transport close の観測優先順を common transport contract に追記する。
- [ ] local subprocess adapter の goroutine ownership を整理し、stdout decoder を drain してから process wait/terminal channel close を publish する。stdin close、interrupt、kill、exit error の既存 semantics を regression で保護する。
- [ ] immediate-exit fixture を使う runtime unit regression と session end-to-end regression を追加する。response 未送信の early exit は `runtime-stopped` のままであることを確認する。
- [ ] targeted repeat test と race detector で native session/runtime を検証し、WASM/WASI counterpart を含む package test を実行する。
- [ ] implementation PR で resolved issue と execution plan を削除し、原因、contract change、repeat-test evidence を記録する。

## Dependencies and Parallelism

- depends on: none
- [sequential] common transport contract を先に更新してから runtime lifecycle implementation を変更する。
- [parallel after contract] runtime-level immediate-exit regression と session-level late-response/game-over regression の fixture/assertion は並行して作業できる。
- [sequential] full repeat/race verification は adapter と両 test lane が揃った後に行う。

## Verification

- `GOPATH=/tmp/ai-arena-session-flake/go GOMODCACHE=/tmp/ai-arena-session-flake/go/pkg/mod GOCACHE=/tmp/ai-arena-session-flake/go-build go test ./internal/platform/runtime ./internal/platform/session -run 'Test.*(ImmediateExit|InitTurnTimeoutGameOverAndLateResponse)' -count=100`
- same focused packages with `-race` (repeat count may be reduced only if the command-level timeout requires it; record the actual count)
- `make test` using the repository-owned temporary cache defaults
- `make lint`
- confirm that the native session case reports `accepted` for `game_over` after a late response, the existing WASM/WASI counterpart passes, and an exit before any matching response remains `runtime-stopped`

## Risks and Mitigations

- stdout drain before `Wait()` could delay final process reaping or interact with shutdown cancellation.
  - mitigation: retain the existing `Close` deadline/interrupt/kill path; cover natural exit, clean stdin-close exit, and forced shutdown behavior at the adapter boundary.
- a test-only fixture change could mask the production race.
  - mitigation: add the primary regression at the local subprocess adapter with an immediate-exit child; keep the session test as end-to-end coverage rather than relying on sleeps or polling.
- accepting an old timed-out response after close could corrupt the final request.
  - mitigation: preserve `Session.call` late-ID filtering and explicitly repeat the timeout/late-response scenario in verification.
