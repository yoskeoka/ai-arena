# operator-ui local verification

`operator-ui/` の local browser verification は、repo-owned な Playwright harness を canonical とする。
目的は human manual check 依存を減らし、AI agent でも contributor でも同じ command で
`preset queue`、`active/completed visibility`、`completed detail`、`artifact access entry`
の回帰を自己確認できるようにすることにある。

local verification は 2 lane で扱う。

- fixture local regression lane:
  `0068` の軽量 lane。fixture backend を起動し、最小回帰を素早く確認する
- real local inspection/capture lane:
  `0070` の実運用寄り lane。actual `arena-service` と actual `operator-ui` を起動し、
  preset queue から completed detail までを確認し、review artifact を保存する
- auth-enabled GitHub regression lane:
  repo-owned provider test double つき auth-enabled backend を起動し、
  `/login -> provider form -> callback -> session cookie -> /operator -> logout`
  を Playwright で回帰確認する

## Scope

この local verification が確認するもの:

- backend の `/healthz` 応答
- preset queue panel が visible で、1 action で enqueue できること
- active matches panel に queued submission が表示されること
- completed matches panel と completed detail が visible であること
- completed detail の `result_summary` と delegated artifact access entry が表示されること

この local verification が確認しないもの:

- Postgres-backed durable backend
- GitHub Actions 上の browser CI orchestration
- production/staging deploy 済み service との疎通

## Fixture local regression lane

local canonical lane は host-native のまま維持する。
CI remote verify が Playwright Docker image を使っていても、
contributor / AI agent が辿る repo-owned command はこの節の `pnpm` 実行を正本にする。

fixture local verification は次でよい。

```sh
cd operator-ui
pnpm run verify:local
```

この command は次を自動で行う。

- `node_modules` が missing なときだけ `pnpm install --frozen-lockfile`
- Playwright Chromium executable が missing なときだけ browser install helper
- `go run ./cmd/operator-ui-fixture --listen-addr 127.0.0.1:10000`
- `pnpm exec vite --host 127.0.0.1 --port 4173 --strictPort`
- Playwright browser verification

fixture backend は repo 内の Go service package を使い、次の状態を seed する。

- 1 queued submission
- 1 completed submission
- completed submission 向け `result-summary` download link

つまり、この lane は local backend/frontend/browser を同一環境で起動するが、
durable Postgres lane までは持ち込まない。

browser install が local host で hang する場合は、
canonical lane を切り替える前に `Node 22 LTS` で次を再試行してよい。

```sh
cd operator-ui
pnpm exec playwright install chromium
```

local Docker 実行は debugging fallback としては許容してよいが、
repo-owned canonical lane ではない。

## Real local inspection/capture lane

Postgres harness と `SeaweedFS` が使える local 環境では、`0070` lane を次で起動してよい。

```sh
make postgres-up
cd operator-ui
pnpm run verify:local:real
```

この command は次を自動で行う。

- `node_modules` が missing なときだけ `pnpm install --frozen-lockfile`
- Playwright Chromium executable が missing なときだけ browser install helper
- local compose 管理の Postgres を reset-first で張り直す
- `make postgres-up`
- `make postgres-schema-apply`
- local object storage が bootstrap できるなら `make seaweed-bootstrap`
- `make render-build`
- `PORT=10000 make render-start`
- `pnpm exec vite --host 127.0.0.1 --port 4173 --strictPort`
- Playwright browser verification

queue/state backend は Postgres を正本にする。
artifact backend は local object storage を優先する。
`SeaweedFS` bootstrap ができない環境では、artifact backend だけ file-backed fallback を使ってよい。
この lane の preset bootstrap は `make render-build` が生成する prepared preset executable を使い、
`presets.example.json` ではなく deploy-shaped catalog を正本にする。

default DSN は local compose harness に合わせて次を使う。

```text
postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable
```

CI と同じ 5432 port の外部 Postgres を使いたい場合は、
`AI_ARENA_PG_TEST_DSN` と `AI_ARENA_PG_ATLAS_DEV_DSN` を override してよい。

## Observation hooks

browser automation は role / visible text を第一選択とする。
ただし次の `data-testid` は stable contract として利用してよい。

- `operator-panel-preset-queue`
- `operator-panel-active-matches`
- `operator-panel-completed-matches`
- `operator-panel-completed-detail`
- `preset-queue-action-<preset-id>`
- `match-row-<run-id>`
- `match-detail-<run-id>`
- `artifact-entry-<artifact-kind>`

## Artifact paths

fixture local lane の failure artifact は `operator-ui/` 配下に出す。

