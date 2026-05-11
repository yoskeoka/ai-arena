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

## Impact

- docs-only PR でも `make test` を green にできず、Step 3 execution PR の通常 verification を満たせない。
- `e2e/arena_runner_test.go` の happy path と failure path の両方が巻き込まれており、
  `arena-runner` の init/session lifecycle か test fixture のどちらかに横断的な問題がある可能性が高い。
- `docs/issues/done/arena-runner-e2e-status-failed-baseline.md` は「その時点では main で再現しなかった」
  履歴なので、今回の failure は別 issue として追跡する必要がある。

## Follow-up

- `main` で `make test` と `go test ./e2e -run 'TestArenaRunnerHappyPaths|TestArenaRunnerFailurePaths|TestArenaRunnerJankenTimeoutAndInvalidAffectPlacement' -count=1`
  を再実行し、baseline 由来か branch/worktree 固有かを切り分ける。
- `simultaneous` と `janken` failure について `result-summary.json` / `record.json` / `structured-log.ndjson`
  を採取し、どの init path で `invalid-timeout` が入るかを確認する。
- runner の init/session 実装 regression か、fixture/test 期待値 drift かを切り分け、必要なら別 exec-plan を作る。
