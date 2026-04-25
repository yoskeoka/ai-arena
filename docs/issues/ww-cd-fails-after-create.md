# `ww cd` failed immediately after `ww create`

## Summary

During execution of Phase 1 task `001-game-concept`, the prescribed setup flow:

```sh
ww create --repo ai-arena docs/001-game-concept
cd "$(ww cd --repo ai-arena docs/001-game-concept)"
```

did not work as documented.

## Observed behavior

- `ww create --repo ai-arena docs/001-game-concept` succeeded
- created worktree path:
  `/path/to/.../.worktrees/ai-arena@docs-001-game-concept`
- immediate follow-up `ww cd --repo ai-arena docs/001-game-concept` failed with:
  `no worktree found for branch "docs/001-game-concept"`

## Expected behavior

After successful `ww create`, the matching `ww cd` invocation should resolve the
newly created worktree path for the same repo and branch.

## Impact

- workflow instructions could not be followed literally
- execution continued by using the created path directly
- no code/spec content was blocked, but the standard worktree UX was broken

## Follow-up

Investigate whether `ww cd` expects a different branch key format than
`ww create`, or whether the worktree registration step is incomplete for `docs/*`
branches.
