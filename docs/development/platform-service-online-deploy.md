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
- https://render.com/docs/custom-domains
- https://render.com/docs/free
- https://render.com/docs/tls
- https://render.com/docs/inbound-ip-rules
- https://render.com/pricing
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

Custom domain / network protection research note:

- Render docs 上は free web service でも custom domain を設定できる
- Hobby workspace pricing では custom domains は 2 included とされている
- TLS は Render-managed certificate を自動で使える
- ただし inbound IP rule は web service では `Scale or Enterprise` 前提であり、
  free/Hobby current path では backend を IP allowlist で閉じる案は採らない
- この repo の current decision は custom domain なしで進めることであり、
  `onrender.com` default domain を staging / production backend URL として使う
- custom domain は deferred option として plan を残し、auth / Access / naming が固まった時点で再判断する

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

GitHub OAuth first landing contract:

- callback path は provider 衝突回避のため `/auth/{provider}/callback` を正本にする
  - GitHub OAuth app の callback path:
    `/auth/github/callback`
- current deploy shape では frontend が `Cloudflare Pages`、token exchange / session 発行主体が
  `Render` backend なので、GitHub OAuth app の callback URL は backend origin へ向ける
  - staging callback URL:
    `https://ai-arena-staging-p4ml.onrender.com/auth/github/callback`
  - production callback URL:
    `https://ai-arena-service.onrender.com/auth/github/callback`
- callback で受けた認可 code の token exchange、session 発行、login flow 復帰 redirect は
  backend 側で完結させる
- browser が自動送信する http-only cookie の正本は backend origin に寄せる
  - current `pages.dev` / `onrender.com` split-origin のままでも整合する
- frontend origin から same-origin で auth API を見せたい場合は、
  custom domain または Cloudflare proxy を伴う follow-up として扱う
- GitHub OAuth app は staging / production で 2 つ作成し、
  env var 名はそろえたまま value を環境ごとに分ける
- repo / runtime が共有する GitHub OAuth env 名は次で固定する
  - `ARENA_GITHUB_OAUTH_CLIENT_ID`
  - `ARENA_GITHUB_OAUTH_CLIENT_SECRET`
- backend は authorization URL generation / token exchange を supported OAuth library で扱う前提とし、
  上記 secret は provider client config 入力として使う
- callback / return flow を hard-code しないため、frontend return origin allowlist は env で上書きしてよい
  - `ARENA_AUTH_ALLOWED_RETURN_ORIGINS`
  - empty のときは repo canonical origin
    `http://localhost:4173`,
    `http://127.0.0.1:4173`,
    `http://localhost:5173`,
    `http://127.0.0.1:5173`,
    `https://staging.ai-arena.pages.dev`,
    `https://ai-arena.pages.dev`
    を default にしてよい
- pending OAuth state cookie の signing secret は次で override してよい
  - `ARENA_AUTH_COOKIE_SIGNING_SECRET`
  - empty のときは `ARENA_GITHUB_OAUTH_CLIENT_SECRET` を fallback に使ってよい
- 将来 Google などを追加するときも、
  provider-specific secret 名は `ARENA_<PROVIDER>_OAUTH_CLIENT_ID` /
  `ARENA_<PROVIDER>_OAUTH_CLIENT_SECRET` の規則に従う
- OIDC provider を追加するときは、client credentials に加えて issuer を env inventory に持たせてよい
  - `ARENA_GOOGLE_OIDC_ISSUER`
  - `ARENA_GOOGLE_OAUTH_CLIENT_ID`
  - `ARENA_GOOGLE_OAUTH_CLIENT_SECRET`
- ID token verification は backend 側で完結させ、
  callback / exchange 実行面を Cloudflare 側へ移すまでは provider secret や issuer config を frontend 側へ複製しない
- callback handler や token exchange を frontend origin 側へ移すまでは、
  同じ GitHub OAuth secret を Cloudflare 側へ常設しない

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
- `NEON_STAGING_MIGRATION_DSN`
  - GitHub Actions staging migration lane 用の direct/admin connection string
