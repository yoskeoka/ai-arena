# arena-runner e2e が init failure で `status = "failed"` になる

## Summary

`spec-responsibility-wording` の docs-only execution branch で `make test` を実行したところ、
`e2e/arena_runner_test.go` の複数ケースが `status = "failed"` で失敗した。

今回確認した failure は以下:

- `TestArenaRunnerHappyPaths/simultaneous`
- `TestArenaRunnerFailurePaths/timeout`
- `TestArenaRunnerFailurePaths/bad-json`
- `TestArenaRunnerFailurePaths/mismatched-id`
- `TestArenaRunnerJankenTimeoutAndInvalidAffectPlacement`

失敗ログでは `match_failed` と `init failed for p1: invalid-timeout` が出ており、
docs 変更だけでは説明できない runner / test baseline 側の regression が疑われる。

## Captured Output

実行コマンド:

```sh
rtk make test
```

代表的な失敗要約:

```text
--- FAIL: TestArenaRunnerHappyPaths (12.65s)
    --- FAIL: TestArenaRunnerHappyPaths/simultaneous (3.94s)
        arena_runner_test.go:121: status = "failed", want completed
--- FAIL: TestArenaRunnerFailurePaths (23.84s)
    --- FAIL: TestArenaRunnerFailurePaths/timeout (2.87s)
        arena_runner_test.go:268: status = "failed", want "completed"
    --- FAIL: TestArenaRunnerFailurePaths/bad-json (3.59s)
        arena_runner_test.go:268: status = "failed", want "completed"
    --- FAIL: TestArenaRunnerFailurePaths/mismatched-id (3.72s)
        arena_runner_test.go:268: status = "failed", want "completed"
--- FAIL: TestArenaRunnerJankenTimeoutAndInvalidAffectPlacement (3.09s)
    arena_runner_test.go:641: event log missing invalid action failure
FAIL
FAIL    github.com/yoskeoka/ai-arena/e2e    66.387s
```

`TestArenaRunnerJankenTimeoutAndInvalidAffectPlacement` では、失敗時の event log から少なくとも以下を確認した。

```text
{Seq:1 Kind:match_started Turn:0 PlayerID: ...}
{Seq:2 Kind:game_master_initialized Turn:0 PlayerID: ...}
{Seq:3 Kind:session_initialized Turn:0 PlayerID:p1 Payload:{"Status":"no_action","FailureReason":"invalid-timeout","Payload":null,"IgnoredLateResponseIDs":null}}
{Seq:4 Kind:session_shutdown_started Turn:0 PlayerID:p1 Payload:{"phase":"failed"}}
{Seq:5 Kind:session_shutdown_completed Turn:0 PlayerID:p1 Payload:{"phase":"failed"}}
{Seq:9 Kind:runtime_exited Turn:0 PlayerID:p3 Payload:{"error":"context deadline exceeded","stage":"shutdown"}}
{Seq:10 Kind:session_shutdown_failed Turn:0 PlayerID:p3 Payload:{"error":"context deadline exceeded","stage":"close"}}
{Seq:13 Kind:match_failed Turn:0 PlayerID: Payload:{"error":"init failed for p1: invalid-timeout"}}
```

この branch では、期待されていた `invalid-illegal-action` 系の failure 記録へ進む前に、
`p1` の init failure によって試合全体が `match_failed` へ落ちている。

完全な実行ログはローカル保存済み:

```text
~/.local/share/rtk/tee/1778525955_make_test.log
```

## Impact

- docs-only PR でも `make test` を green にできず、Step 3 execution PR の通常 verification を満たせない。
- `e2e/arena_runner_test.go` の happy path と failure path の両方が巻き込まれており、
  `arena-runner` の init/session lifecycle か test fixture のどちらかに横断的な問題がある可能性が高い。
- `docs/issues/done/arena-runner-e2e-status-failed-baseline.md` は「その時点では main で再現しなかった」
  履歴なので、今回の failure は別 issue として追跡する必要がある。
- 再現が偶発的でも、少なくとも今回観測した assertion と event log 断片はこの issue から再参照できる。

## Follow-up

- `main` で `make test` と `go test ./e2e -run 'TestArenaRunnerHappyPaths|TestArenaRunnerFailurePaths|TestArenaRunnerJankenTimeoutAndInvalidAffectPlacement' -count=1`
  を再実行し、baseline 由来か branch/worktree 固有かを切り分ける。
- `simultaneous` と `janken` failure について `result-summary.json` / `record.json` / `structured-log.ndjson`
  を採取し、どの init path で `invalid-timeout` が入るかを確認する。
- runner の init/session 実装 regression か、fixture/test 期待値 drift かを切り分け、必要なら別 exec-plan を作る。
