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
