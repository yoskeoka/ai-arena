# phase7-artifact-submission-e2e-staging
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

artifact submission の operated-service acceptance を固定する。local S3-compatible lane と release asset を使い、
upload → validate → registry → queue → materialize → game/AI WASI start を実証する。

## Existing Implementation References

- `tools/dev/package-builtin-game-bundles.sh`: echo/janken game bundle fixtures
- `tools/dev/verify-local-object-storage.go`: local object-storage harness
- `cmd/arena-service/main.go`: service/worker runtime
- `docs/exec-plan/todo/0007-ai-arena-release-artifacts.md` in reversi-ai-arena: first external game release assets

## Code Change Map

- local object-storage harness and CI scripts (MODIFY): bundle upload/staging lane
- built-in and Reversi fixture/release docs (MODIFY): exact bundle invocation and SHA evidence
- integration tests (NEW): filesystem/R2 parity and complete upload-to-match proof
- `docs/specs/platform-artifact-bundle.md` (MODIFY): operated-service acceptance behavior

## Sub-tasks

- [ ] Run the same uploaded bytes through filesystem and local S3-compatible stores.
- [ ] Exercise echo/janken and first Reversi release game/AI bundle pairs.
- [ ] Capture digest, materialization and WASI start evidence in deterministic integration tests.

## Verification

- local S3-compatible `upload -> validate -> materialize -> game/AI start` proof
- CI-safe integration target and workflow lint