- `NEON_PRODUCTION_MIGRATION_DSN`
  - GitHub Actions production migration lane 用の direct/admin connection string

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
- https://developers.cloudflare.com/pages/platform/known-issues/
- https://developers.cloudflare.com/cloudflare-one/access-controls/policies/
- https://developers.cloudflare.com/cloudflare-one/integrations/identity-providers/one-time-pin/
- https://developers.cloudflare.com/cloudflare-one/integrations/identity-providers/cloudflare/
- https://www.cloudflare.com/plans/zero-trust-services/

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
- `online-release-staging-verify` の canonical remote lane は、
  GitHub-hosted runner 上で browser install を行うのではなく
  Playwright 公式 Docker image を runtime として使う
- remote lane の image tag は `operator-ui/package.json` の `@playwright/test` version と一致させる。
  たとえば repo が `1.54.1` を使うなら、
  workflow image も `mcr.microsoft.com/playwright:v1.54.1-noble` のように pin する
- remote lane では `pnpm exec playwright install --with-deps chromium` を追加実行しない。
  browser runtime と Linux dependency は pinned image 側を正本にする
- current remote verify lane の artifact 正本 path は次とする
  - `operator-ui/test-results/remote/`
  - `operator-ui/playwright-report/remote/`
- workflow summary には verify 対象 commit SHA、frontend URL、backend URL、
  artifact 名 `online-release-staging-verify` を残す

Direct Upload contract:

- current project は `Direct Upload` であり、Git integration へ後から切り替えない
- staging preview は `wrangler pages deploy ... --branch staging` のような CI-triggered preview deploy で扱う
- production deploy は tag/release lane の CI から明示的に publish する
- Pages 単体の branch control には依存せず、deploy trigger は repo workflow が握る

Cloudflare Access research note:

- Pages preview deployment は default public であり、preview protection を有効にする場合は
  Access policy を別途設定する
- Access policy では少なくとも email、email domain、IP range、country、
  login method、identity provider group、Cloudflare account member、service token、
  mTLS client cert といった制約を組み合わせられる
- first internal-use candidate としては `One-time PIN + allowlisted email addresses` か
  `Cloudflare account member only` が現実的である
- Zero Trust free plan は小規模 internal use では候補になるが、
  user-cap を踏まえると public GA 後の product auth 代替にはしない
- したがって current decision は、
  Cloudflare Access を staging / internal operator surface protection 候補として扱い、
  public-facing auth は別 child plan で product contract として扱うことである
- Pages preview を Access で閉じても `onrender.com` backend direct access 問題は別論点として残る
  ため、backend protection を達成したと誤認しない

## Internal Surface Protection Decision

Phase 6 時点で棚卸しする surface は次の 3 つである。

- staging frontend:
  `https://staging.ai-arena.pages.dev`
- production frontend:
  `https://ai-arena.pages.dev`
- staging / production backend:
  `https://ai-arena-staging-p4ml.onrender.com` と
  `https://ai-arena-service.onrender.com`

current path では、frontend と backend で保護境界を分けて扱う。

- Pages frontend は `Cloudflare Access` で internal surface として閉じる候補を第一選択にする
- `onrender.com` backend は Access では閉じられないため、
  current phase では public reachability を残したまま運用上の露出最小化で扱う
- public-facing product auth は
  `0087-product-auth-and-gated-signup-01-github-login-and-account-linking-foundation.md`
  を current execution 入口として別契約で扱う

選択肢比較:

- `no-auth-yet`
  - 利点: 追加 provider 設定なしで最短
  - 欠点: staging preview も production frontend も public のままになり、
    internal operator surface を accidental discovery から守れない
  - 判断: 第一選択にしない
- `Cloudflare Access on Pages`
  - 利点: current Pages / Cloudflare inventory のまま導入でき、
    repo に secret value を残さず developer email ベースで制御しやすい
  - 欠点: Render backend 直アクセスは残る。product auth の代替にもならない
  - 判断: Phase 6 / 7 初期の第一選択
- `product auth now`
  - 利点: backend / frontend を一貫した platform contract で閉じられる
  - 欠点: operator internal surface protection と public signup/auth の責務が混ざり、
    0076 以降の feature work を止める
  - 判断: `0080` へ defer

Access policy の first landing は次を基本形とする。

- login method:
  `One-time PIN`