- screenshots:
  `operator-ui/test-results/`
- traces:
  `operator-ui/test-results/`
- HTML report:
  `operator-ui/playwright-report/`

real local inspection/capture lane は success 時にも review artifact を保存する。

- completed detail screenshot:
  `operator-ui/test-results/real-local/completed-detail.png`
- Playwright trace:
  `operator-ui/test-results/real-local/operator-ui-flow.zip`
- backend log:
  `operator-ui/test-results/real-local/backend.log`
- frontend log:
  `operator-ui/test-results/real-local/frontend.log`
- HTML report:
  `operator-ui/playwright-report/real-local/`

auth-enabled GitHub regression lane は次で起動してよい。

```sh
make postgres-up
cd operator-ui
pnpm run verify:local:auth
```

この command は次を自動で行う。

- `node_modules` が missing なときだけ `pnpm install --frozen-lockfile`
- Playwright Chromium executable が missing なときだけ browser install helper
- local compose 管理の Postgres を reset-first で張り直す
- `make postgres-schema-apply`
- repo-owned mock GitHub OAuth server を別 process で起動
  - `ARENA_SERVICE_POSTGRES_DSN` があるときは canonical test user catalog を idempotent seed
- provider base URL override を注入した `arena-service` 起動
- `pnpm exec vite --host 127.0.0.1 --port 4173 --strictPort`
- Playwright で login page、provider form、callback、session、logout を検証

この lane では `.env` を読まず、
dummy client id/secret と repo-owned provider test double を使う。
人手の GitHub OAuth secret は不要とする。

auth regression lane の backend へ注入してよい endpoint override は次だけとする。

- `ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL`
- `ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL`

authorize / token / user の個別 URL env は local/CI verification contract に含めない。
mock GitHub server 側は code-embedded catalog として
`spectator-user01`、`developer-user01`、`operator-user01`
のような existing account user と、
`operator-signup-user01` のような signup-only user を保持してよい。
browser form は `user_id` text input と login button の最小 UI でよい。

auth regression の repo-owned command は
existing login lane と first signup lane の両方を同じ `pnpm run verify:local:auth`
の中で確認してよい。

first signup lane の manual spot-check では、
invite URL helper を使って signup-only user 向け login を開始してよい。

```sh
./tools/dev/local-invite-url.sh
```

この helper は operator role の invite token と
`http://localhost:5173/login?invite_token=...` を返す。
manual check ではその URL を開き、
provider form で signup-only user を選んで `/operator` 到達を確認する。

auth regression artifact は `operator-ui/` 配下に出す。

- screenshots:
  `operator-ui/test-results/auth-local/`
- traces:
  `operator-ui/test-results/auth-local/`
- backend log:
  `operator-ui/test-results/auth-local/backend.log`
- mock GitHub log:
  `operator-ui/test-results/auth-local/github-oauth-test-double.log`
- HTML report:
  `operator-ui/playwright-report/auth-local/`

artifact は git へ commit しない。
PR review へ添付する screenshot / trace / log は、まずこの path 群から取る。

## Local auth note

GitHub login の local manual check では、agent は `.env` を直接読まない。
human-managed secret は `.envrc -> dotenv` と `direnv exec` 経由で backend process へ渡す。

manual local auth lane の起動は `DEVELOPMENT.md` の `make start-backend-local` /
`make start-frontend-local` を正本にする。

- `tools/dev/operator-ui-backend.sh` を human が直接起動する場合、
  `OPERATOR_UI_TEST_SCENARIO` が空なら script 側が `direnv exec` を使って
  `make render-start` を呼んでよい
- Playwright verification lane は auth flow の canonical verification ではないため、
  `OPERATOR_UI_TEST_SCENARIO` が入っている run では `direnv exec` を自動で挟まない
- local callback URL の正本は
  `http://localhost:10000/auth/github/callback`
- manual local auth check では frontend も `http://localhost:5173` を使い、
  `127.0.0.1` と混在させない
- first operator signup 用の invite token が必要なら、
  repo helper command または
  `./app signup-invite-create --role operator` /
  `go run ./cmd/arena-service signup-invite-create --role operator`
  を backend と同じ Postgres DSN で実行して発行してよい

上記 manual local auth lane は human GitHub login 確認用であり、
repo-owned auth regression の正本は `pnpm run verify:local:auth` とする。

## Optional agent tactic

OpenAI の browser-interactive / Playwright 系 skill を使う場合でも、
repo contract は `pnpm run verify:local` の実行結果を正本とする。
agent-specific tooling は、その command の補助に留める。
