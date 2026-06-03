# operator-ui local verification

`operator-ui/` の local browser verification は、repo-owned な Playwright harness を canonical とする。
目的は human manual check 依存を減らし、AI agent でも contributor でも同じ command で
`preset queue`、`active/completed visibility`、`completed detail`、`artifact access entry`
の回帰を自己確認できるようにすることにある。

## Scope

この local lane が確認するもの:

- backend の `/healthz` 応答
- preset queue panel が visible で、1 action で enqueue できること
- active matches panel に queued submission が表示されること
- completed matches panel と completed detail が visible であること
- completed detail の `result_summary` と delegated artifact access entry が表示されること

この local lane が確認しないもの:

- Postgres-backed durable backend
- GitHub Actions 上の browser CI orchestration
- production/staging deploy 済み service との疎通

## Canonical command

初回だけ Playwright browser を install する。

```sh
cd operator-ui
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
```

以後の local verification は次でよい。

```sh
cd operator-ui
pnpm run verify:local
```

この command は次を自動で行う。

- `go run ./cmd/operator-ui-fixture --listen-addr 127.0.0.1:10000`
- `pnpm run dev -- --host 127.0.0.1 --port 4173`
- Playwright browser verification

fixture backend は repo 内の Go service package を使い、次の状態を seed する。

- 1 queued submission
- 1 completed submission
- completed submission 向け `result-summary` download link

つまり、この lane は local backend/frontend/browser を同一環境で起動するが、
durable Postgres lane までは持ち込まない。

## Observation hooks

browser automation は role / visible text を第一選択とする。
ただし次の `data-testid` は stable contract として利用してよい。

- `operator-panel-preset-queue`
- `operator-panel-active-matches`
- `operator-panel-completed-matches`
- `operator-panel-completed-detail`
- `preset-queue-action-<preset-id>`
- `match-row-<submission-id>`
- `match-detail-<submission-id>`
- `artifact-entry-<artifact-kind>`

## Failure artifacts

Playwright の failure artifact は `operator-ui/` 配下に出す。

- screenshots:
  `operator-ui/test-results/`
- traces:
  `operator-ui/test-results/`
- HTML report:
  `operator-ui/playwright-report/`

artifact は git へ commit しない。

## Optional agent tactic

OpenAI の browser-interactive / Playwright 系 skill を使う場合でも、
repo contract は `pnpm run verify:local` の実行結果を正本とする。
agent-specific tooling は、その command の補助に留める。