- allowlist:
  開発者 email address を個別登録する
- 対象:
  staging preview を優先し、production frontend も internal-only 運用を続ける間は同じ方針で扱う

`Cloudflare account member only` は Cloudflare account に contributor を増やしたいときだけ採る。
current path では Cloudflare account member 追加よりも、
email allowlist のほうが repo 外 secret 配布を減らしやすい。

## Backend Direct Exposure Decision

`onrender.com` backend は free/Hobby 前提では IP allowlist や Access で閉じられない。
そのため current decision は「閉じたとみなさない」である。

現時点で許容する理由:

- current backend は high-value mutation surface をまだ広く持たない
- operator UI 側から使う API は Phase 6 の online confirmation に必要な最小面積へ寄せている
- product auth を先に仮実装するより、露出事実と closure 条件を明示して次 plan へ渡すほうが安全

この前提で守る運用ルール:

- backend URL は repo inventory に残してよいが、secret や credential と並べて書かない
- staging / production frontend が private でも backend は private ではない、と runbook に明記する
- new mutation endpoint や privileged read を追加する plan は、
  その endpoint を product auth / operator auth でどう閉じるかを同じ plan で決める

この risk を閉じたとみなせる条件:

- backend も Cloudflare 管理下 hostname / proxy 配下へ移し、Access または同等 control で守れる
- または backend 自体に platform auth / authorization を実装し、
  anonymous direct access では privileged operation が成立しない
- いずれの path でも staging / production の verification runbook が新しい境界に更新されている

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

## Release Flow Decision

Phase 6 closure では、online confirmation を次の release flow として閉じる。

1. local verification
   - browser / backend / artifact lane を local で起動し、
     `preset queue -> active/completed visibility -> completed detail` を確認する
2. CI
   - file-backed browser lane、Postgres-backed browser lane、
     既存 Go quality gate を通す
3. staging deploy
   - Pages preview と Render staging service を更新する
   - deploy 前に staging DB へ versioned migration を apply する
   - staging backend / UI の URL、deploy SHA、関連 artifact を記録する
4. staging verification
   - local / CI と同じ acceptance surface で remote staging を確認する
   - verification artifact を review 用に残す
5. production release
   - staging で確認済みの commit SHA だけを明示昇格する
   - Production backend は auto deploy `off` を維持する
   - backend/frontend deploy 前に production DB へ versioned migration を apply する

この release flow が repo-owned workflow / runbook として固定されるまでは、
Phase 6 は completed とみなさない。

## Repo-owned Release Workflow

repo で使う canonical workflow 名は次で固定する。

- staging deploy:
  `.github/workflows/online-release-staging.yml`
- staging verification:
  `.github/workflows/online-release-staging-verify.yml`
- production release:
  `.github/workflows/online-release-production.yml`

repo workflow は `verified commit` を主語にしつつ、trigger は次で自動化する。

- staging deploy:
  `main push` 後、同じ SHA の push-triggered CI workflow がすべて `success` になった時点で自動起動する
- staging verification:
  staging deploy workflow が `success` で終わった時点で自動起動する
- production release:
  `tag push` で自動起動する。ただし target SHA が `origin/main` に含まれない場合は fail する

manual rerun / rollback 用に `workflow_dispatch` も残してよい。
manual dispatch の `commit_sha` input は 40 桁の full SHA を正本とするが、
repository 内で一意に解決できる短縮 hexadecimal SHA も受け付けてよい。
workflow は deploy/verify/release の実処理前に canonical full SHA へ正規化する。

schema change を含む rollout は backward-compatible release を前提にする。
expand / dual-read-write / cleanup を 1 回の release に押し込めず、
staging rehearsal を通した順序だけを production に昇格させる。

### Staging Deploy Contract

staging deploy workflow は次を 1 run にまとめる。

1. target commit SHA を checkout する
2. `NEON_STAGING_MIGRATION_DSN` を使って staging DB へ versioned migration を apply する
3. `operator-ui/` を canonical `pnpm install` / `pnpm run build` で build する
   - build-time に `VITE_OPERATOR_API_BASE_URL=${STAGING_BACKEND_URL}` を渡し、
     preview frontend が staging backend URL を直接参照できるようにする
