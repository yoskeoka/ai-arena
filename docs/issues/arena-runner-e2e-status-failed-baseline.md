# arena-runner e2e returns `status = "failed"` during baseline verification

## Summary

While verifying the `pinact` rollout branch, `make test` failed in `e2e/arena_runner_test.go` even though this branch changes only workflow/docs files.

Observed failures included:

- `TestArenaRunnerHappyPaths/simultaneous`
- `TestArenaRunnerHappyPaths/sequential`
- multiple `TestArenaRunnerFailurePaths/*` cases
- `TestArenaRunnerStartFromSnapshot`
- `TestArenaRunnerResumeFromHistoryAndContinue`

Each failure reported `status = "failed", want completed`.

## Impact

- The repository-level quality gate is not green on the current baseline.
- The `pinact` rollout branch cannot claim a fully passing `make test` run without a separate fix for arena-runner execution behavior.

## Follow-up

- Reproduce on `main` to confirm whether this is fully baseline and not branch-specific.
- Capture the artifact/status output for one happy-path case and one failure-path case to identify where the runner transitions to `failed`.
- Decide whether the expected status in these tests drifted from the current runner contract, or whether the runner regressed.
