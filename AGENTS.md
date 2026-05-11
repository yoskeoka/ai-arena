# AI Arena Agent Instructions

## Documentation Language Policy

- Internal documents under `docs/*` such as plans, issues, specs, and ADRs should be written in Japanese.
- This policy applies only to internal repository documentation.

## Language Boundaries

- Comments in implementation code should be written in English.
- Commit messages should be written in English.
- PR titles and PR descriptions should be written in English.
- The language of product-generated UI and messages is not affected by the `docs/*` language policy.

## Notes

- Do not treat this file as the final authority for public UI, public API, or external documentation language policy.
- See `docs/design-decisions/adr.md` for the background and decision history behind the internal documentation policy.
- When editing GitHub Actions workflows or composite actions, use `pinact` to pin or update `uses:` references rather than hand-editing version tags.

## Observation Discipline

- For `arena-runner` artifacts, default reading order is `result-summary.json` -> `exported-snapshot.json` / `snapshot.json` -> `structured-log.ndjson` / `record.json` / `history.json`.
- Treat `record.json.event_log` and `history.json` as source-of-truth / replay inputs, not the default first artifact for ordinary result inspection.
- When quoting or summarizing dungeon runs for implementation work, prefer compact artifacts first and avoid pasting long per-turn logs unless causal tracing is required.

## Deterministic Golden Discipline

- When an ai-arena PR updates a deterministic regression golden, use the ai-arena local `.github/PULL_REQUEST_TEMPLATE.md` section for that update, check the golden-update box, and explain why the update is allowed.
- The same PR must also make that update reason explicit in the relevant spec and/or exec-plan, rather than leaving the rationale only in the PR body.