4. `Cloudflare Pages` preview へ `staging` branch alias で direct upload する
5. `Render staging` を同じ commit SHA で deploy hook 起動する
6. workflow summary に backend / frontend URL と commit SHA を残す
   - DB migration が deploy より前に完了したことも残す

auto trigger contract:

- authoritative trigger は `main push` で 1 回だけ起動する `online-release-staging` 自身とする
- release workflow の `prepare` job が commit diff を見て deploy 要否を判定する
  - initial skiplist は狭く始め、`docs/**`、`README.md`、
    `.github/PULL_REQUEST_TEMPLATE.md` だけを non-release change とみなす
  - 上記だけが changed のときは `should_deploy=false` で clean に skip する
  - それ以外の change は release candidate として扱う
- release candidate の場合だけ、同じ `head_sha` に対する required push workflow
  (`go-ci` / `operator-ui-browser`) のうち、
  その workflow 自身の `push.paths` 対象に当たるものを poll し、
  全件 `success` を確認してから deploy へ進む
- required push workflow に `failure` / `cancelled` / `timed_out` が出た場合、
  staging deploy workflow 自体を failed にして止める
- required push workflow が 1 本も該当しない release candidate
  (例: release workflow 自身の変更) は、そのまま deploy へ進めてよい
- 同じ SHA に対して staging deploy は 1 回だけ進める

staging frontend URL は current project shape では次を正本とする。

- `https://staging.ai-arena.pages.dev`

staging backend URL は current service inventory では次を正本とする。

- `https://ai-arena-staging-p4ml.onrender.com`

repo に必要な GitHub secret 名:

- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_ACCOUNT_ID`
- `RENDER_STAGING_DEPLOY_HOOK_URL`
- `NEON_STAGING_MIGRATION_DSN`

deploy hook secret は value 自体を repo に残さず、
Render Dashboard の service settings から再発行できる issuance path だけを共有する。

migration secret は value 自体を repo に残さず、
Neon Console の Connect modal から pooled/direct を取り違えないように
取得経路だけを共有する。

staging preview は static Pages deploy なので、current shape では repo-owned proxy を持たない。
したがって preview frontend は same-origin `/api` fallback に依存してはならない。
backend 側も `https://staging.ai-arena.pages.dev` からの cross-origin fetch を受け付けなければならない。

### Staging Verification Contract

staging verification workflow は local / CI lane と acceptance surface をそろえるため、
`operator-ui` の Playwright managed-backend lane を remote URL 相手に再利用する。

確認対象:

- backend `GET /healthz`
- `POST /api/v1/preset-matches`
- `GET /api/v1/matches/active`
- `GET /api/v1/matches/completed`
- `GET /api/v1/matches/{submission_id}`
- frontend operator surface の queue / active / completed / detail 操作
- delegated artifact download link の有無

staging verification workflow は少なくとも次を artifact として残す。

- Playwright trace zip
- screenshot
- HTML report

verification 成功後、workflow summary には次を明記する。

- verified commit SHA
- verified staging frontend URL
- verified staging backend URL
- artifact location

auto trigger contract:

- upstream `online-release-staging` run が `success` のときだけ自動起動する
- verify 対象 SHA は upstream staging run の `head_sha` をそのまま使う
- default target URL は repo inventory に記録した staging frontend/backend URL とする

### Production Release Contract

production release workflow は次を守る。

- `tag push` を canonical trigger とする
- target SHA は pushed tag が指す commit を使う
- target SHA が `origin/main` に含まれない場合は release を失敗させる
- `NEON_PRODUCTION_MIGRATION_DSN` を使って production DB へ versioned migration を先に apply する
- backend deploy は `Render Production auto deploy = off` を維持したまま、
  deploy hook で明示起動する
- frontend deploy は `Cloudflare Pages` production へ同じ commit build artifact を upload する
  - build-time に `VITE_OPERATOR_API_BASE_URL=${PRODUCTION_BACKEND_URL}` を渡す
- backend は `https://ai-arena.pages.dev` からの cross-origin fetch を受け付けなければならない
- workflow summary に promoted commit SHA と trigger tag を残す

repo に必要な GitHub secret 名:

- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_ACCOUNT_ID`
- `RENDER_PRODUCTION_DEPLOY_HOOK_URL`
- `NEON_PRODUCTION_MIGRATION_DSN`

manual rerun では staging verification run URL を任意 input として添えてよい。
repo workflow は、tag-triggered path では tag 名と promoted SHA を最小記録として扱う。

### Rollback Contract

rollback は「前回の known-good commit SHA を staging / production workflow に再入力する」形を canonical とする。

- staging rollback:
  `online-release-staging.yml` に previous good SHA を渡して再実行する
- production rollback:
  `online-release-production.yml` に previous good SHA と元の staging verification evidence を渡して再実行する
- `previous good SHA` は full SHA を優先し、短縮 SHA を使う場合も repository 内で一意に解決できる値だけを使う

Phase 6 では DB schema rollback や object migration rollback の自動化までは扱わない。
この line の rollback は backend/frontend process を known-good SHA に戻すところまでを正本とする。

## Developer Access Inventory

repo に残してよいのは inventory と issuance path だけであり、secret value 自体は残さない。

残してよいもの:

- staging / production URL
- Render workspace / project / environment / service ID
- Neon organization / project / branch ID
- R2 account ID / bucket 名 / endpoint
- Pages project 名と production URL
- Access policy 名、適用対象 hostname、想定 login method
- credential / token / password をどこで発行するかの runbook

repo に残さないもの:

- password 本文
- session token
- API key
- DSN 実値
- R2 secret access key
- Access service token secret

current path では、staging / production へ開発者が到達するための追加 secret は
provider dashboard、secret manager、または Access issuance flow にだけ置く。

### Staging Access Runbook

1. frontend URL と backend URL を inventory から確認する
   - frontend:
     `https://staging.ai-arena.pages.dev`
   - backend:
     `https://ai-arena-staging-p4ml.onrender.com`
2. staging frontend を Access policy 配下に置く場合は、
   Cloudflare Zero Trust dashboard で該当 application の allowlist に開発者 email を追加する
3. login method が `One-time PIN` の場合は、対象 email で PIN を受け取って frontend へ入る
4. backend 直確認が必要なときは、frontend protection と別論点であることを意識して
   `curl -i https://ai-arena-staging-p4ml.onrender.com/healthz` や preflight check だけを行う
5. repo や PR には PIN、Access cookie、session token を残さない

### Production Access Runbook

1. frontend URL と backend URL を inventory から確認する
   - frontend:
     `https://ai-arena.pages.dev`
   - backend:
     `https://ai-arena-service.onrender.com`
2. production frontend を internal-only で使う間は、
   staging と同じ Access policy pattern を production hostname にも適用する
3. production backend 直確認は staging より厳しく扱い、
   health / headers / CORS など最小 read-only 確認に留める
4. deploy hook、DSN、R2 credential、Access service token は repo に残さず、
   provider dashboard または secret manager の issuance path だけを共有する

## Staging Failure Runbook

`service_queue_records` schema 未適用のような staging startup failure を疑うときは、
次の順序で切り分ける。

1. staging backend health を確認する
   - `curl -i https://ai-arena-staging-p4ml.onrender.com/healthz`
2. remote operator flow が CORS / 5xx で崩れていないかを確認する
   - `curl -i -X OPTIONS https://ai-arena-staging-p4ml.onrender.com/api/v1/preset-matches -H 'Origin: https://staging.ai-arena.pages.dev' -H 'Access-Control-Request-Method: POST'`
3. Render deploy log で relation not found や startup exit を確認する
4. Neon staging DB に direct/admin DSN で接続し、対象 table が存在するか確認する
   - `psql "$NEON_STAGING_MIGRATION_DSN" -c '\dt service_queue_records'`
5. 未適用なら repo の target commit を checkout した状態で migration lane を再実行する
   - `AI_ARENA_PG_MIGRATION_DSN="$NEON_STAGING_MIGRATION_DSN" make postgres-migrate-apply`
6. 手動で同等 schema を先に入れた DB を今後の workflow に乗せる場合は、
   revision history を 1 回だけ baseline する
   - `AI_ARENA_PG_MIGRATION_DSN="$NEON_STAGING_MIGRATION_DSN" make postgres-migrate-baseline VERSION=20260529000000`
