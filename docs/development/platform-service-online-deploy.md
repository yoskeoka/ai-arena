# Platform service online deploy bootstrap

online service skeleton の first landing provider inventory と、
repo が前提にする deploy / secret contract をこの文書で固定する。

click-by-click の signup 手順は扱わない。
ここに残すのは repo 側が共有すべき provider asset inventory、secret name、
および first landing の運用方針だけである。

## Render

References:

- https://render.com/docs/cli-reference
- https://render.com/docs/deploys
- https://api-docs.render.com/reference/update-service

Observed via `render` CLI on 2026-06-02:

- workspace:
  `Render workspace` (`tea-cspvg7ggph6c738l3mi0`)
- project:
  `ai-arena` (`prj-d8e6bk4p3tds73891cb0`)
- environments:
  - `Production` (`evm-d8e6bk4p3tds73891cbg`)
  - `Staging` (`evm-d8e6c0ek1jcs739lit3g`)
- services:
  - `ai-arena-service` (`srv-d8esn1h9rddc73cmcjf0`)
    - URL: `https://ai-arena-service.onrender.com`
    - slug: `ai-arena-service`
    - region: `oregon`
    - branch: `main`
    - observed auto deploy: `no`, trigger `off`
  - `ai-arena-stg` (`srv-d8e6mg9o3t8c73f68b5g`)
    - URL: `https://ai-arena-staging-p4ml.onrender.com`
    - region: `oregon`
    - branch: `main`
    - observed auto deploy: `yes`, trigger `checksPass`

First landing contract:

- `Staging` は Git-backed service を使い、`main` の `checksPass` auto deploy を維持する
- `Production` も当面は Git-backed service のままでよいが、auto deploy は `off` にする
- `Production` 反映は manual deploy で行い、Staging で検証済みの commit SHA を指定する
- image deploy や release branch 導入は follow-up に残す

Current operational note:

- `ai-arena.onrender.com` は取得せず、Production default domain は
  `https://ai-arena-service.onrender.com` を使う
- custom domain は当面導入しない。収益化や外部公開要件が固まった後に再検討する
- 2026-06-02 時点の desired deploy policy は observed state と一致している

Service-level env / secret contract:

- `ARENA_SERVICE_POSTGRES_DSN`
- `ARENA_SERVICE_PRESET_CONFIG`
  - `arena-service serve` が読む server-known preset catalog の JSON path
- `ARENA_SERVICE_ARTIFACT_BACKEND`
  - `filesystem` or `r2`
- `ARENA_SERVICE_ARTIFACT_R2_ACCOUNT_ID`
- `ARENA_SERVICE_ARTIFACT_R2_BUCKET`
- `ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT`
- `ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID`
- `ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY`

## Neon Postgres

References:

- https://neon.com/docs/reference/neon-cli
- https://neon.com/docs/reference/cli-branches
- https://neon.com/docs/get-started-with-neon/connect-neon

Observed via `neonctl` on 2026-06-02:

- organization:
  `ai-arena` (`org-shiny-violet-06627432`)
- projects:
  - `ai-arena-stg` (`cold-math-10457878`)
    - branch id: `br-red-forest-aooov67v`
    - branch name: `staging`
  - `ai-arena-prod` (`restless-feather-07305018`)
    - branch id: `br-calm-sky-ao2sdz3p`
    - branch name: `production`

Verified non-interactive commands:

- `npx neonctl@latest me`
- `npx neonctl@latest orgs list`
- `npx neonctl@latest projects list --org-id org-shiny-violet-06627432`

Current operational note:

- `branches list --project-id ...` 自体は browser login 完了後に利用できる
- database name / pooled connection string / direct connection string は、
  Neon dashboard の Connect modal か API key-backed CLI run で別途棚卸しする
- repo へは接続文字列そのものを残さず、secret name だけを残す

Secret contract:

- `ARENA_SERVICE_POSTGRES_DSN`
  - first landing の runtime では pooled connection string を優先する
- optional operator note:
  - direct connection string は migration / admin lane 用に separate secret として保持してよい

## Cloudflare R2

References:

- https://developers.cloudflare.com/r2/get-started/cli/
- https://developers.cloudflare.com/r2/reference/wrangler-commands/
- https://developers.cloudflare.com/r2/get-started/s3/

Observed via `wrangler` on 2026-06-02:

- account:
  `Cloudflare account` (`9ea0d3de35f99cc17ef3a939e2968e8b`)
- buckets:
  - `ai-arena-stg`
  - `ai-arena-prod`

Verified non-interactive commands:

- `npx wrangler whoami`
- `npx wrangler r2 bucket list`
- `npx wrangler r2 object put ai-arena-stg/... --file ... --remote`
- `npx wrangler r2 object get ai-arena-stg/... --file ... --remote`
- `npx wrangler r2 object delete ai-arena-stg/... --remote`

CLI choice for this repo:

- first landing の R2 CLI は `npx wrangler` を優先する
- `npx cf` は現時点では採用しない
- official docs でも、Wrangler は bucket setting と single object operation の基本 CLI として案内されている
- repo に恒久採用するまでは `npx wrangler` のまま使い、採用決定後に `devDependencies` へ追加する

