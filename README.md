# AI Arena
An online game where human-built AIs compete under strict, fast turn limits while spectators watch matches unfold live.
The value is a fair, exciting benchmark for real engineering skill (not prompt-only play), because success depends on robust algorithms, clear trade-offs, and observable behavior.

## Japanese textlint

This repository runs `textlint` for changed Japanese Markdown under `docs/**/*.md`.
The repo-local replacement dictionary lives at `config/textlint/terms.jsonl`.
Use one JSON object per line:

```json
{"pattern":"\\btaxonomy\\b","replacement":"分類"}
```

`pattern` is JavaScript regular-expression source text stored inside JSON.
The example above uses `\\b`, which means "word boundary", so it matches the standalone word `taxonomy` but not `taxonomyMap`.

Common pattern building blocks:

- `\\bword\\b`: match a standalone English word
- `^text$`: match a whole line exactly
- `foo.*bar`: match text from `foo` through the next `bar` on the same line
- `[0-9]`: one digit
- `[A-Za-z0-9_-]+`: one or more ASCII letters, digits, `_`, or `-`
- `\\.` `\\(` `\\)` `\\\\`: match literal `.`, `(`, `)`, and `\`

Notes:

- This dictionary uses JavaScript regex syntax, not shell glob syntax.
- `pattern` is compiled with the global `g` flag by the custom rule.
- Add separate dictionary entries when you need distinct case-sensitive patterns rather than relying on inline flag syntax.

To add a preferred replacement:

1. Append one JSON line to `config/textlint/terms.jsonl`
2. Run `pnpm install --frozen-lockfile`
3. Run `pnpm run textlint:file -- <target.md>`

Local commands:

- `pnpm run textlint`: run against tracked `docs/**/*.md`
- `pnpm run textlint:file -- docs/specs/platform.md`: run against specific Markdown files