7. その後に staging backend deploy をやり直し、
   `online-release-staging-verify` で同じ commit SHA を検証する

repo-owned migration helper は empty DB では初回 apply を自動許可するが、
user table が存在するのに Atlas revision history がない DB では fail-fast する。
staging remediation 後のような manual schema 済み DB は、人間が baseline を完了させてから
通常 workflow に戻す。

## Custom Domain Deferred Option

current path では custom domain を導入しない。
ただし deferred option として、次の事実を記録する。

- References:
  - https://www.cloudflare.com/products/registrar/
  - https://developers.cloudflare.com/registrar/
  - https://developers.cloudflare.com/registrar/get-started/register-domain/
- Render free/Hobby でも custom domain 自体は設定できる
- custom domain を 1-2 個使うだけなら Render 側追加コストは current pricing では発生しにくい
- Cloudflare Registrar は TLD ごとの registry / ICANN cost ベースで扱われ、
  固定の単一価格ではない
- domain 自体の維持費は TLD 依存であり、購入前に `.com` / `.net` / `.dev` の
  実際の登録 / 更新額を確認する
- 初年度だけ安く更新で跳ねる TLD より、継続更新価格を優先して選ぶ
- naming 候補は `ai-arena` そのもの、あるいは operator/admin/public の役割が読める
  prefix / subdomain を含めて評価する
- auth を product 側で実装した結果、custom domain 不要判断に戻る可能性もある

後続で custom domain を採る条件:

- backend も Cloudflare 管理下 hostname へ寄せて Access で守りたい
- public-facing naming / branding を整えたい
- production / staging / operator surface を host 単位でより明確に分けたい

## Repo-local startup contract

- backend process の first landing command:
  - `make render-build`
  - `make render-start`
- desired Render build/start command for the first remote lane:
  - build:
    `make render-build`
  - start:
    `make render-start`
- real remote lane の canonical preset catalog は
  `./config/platform-service/presets.remote-bootstrap.json` とする
- この catalog は `presets.example.json` を流用せず、
  `make render-build` が生成する prepared preset executable を参照しなければならない
- staging / production の `ARENA_SERVICE_PRESET_CONFIG` は上記 canonical path を指す
- `make render-start` は Render の `PORT` を優先して
  `0.0.0.0:$PORT` へ bind する。`PORT` 未設定時だけ `10000` を fallback に使う
- preset catalog は server-known participant set のみを持ち、
  operator request は `preset_id` と optional `submission_id` / `match_id` / `output_dir` override までに留める
- 2026-06-02 の `render services --confirm -o json` 観測では、
  `ai-arena-service` / `ai-arena-stg` ともに build command は `go build -tags netgo -ldflags '-s -w' -o app`、
  start command は `./app` のままだった。
  Render 設定更新後は `make render-build` / `make render-start` へ寄せる
  remote polling API を出す前に service command を上記 desired shape へ更新する必要がある

## Release Runbook

release operator は次の順で実行する。

1. local verification
   - `docs/development/operator-ui-local-verification.md` の real-local lane を通す
2. Merge PR into `main`
3. CI confirmation
   - `online-release-staging.yml` の `prepare` が merged SHA に対する required push workflow の
     全件 `success` を待つ
4. staging deploy
   - `online-release-staging.yml` が同じ merged SHA で自動起動する
5. staging verification
   - `online-release-staging-verify.yml` が同じ SHA で自動起動する
   - schema change を含む release では staging DB migration apply が summary に残ることを確認する
6. production release
   - GitHub Release 作成などで tag を push する
   - `online-release-production.yml` が tag SHA で自動起動する

staging verification が failed の間は production tag を作らない。

## Required Evidence

PR review や release handoff で最低限残す evidence は次で固定する。

- local:
  representative screenshot または Playwright artifact
- CI:
  `go-ci` と `operator-ui-browser` の成功 run
- staging deploy:
  workflow summary 上の frontend/backend URL と commit SHA
  - DB migration が先に適用されたこと
- staging verification:
  Playwright artifact と workflow summary
- production release:
  workflow summary 上の promoted SHA と trigger tag
  - DB migration が先に適用されたこと
