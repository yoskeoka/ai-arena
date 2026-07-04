# Platform Service Operator UI 仕様

## 目的

このドキュメントは、Phase 7 operator browser workflow として `Cloudflare Pages` から配信する operator UI contract を定義する。

対象は operator が game registration、AI submission、match request、run follow-up、ranking snapshot、artifact access を 1 つの route family で辿れる最小 surface である。表現や design system は固定せず、nav / route shape、state model、polling cadence、browser verification が依存してよい observation surface を定義する。

## この spec の責務範囲

この spec が定義するもの:

- operator nav / route shape
- page-local fetch / polling / mutation contract
- general registration / request / ranking read surface
- invite issuance surface
- preset queue / run follow-up action の最小 interaction
- delegated artifact access metadata の表示順と refresh 振る舞い
- browser verification が依存してよい stable observation surface

この spec が定義しないもの:

- spectator replay/viewer
- real-time push update
- advanced filtering、search、pagination
- public upload UI
- public leaderboard / tournament UI
- Pages Functions や server-side rendering
- 特定 AI agent / interactive skill の導入必須化
- request / response field inventory

## 参照関係

- `docs/specs/platform-frontend-architecture.md`: broader frontend の route / shared / API / auth boundary の正本
- `docs/specs/index.md`: operator-facing HTTP contract と JSON-RPC contract の lookup index
- `typespec/generated/openapi/operator/openapi.json`: operator route family の emitted OpenAPI artifact
- `typespec/namespaces/operator/api.tsp`: operator route family の TypeSpec source
- `operator-ui/src/generated/operator-api/`: operator surface で使う emitted client seam
- `docs/specs/platform-service-read-model.md`: compact row / detail view の正本
- `docs/specs/platform-service-skeleton.md`: first landing topology と Pages 配置方針の正本

## Delivery Shape

first landing の operator UI は `Cloudflare Pages` から配信する static app とする。

この surface は broader frontend の `operator` page family に属する shallow route family として扱う。

- rendering は browser 上の client-side application で完結してよい
- data fetch は operator-facing HTTP API へ直接行う
- server-side rendering、Server Components、Pages Functions は前提にしない
- implementation は後続 task の view 拡張を見越して component-based UI を採用してよい
- broader frontend の route-first rule に従い、page-specific state、polling、presentation は `operator` route family 配下へ閉じてよい
- concrete router library は first landing contract に含めない
- current `/` entry は `/operator` alias または redirect として扱ってよい

Phase 7 の operator route family は少なくとも次を持たなければならない。

- `/operator`
  - overview
- `/operator/invites`
  - invite issuance
- `/operator/games`
  - game registration list / create
- `/operator/submissions`
  - AI submission list / create
- `/operator/requests`
  - match request list / create
- `/operator/rankings`
  - ranking snapshot read
- `/operator/runs/{run_id}`
  - one run detail と follow-up action

## Screen Model

minimal operator UI は nav 上で少なくとも次の page/surface を持つ。

- overview page:
  preset queue、active runs、completed runs、selected run summary
- invites page:
  role-select invite issuance form と one-shot result view
- games page:
  registered game list と create form
- submissions page:
  admitted AI list と create form
- requests page:
  accepted match request list と create form
- rankings page:
  selected scope の durable ranking snapshot read
- run detail page:
  compact summary、submitted players、replay input locator group、
  artifact access、queued cancel / retry / rerun / promote action

browser verification は、少なくとも次の acceptance surface を route 遷移込みで観測できなければならない。

- operator nav が visible で、`Overview`、`Invites`、`Games`、`Submissions`、`Requests`、`Rankings` を辿れる
- overview page で preset queue panel、active runs panel、completed runs panel を表示できる
- invites page で `participant|developer|operator` のいずれかの invite を 1 件作成し、`invite_token` と `invite_url` を表示できる
- games page で registered game を 1 件以上作成し、list へ反映できる
- submissions page で AI submission を 1 件以上作成し、list へ反映できる
- requests page で manual match request を 1 件以上作成し、accepted request と latest run を表示できる
- run detail page で selected run の `result_summary` と artifact access entry を表示できる
- rankings page で completed official run の scope を選び、snapshot entry を表示できる

auth-enabled GitHub regression lane では、上記 operator surface に到達する前段として次も acceptance surface に含めなければならない。

- `/login` page の heading と GitHub login CTA
- provider authorize form の submit action
- callback 完了後の authenticated principal 表示
- invite token 付き `/login` から signup-only GitHub user で first signup を完了できること

browser verification lane は少なくとも次の 3 系統で同じ acceptance surface を共有しなければならない。

- fixture local regression lane:
  fixture-seeded backend を使う軽量 regression lane
- real local inspection/capture lane:
  actual `arena-service` と actual `operator-ui` を contributor または AI agent が同一環境で起動し、
  preset queue から completed detail までを操作し、review artifact を保存する lane
- dedicated CI browser lane:
  actual operator API request により preset queue から started/completed state を作り、
  同じ panel / detail / artifact observation surface を継続検証する lane
- auth-enabled GitHub regression lane:
  auth-enabled backend と別 process の repo-owned provider test double を使い、
  `/login -> /auth/github/login -> provider form -> callback -> session cookie -> /operator`
  を通したうえで同じ operator surface を確認する lane

