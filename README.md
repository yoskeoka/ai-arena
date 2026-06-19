# AI Arena
An online game where human-built AIs compete under strict, fast turn limits while spectators watch matches unfold live.
The value is a fair, exciting benchmark for real engineering skill (not prompt-only play), because success depends on robust algorithms, clear trade-offs, and observable behavior.

## Local runtime commands

For contributor setup, local secrets, and manual GitHub login verification, see [DEVELOPMENT.md](./DEVELOPMENT.md).

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

### Run `arena-service`

Use `arena-service` when you want the local online-service shape: operator HTTP API, queue store, and in-process worker loop.

For deploy-shaped local verification, start the Docker Postgres harness first, bootstrap the SeaweedFS object-storage harness, and then launch the service against both backends:

```sh
make postgres-up
make postgres-schema-apply
make seaweed-bootstrap
make render-build
ARENA_SERVICE_POSTGRES_DSN=postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable \
ARENA_SERVICE_PRESET_CONFIG=./config/platform-service/presets.remote-bootstrap.json \
ARENA_SERVICE_ARTIFACT_BACKEND=r2 \
ARENA_SERVICE_ARTIFACT_R2_BUCKET=ai-arena-local \
ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT=http://127.0.0.1:8333 \
ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID=admin \
ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY=secret \
PORT=10000 \
make render-start
```

This starts `arena-service serve` on `http://127.0.0.1:10000`.
For the current route surface and payload contract, use the specs under `docs/specs/` instead of treating this README as an endpoint catalog.

Before opening the frontend, verify the backend is actually up:

```sh
curl http://127.0.0.1:10000/healthz
```

This should return `200 OK`.

The default local Postgres harness DSN is:

```text
postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable
```

The Docker Postgres harness definition lives in `docs/development/platform-service-postgres.md`.
The SeaweedFS object-storage harness and delegated artifact verification flow live in `docs/development/platform-service-object-storage.md`.
`./config/platform-service/presets.remote-bootstrap.json` is the deploy-shaped preset catalog.
`./config/platform-service/presets.example.json` remains the lightweight contributor example and is not the staging/production canonical value.
When you are done with the local backends, stop them with:

```sh
make postgres-down
make seaweed-down
```

If you explicitly want the lightweight queue-only lane instead of the deploy-shaped Postgres lane, omit `ARENA_SERVICE_POSTGRES_DSN` and start `arena-service` with the in-memory store.
If you want the lightweight artifact lane as well, omit `ARENA_SERVICE_ARTIFACT_BACKEND` and the service keeps the local filesystem backend.
In that lightweight lane, it is fine to point `ARENA_SERVICE_PRESET_CONFIG` at `./config/platform-service/presets.example.json`.

The equivalent direct command is:

```sh
ARENA_SERVICE_POSTGRES_DSN=postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable \
ARENA_SERVICE_ARTIFACT_BACKEND=r2 \
ARENA_SERVICE_ARTIFACT_R2_BUCKET=ai-arena-local \
ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT=http://127.0.0.1:8333 \
ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID=admin \
ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY=secret \
go run ./cmd/arena-service serve \
  --listen-addr 0.0.0.0:10000 \
  --preset-config ./config/platform-service/presets.remote-bootstrap.json
```

### Run the operator UI

To verify the UI added in this task, start the local frontend dev server in a second terminal:

```sh
cd operator-ui
pnpm install
pnpm run dev
```

Then open the local Vite URL printed by the command, usually `http://127.0.0.1:5173`.
Keep `arena-service` running while the UI is open so the polling panels and preset queue actions can talk to the local API.
For local development, the Vite dev server proxies `/api` and `/healthz` to `http://127.0.0.1:10000`, so you can leave the UI's base URL field blank.

For the repo-owned local browser verification lane, use the Playwright harness instead of a manual browser loop:

```sh
cd operator-ui
pnpm run verify:local
```

This command self-bootstraps missing `operator-ui` dependencies and the local Playwright Chromium browser,
then starts a Go fixture backend, starts the local Vite frontend, and verifies
the preset queue / active matches / completed detail / artifact access surface automatically.
The detailed runbook lives in `docs/development/operator-ui-local-verification.md`.
If Playwright still fails after browser bootstrap, check the runbook for host-library troubleshooting.

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
