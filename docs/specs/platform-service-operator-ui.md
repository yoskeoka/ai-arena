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
- run follow-up action の最小 interaction
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
- `operator-ui/src/lib/operatorApiClient.ts`: browser-specific adapter seam。generated contract を route family へ接続し、base URL / credentials / login URL / UI-facing error normalization を local ownership に留める
- `docs/specs/platform-service-read-model.md`: compact row / detail view の正本
- `docs/specs/platform-service-skeleton.md`: first landing topology と Pages 配置方針の正本

## Delivery Shape

first landing の operator UI は `Cloudflare Pages` から配信する static app とする。

この surface は broader frontend の `operator` page family に属する shallow route family として扱う。

- rendering は browser 上の client-side application で完結してよい
- data fetch は operator-facing HTTP API へ直接行う
- route page と shared hook は handwritten DTO / handwritten path assembly を source-of-truth にしてはならず、operator route family 向けに emit された TypeSpec-generated contract を使わなければならない
- server-side rendering、Server Components、Pages Functions は前提にしない
- implementation は後続 task の view 拡張を見越して component-based UI を採用してよい
- broader frontend の route-first rule に従い、page-specific state、polling、presentation は `operator` route family 配下へ閉じてよい
- concrete router library は first landing contract に含めない
- current `/` entry は `/operator` alias または redirect として扱ってよい
- browser-specific concern である base URL normalize、credentialed fetch policy、GitHub login URL assembly、HTTP error message normalization は thin local adapter に閉じ込めてよい

Phase 7 の operator route family は少なくとも次を持たなければならない。

- `/operator`
  - overview
- `/operator/invites`
  - invite issuance
- `/operator/games`
  - game registration list / create
- `/operator/submissions`
  - AI bot list / create / revision
- `/operator/requests`
  - match request list / create
- `/operator/rankings`
  - ranking snapshot read
- `/operator/runs/{run_id}`
  - one run detail と follow-up action

## Screen Model

minimal operator UI は nav 上で少なくとも次の page/surface を持つ。

- overview page:
  active runs、completed runs、selected run summary
- invites page:
  role-select invite issuance form と one-shot result view
- games page:
  registered game list、game bundle ZIP の file chooser、admission status/error、manifest 由来の read-only
  game ID / game version / artifact digest、supported rulesets selector、activation action。activation は
  admission 成功後だけ可能であり、成功時は form state を reset して scope list を refresh する。upload または
  activation failure は既存 list を壊さず form-local error として表示する。admission 後に activation
  されない artifact の cleanup はこの surface の範囲外であり、`docs/issues/0038-unactivated-game-bundle-retention.md`
  を参照する
- submissions page:
  scope、bot name、new bot / existing bot revision の choice、AI bundle ZIP の file chooser、admission
  status/error、admitted artifact digest の read-only confirmation、bot create/revise action、sign-in 済み principal
  の bot list。
  bundle upload は selected scope とともに行い、admission 成功後だけ create/revise を可能にする。admitted artifact
  digest は read-only confirmation として表示してよいが、artifact digest、AI submission ID、runtime/AI identity、
  artifact reference を入力する control を page に置いてはならない。create/revise 成功時は form state を reset して
  selected scope の bot list を refresh する。file の再選択または upload failure は prior admission を invalidate し、
  既存 list を壊さず form-local error として表示する。legacy AI submission の create/list surface は migration
  compatibility のため HTTP API に残り得るが、この page には表示しない
- requests page:
  accepted match request list と create form
- rankings page:
  selected scope の durable ranking snapshot read
- run detail page:
  compact summary、submitted players、replay input locator group、
  artifact access、queued cancel / retry / rerun / promote action。compact summary 内の metadata は、
  lifecycle/action message だけでなく attempt、game identity、ruleset、output directory、result summary locator を
  通常の operator display 状態で判読可能に表示しなければならない。dark background に置く metadata は、
  background と十分に視覚的に区別できる foreground color を明示しなければならない。light surface の metadata
  表示、route/API payload、polling、run follow-up action availability はこの要件によって変更しない

browser verification は、少なくとも次の acceptance surface を route 遷移込みで観測できなければならない。

- operator nav が visible で、`Overview`、`Invites`、`Games`、`Submissions`、`Requests`、`Rankings` を辿れる
- overview page で active runs panel、completed runs panel、selected run summary/detail を表示できる
- invites page で `participant|developer|operator` のいずれかの invite を 1 件作成し、`invite_token` と `invite_url` を表示できる
- games page で registered game を 1 件以上作成し、list へ反映できる
- submissions page で scope-compatible AI bundle ZIP を upload し、admitted artifact digest を確認して bot を
  1 件作成または revision し、bot list へ反映できる
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
  active/completed run と completed detail までを操作し、review artifact を保存する lane