contributor / operator / AI agent が依存してよい canonical local entrypoint は、長い env var 列ではなく次の既存 repo-owned command とする。

- `pnpm run verify:local`
- `pnpm run verify:local:real`
- `pnpm run verify:local:auth`

lane の mode 実装詳細、artifact dir 既定値、auth/mock distinction は helper 側へ閉じ込めなければならない。
default 成功時 output は quiet summary と exec log path に留め、
full diagnostic は explicit verbose opt-in 時だけ常時露出してよい。
failure 時も wrapper は full output の貼り返しではなく、exec log path と短い診断導線だけを返してよい。

dedicated CI browser lane の browser provisioning は、local canonical lane の Playwright browser bootstrap helper へ隠してはならない。

- current host-runner lane は workflow-managed `chrome` runtime を使ってよい
- workflow は browser version と executable presence を job log 上で明示しなければならない
- dedicated CI lane は `OPERATOR_UI_SKIP_BROWSER_BOOTSTRAP=1` により local helper の `playwright install chromium` fallback を bypass しなければならない
- lane ごとに runtime が異なっても、acceptance surface と artifact contract は共通に保たなければならない

dedicated CI browser lane は、browser runtime を repo checkout 外の pinned Playwright 公式 image へ載せてもよい。
ただし image version は repo が使う `@playwright/test` version と一致しなければならない。

auth-enabled GitHub regression lane は、current public login hand の regression capture を目的とする。

- login page の `Continue with GitHub` から provider authorize form へ進めること
- existing account scenario では、provider form 上の available test users から seed 済み `user_id` を選び、login 完了後、backend callback が session cookie を発行すること
- first signup scenario では、invite token 付き `/login` から signup-only `user_id` を選び、callback 中に account bootstrap / role bind / session cookie 発行まで完了すること
- callback 後に browser が `/operator` へ戻り、protected operator nav と overview surface を表示できること
- `GET /auth/session` が authenticated principal を返すこと
- logout 後は `/login` へ戻り、protected route が再度 session を要求すること

この lane は product login hand を増やすものではない。
provider test double は local / CI verification seam に限ってよい。
backend 側へ持ち込んでよい override は GitHub provider base URL だけであり、test double の authorize/token/user 実装や fixed user catalog を `arena-service` の HTTP serve path へ混在させてはならない。

overview 初期表示では completed runs panel の先頭 item を自動選択してよい。
completed item がない場合は、run detail surface は empty state を表示してよい。

## State Model

UI は少なくとも次の client state を持つ。

- operator API base URL
- selected operator route / selected run identity
- preset catalog read model
- active run items
- completed run items
- registered game items
- admitted AI items
- accepted match request items
- selected ranking scope
- selected ranking snapshot
- selected detail response
- read status:
  `idle|loading|ready|error`
- write status:
  `idle|submitting|success|error`

state は browser reload をまたいで永続化しなくてよい。
artifact access metadata や detail payload を local storage 等へ保存してはならない。

remote `Cloudflare Pages` deploy では、operator API base URL の初期値を build-time 設定で固定しなければならない。

- local `vite` development では base URL blank により same-origin `/api` proxy を使ってよい
- remote `Pages` deploy では same-origin `/api` fallback を前提にしてはならない
- staging / production deploy workflow は、それぞれの canonical backend URL を `VITE_OPERATOR_API_BASE_URL` として build に渡さなければならない

## Polling Contract

- overview active runs:
  5 秒 cadence で `GET /api/v1/matches/active` を poll してよい
- overview completed runs:
  10 秒 cadence で `GET /api/v1/matches/completed` を poll してよい
- selected run detail:
  run が選択されている間、15 秒 cadence で `GET /api/v1/runs/{run_id}` を再取得してよい

list polling と detail polling は independent timer でよい。
1 回の request failure で polling loop 全体を停止してはならない。
失敗時は最後に成功した表示を維持しつつ、panel ごとに error state を表示してよい。
list/detail/ranking endpoint が expected JSON shape を返さない場合も、blank page や uncaught exception ではなく panel-local error state へ落とさなければならない。

## Preset Queue Interaction

preset queue panel は、server-known preset catalog を operator が 1 click で enqueue できる surface とする。

- 1 action は 1 preset request だけを送る
- successful enqueue 後は active matches panel を即時 refresh してよい
- `submission_id` や `match_id` の manual override は first landing では必須にしない

first landing では preset catalog 自体を static UI config として配ってよい。
ただし enqueue request の `preset_id` は API contract に従う server-known value と一致しなければならない。

successful enqueue 後、local browser verification は preset queue action により active matches panel の row 増加または newly queued submission の可視化を観測できなければならない。

real local inspection/capture lane と dedicated CI browser lane では、preset queue action のあと actual service backend が `queued|leased|running|persisting|completed` のいずれかへ進むことを backend poll で待ち、browser reload または subsequent polling により completed row / detail 表示まで到達できなければならない。

## General Registration / Request / Ranking Interaction

games page、submissions page、requests page は、operator-facing write route を使う minimal form surface を持たなければならない。

- successful create 後は、対応 list を即時 refresh してよい
- requests page の item は `latest_run_id` を run detail deep-link として表示してよい
- rankings page は completed official run の scope または operator-selected scope から `GET /api/v1/rankings` を呼び、stored snapshot を read-only 表示してよい

run detail page は follow-up action を visibility とともに提供してよい。
