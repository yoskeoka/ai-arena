# `TestArenaRunnerFailurePaths/timeout` intermittently returns `failed` instead of `completed`

## Summary

`make test` during `platform-phase5-06-dungeon-expansion-foundation-02-turn-engine-pipeline` hit an intermittent
failure in `e2e/TestArenaRunnerFailurePaths/timeout`.

## Failure Output

- Command: `make test`
- Date: `2026-05-12`
- Failing test: `TestArenaRunnerFailurePaths/timeout`
- Representative assertion:

```text
--- FAIL: TestArenaRunnerFailurePaths (14.23s)
    --- FAIL: TestArenaRunnerFailurePaths/timeout (3.35s)
        arena_runner_test.go:268: status = "failed", want "completed"
```

## Immediate Follow-up

- Targeted rerun `go test ./e2e -run 'TestArenaRunnerFailurePaths/timeout' -count=1` passed immediately afterward
- Current dungeon turn-engine refactor does not touch `echo-count` game logic directly, so this was treated as an unrelated blocker note

## Questions

- Is the timeout path in `runArena` or `arena-runner` sensitive to timing and occasionally escalates from player timeout to match failure?
- Should this case collect the produced record/log bundle on failure so status transitions can be inspected after the fact?
