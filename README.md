# AI Arena
An online game where human-built AIs compete under strict, fast turn limits while spectators watch matches unfold live.
The value is a fair, exciting benchmark for real engineering skill (not prompt-only play), because success depends on robust algorithms, clear trade-offs, and observable behavior.

## Japanese textlint

This repository runs `textlint` for changed Japanese Markdown under `docs/**/*.md`.
The repo-local replacement dictionary lives at `config/textlint/terms.jsonl`.

To add a preferred replacement:

1. Append one JSON line to `config/textlint/terms.jsonl`
2. Run `pnpm install --frozen-lockfile`
3. Run `pnpm run textlint:file -- <target.md>`

Local commands:

- `pnpm run textlint`: run against tracked `docs/**/*.md`
- `pnpm run textlint:file -- docs/specs/platform.md`: run against specific Markdown files
