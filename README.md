# AI Arena
An online game where human-built AIs compete under strict, fast turn limits while spectators watch matches unfold live.
The value is a fair, exciting benchmark for real engineering skill (not prompt-only play), because success depends on robust algorithms, clear trade-offs, and observable behavior.

## Local runtime commands

For contributor setup, local secrets, and manual GitHub login verification, see [DEVELOPMENT.md](./DEVELOPMENT.md).

### Installation

```sh
cd operator-ui
pnpm install
```

### Run `arena-runner`

Use `arena-runner` when you want to execute one match directly without the online service layer.
The repo already includes convenience targets for the built-in sample games:

- `make run-echo-simultaneous`
- `make run-echo-sequential`
- `make run-janken-go-wasm`
- `make run-janken-rust-wasm-eval`

Each target writes standard artifacts under a temporary output directory and prints that directory before the run starts.

If you want to call the runner directly instead of using a Make target:

```sh
go run ./cmd/arena-runner \
  --game echo-count \
  --game-version 2.0.0 \
  --ruleset phase2-simultaneous-3turn \
  --match-id local-demo \
  --output-dir ./tmp/arena-runner-output \
  --player p1=./testdata/ai/echo/echo-ai \
  --player p2=./testdata/ai/echo/echo-ai
```

### Run `arena-service` backend

This runs the local ai-arena online service backend.

```sh
make up
make migrate
make seaweed-bootstrap
make start-backend-local
```

This starts `arena-service serve` on `http://127.0.0.1:10000`.

To connect local Postgres, use psql

```sh
psql "postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable"
```

### Stop `arena-service` backend

```sh
make down
```

### Run `arena-service` frontend

```sh
make start-frontend-local
```

### Run automated verification test on local

**Prerequisite**: stop arena-service backend and arena-service frontend before run the verification. This command will run backend and frontend inside.

For local browser verification, use the Playwright harness instead of a manual browser test:

```sh
cd operator-ui
pnpm run verify:local
```

## Release flow

For the Phase 6 online-service lane, the required release path is:

1. Verify locally before opening or updating the PR.
   - backend: start `arena-service` with the deploy-shaped Postgres/R2 harness
   - frontend: verify the operator UI with `pnpm run verify:local:real` or the equivalent local runbook
2. Merge the PR only after the required CI lanes are green.
   - at minimum, confirm `go-ci` and `operator-ui-browser`
3. After the PR is merged, wait for the merged commit SHA to pass the push-triggered CI workflows on `main`.
   - once they are all green, the repo-owned staging deploy and staging verification automation runs for that same SHA
4. Release that same verified commit SHA to production by creating a GitHub Release or otherwise pushing a tag for it.
   - the repo-owned production release automation runs for the tag SHA
   - the tag must point to a commit that is already in `origin/main`

Do not create the production tag unless the automatic staging verification workflow passed for that same SHA.
The detailed dispatch, rollback, and evidence rules live in `docs/development/platform-service-online-deploy.md`.

## Japanese textlint

This repository runs `textlint` for changed Japanese Markdown under `docs/**/*.md`.
The repo-local replacement dictionary lives at `config/textlint/terms.jsonl`.
Use one JSON object per line:

```json
{"pattern":"\\btaxonomy\\b","replacement":"分類"}
```

`pattern` is JavaScript regular-expression source text stored inside JSON.
The example above uses `\\b`, which means "word boundary", so it matches the standalone word `taxonomy` but not `taxonomyMap`.

<details>
<summary>Pattern Details</summary>

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
2. Run `pnpm run textlint:file -- <target.md>`

Local commands:

- `pnpm run textlint`: run against tracked `docs/**/*.md`
- `pnpm run textlint:file -- docs/specs/platform.md`: run against specific Markdown files

Both commands self-bootstrap missing root dependencies before invoking `textlint`.

</details>