Operational notes:

- `wrangler r2 object *` は既定で local storage を触るため、remote bucket を触るときは `--remote` が必要
- Codex sandbox では `~/.config/.wrangler/logs` への log write が `EROFS` で失敗するが、
  実コマンド自体は成功することがある
- `wrangler` の object command を並列に叩くと local sqlite lock を踏みうるため、Codex からは直列実行に寄せる

S3-compatible endpoint contract:

- endpoint:
  `https://9ea0d3de35f99cc17ef3a939e2968e8b.r2.cloudflarestorage.com`
- staging bucket:
  `ai-arena-stg`
- production bucket:
  `ai-arena-prod`

Secret contract:

- `ARENA_SERVICE_ARTIFACT_R2_ACCOUNT_ID`
- `ARENA_SERVICE_ARTIFACT_R2_BUCKET`
- `ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT`
- `ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID`
- `ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY`

Artifact delivery contract:

- write model が保持するのは bucket/key 相当の stable locator だけとする
- download URL や delegated token は request 時に発行する derived metadata とし、永続化しない
- completed artifact download で object bytes を `arena-service` が proxy しない

Local verification note:

- local contributor verification では、
  同じ `ARENA_SERVICE_ARTIFACT_BACKEND=r2` / `ARENA_SERVICE_ARTIFACT_R2_*` env contract を
  `SeaweedFS` の local S3-compatible endpoint へ向けて再利用してよい
- この harness は `Cloudflare R2` 自体の代替ではなく、
  deploy-shaped artifact write / locator / delegated download flow を local で確認する lane として扱う

## Cloudflare Pages

References:

- https://developers.cloudflare.com/pages/platform/limits/
- https://developers.cloudflare.com/workers/platform/pricing/
- https://developers.cloudflare.com/pages/get-started/direct-upload/
- https://developers.cloudflare.com/pages/configuration/preview-deployments/
- https://developers.cloudflare.com/pages/configuration/branch-build-controls/

Observed via `wrangler` on 2026-06-02:

- project:
  - `ai-arena`
    - production URL: `https://ai-arena.pages.dev`
    - Git Provider: `No`
    - deploy mode: `Direct Upload`

Current operational note:

- staging 用の別 Pages project は作らない
- Pages project は 1 つに保ち、preview / production を deployment 側で分ける
- `main push -> staging preview`, `tag push -> production` の desired flow は、
  Git integration よりも `Direct Upload + CI` のほうが適している
- static asset build は repo CI が担い、Pages には prebuilt artifact を upload する
- first landing の operator UI build contract は次で固定する
  - app root:
    `operator-ui/`
  - install:
    `pnpm install`
  - build:
    `pnpm run build`
  - output directory:
    `operator-ui/dist`
- `pnpm` v11 の build-script gate により、`operator-ui/pnpm-workspace.yaml` で
  `esbuild` build script を明示許可する
- Pages 側 environment variable 名は未使用でよい。Pages Functions 導入が必要になったら追記する
- dedicated browser CI lane も同じ app root / install / build contract を共有してよい。
  Pages deploy lane が drift しないよう、browser verification で使う frontend install/build command は
  `operator-ui/` の canonical `pnpm` surface に揃える

Direct Upload contract:

- current project は `Direct Upload` であり、Git integration へ後から切り替えない
- staging preview は `wrangler pages deploy ... --branch staging` のような CI-triggered preview deploy で扱う
- production deploy は tag/release lane の CI から明示的に publish する
- Pages 単体の branch control には依存せず、deploy trigger は repo workflow が握る

## Provider Drift Check

first landing の desired contract と、2026-06-02 時点の observed state を次の観点で比較する。

- desired:
  - Render `Production` auto deploy = `off`
  - Render `Staging` auto deploy trigger = `checksPass`
  - R2 buckets = `ai-arena-stg`, `ai-arena-prod`
  - Pages direct-upload project inventory recorded
- observed:
  - Render `Staging` は desired 通り
  - Render `Production` は desired 通り
  - R2 buckets は desired 通り
  - Pages project は desired 通り

## Repo-local startup contract

- backend process の first landing command:
  - `make render-build`
  - `make render-start`
- desired Render build/start command for the first remote lane:
  - build:
    `make render-build`
  - start:
    `make render-start`
- real remote lane では `presets.example.json` をそのまま使わず、
  Render service へ mount / bake / secret-managed path した preset catalog を `ARENA_SERVICE_PRESET_CONFIG` で指定する
- `make render-start` は Render の `PORT` を優先して
  `0.0.0.0:$PORT` へ bind する。`PORT` 未設定時だけ `10000` を fallback に使う
- preset catalog は server-known participant set のみを持ち、
  operator request は `preset_id` と optional `submission_id` / `match_id` / `output_dir` override までに留める
- 2026-06-02 の `render services --confirm -o json` 観測では、
  `ai-arena-service` / `ai-arena-stg` ともに build command は `go build -tags netgo -ldflags '-s -w' -o app`、
  start command は `./app` のままだった。
  Render 設定更新後は `make render-build` / `make render-start` へ寄せる
  remote polling API を出す前に service command を上記 desired shape へ更新する必要がある
