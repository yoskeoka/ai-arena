# AI Arena
An online game where human-built AIs compete under strict, fast turn limits while spectators watch matches unfold live.
The value is a fair, exciting benchmark for real engineering skill (not prompt-only play), because success depends on robust algorithms, clear trade-offs, and observable behavior.

## Local runtime commands

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

For the first local bring-up, build the service binary and start it with the example preset catalog:

```sh
make render-build
ARENA_SERVICE_PRESET_CONFIG=./config/platform-service/presets.example.json \
PORT=10000 \
make render-start
```

This starts `arena-service serve` on `http://127.0.0.1:10000` with:

- `GET /healthz`
- `GET /api/v1/matches/active`
- `GET /api/v1/matches/completed`
- `GET /api/v1/matches/{submission_id}`
- `POST /api/v1/preset-matches`

By default, local startup uses the in-memory queue store.
If you want the durable Postgres-backed lane instead, set `ARENA_SERVICE_POSTGRES_DSN` before `make render-start`.

The equivalent direct command is:

```sh
go run ./cmd/arena-service serve \
  --listen-addr 0.0.0.0:10000 \
  --preset-config ./config/platform-service/presets.example.json
```

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