- dedicated CI browser lane:
  seeded または operator API request により active/completed state を用意し、
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

## Remote Version Identity と Read-only Smoke

backend は認証 middleware の外に public、read-only な `GET /version` を提供する。wire contract の正本は
`typespec/namespaces/operator/version.tsp` と shared model であり、release verification はここで定義された
`version_sha` が target の full commit SHA と完全一致した場合だけ成功とする。空値、短縮 SHA、branch 名、
build 時刻、hostname、および response shape の不正はいずれも version identity にならない。

staging の remote smoke は deploy 済み frontend への接続、backend の exact version identity、匿名
`/auth/session`、および匿名 browser の `/operator` から login route への redirect だけを確認する。
operator API の mutation、fixture ZIP、machine account、OIDC、test auth、game / bot registration、match、
ranking は remote smoke の責務外であり、local または CI の auth-mock lane が継続して検証する。

## Public Liveness と Worker Readiness

backend は認証 middleware の外に public、read-only な `GET /healthz` を提供する。wire response の正本は
`typespec/namespaces/operator/health.tsp` と `typespec/namespaces/shared.tsp` であり、生成された OpenAPI
および client はその出力でなければならない。

`/healthz` の責務は HTTP liveness と worker readiness を分離することである。

- handler が応答可能なら HTTP status は常に `200` とする。worker の pending state を Render health check
  の failure にしてはならない
- response は API component と worker component の状態を返す
- API が応答可能で worker の ownership または initial recovery が未完了の間は worker component を
  `NOT_READY` とする
- worker guard の取得と initial queue recovery が完了した後だけ worker component を `OK` とする
- worker process の終了、context cancellation、ownership timeout、guard/recovery failure の後は
  `OK` を維持してはならない

worker readiness は queue mutation の推測値ではない。worker が ready になる前に initial recovery、claim、
match execution を開始してはならず、ready になった後だけ staging release verification が worker execution
可能と扱う。fixture backend のように real worker を起動しない static service は、その fixture が提供する
read-only backend 全体が利用可能であることを明示的に readiness として返してよい。

staging release は、target commit の `/version` が exact に一致した後、同じ backend の `/healthz` が API と
worker の両方を `OK` と返すまで待つ。polling は 10 秒間隔、最大 42 attempts（最大 7 分）、1 request 15 秒
timeout とし、HTTP failure、malformed response、component の non-`OK` は retry する。最大 attempts 到達時は
last observed HTTP status と component state を release summary に残し、release を failure とする。

remote browser smoke も exact version の確認後に同じ health readiness を read-only に確認する。anonymous
session、login redirect、既存の local fixture / auth-mock の protected flow の責務は変わらない。

production backend release も同じ public endpoint を deployment evidence に使う。migration や deploy mutation の
前に serving backend の full SHA を取得し、target の exact version と ready worker を順に確認する。target
verification が失敗した場合だけ、capture した previous full SHA に対して 1 回だけ backend deploy を起動し、
同じ version/readiness contract で recovery を確認する。verified rollback は target release の成功を意味せず、
workflow は release failure として終了する。version endpoint がない legacy backend の最初の rollout は、manual
dispatch で repository 内に存在する previous full SHA を明示しなければ deploy mutation を開始してはならない。
code rollback は DB schema rollback を含まず、frontend deployment identity と frontend rollback はこの evidence
scope 外とする。

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

## Overview Run Observation

overview は active runs、completed runs、selected run summary/detail を read/follow-up surface として提供する。

- overview は preset catalog、preset queue action、preset 固有の mutation state/error を表示してはならない
- browser load または overview 上の既存 interaction は `/api/v1/preset-matches` を呼び出してはならない
- active/completed run list はそれぞれの polling cadence と panel-local error state を維持する
- completed run を選択した場合は selected run detail を表示し、既存の run follow-up action は引き続き利用できる
- completed runs がない場合は、selected run detail surface は empty state を表示してよい

## General Registration / Request / Ranking Interaction

games page、submissions page、requests page は、operator-facing write route を使う minimal form surface を持たなければならない。

- successful create 後は、対応 list を即時 refresh してよい
- games page は file selection -> bundle admission -> manifest-derived review and ruleset selection -> activation
  の順に進む。game ID、game version、artifact digest、registration ID の manual field は新規 flow に提供してはならない
- requests page の item は `latest_run_id` を run detail deep-link として表示してよい
- rankings page は completed official run の scope または operator-selected scope から `GET /api/v1/rankings` を呼び、stored snapshot を read-only 表示してよい

run detail page は follow-up action を visibility とともに提供してよい。
