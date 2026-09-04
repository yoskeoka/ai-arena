# arena-service-terminal-record-status-flake

## Summary

PR `#307` の `go-ci / go-test-postgres` follow-up で、failed-jobs の再実行時に
`cmd/arena-service.TestRunOnceFailureStillPrintsTerminalRecord` が失敗した。
この PR は match composer、bot revision pinning、operator UI を変更しており、
`cmd/arena-service/main_test.go` は変更していない。最初の失敗は別の既知 flaky
`0020-session-game-over-status-flake` であり、再実行で本 test に切り替わったため、
現時点では unrelated flaky failure として切り分ける。

## Failure Output

- observed on: `2026-09-05`
- failing workflow: `go-ci / go-test-postgres`
- workflow run:
  `https://github.com/yoskeoka/ai-arena/actions/runs/33684843575`
- representative assertion:

```text
--- FAIL: TestRunOnceFailureStillPrintsTerminalRecord (0.02s)
    main_test.go:334: match_status = "failed", want canceled
FAIL
FAIL    github.com/yoskeoka/ai-arena/cmd/arena-service    0.491s
```

## Proposed Solution

- `cmd/arena-service/main_test.go` の
  `TestRunOnceFailureStillPrintsTerminalRecord` で terminal record を読む前に
  cancel 完了を観測する同期点があるか確認する
- worker loop の cancellation と terminal persistence の順序を明示し、
  `failed` と `canceled` のどちらを assertion する契約かを test 名・fixture と揃える
- targeted stress loop と failure 時の record/event dump を追加し、修正前に
  再現可能な failure mode を固定する

## Priority

Postgres CI の unrelated red を発生させ、実装 PR の landing 判定を不安定にする。
`0020-session-game-over-status-flake` とは失敗 package と assertion が異なるため、
別 issue として追跡する。
