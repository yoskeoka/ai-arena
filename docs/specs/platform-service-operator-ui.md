# Platform Service Operator UI 仕様

## 目的

このドキュメントは、Phase 6 first landing で `Cloudflare Pages` から配信する
minimal operator UI contract を定義する。

対象は operator が active/completed match の確認、preset queue、
completed artifact access を行う最小 surface である。
表現や design system は固定せず、state model、polling cadence、
artifact access の扱いだけを定義する。

## この spec の責務範囲

この spec が定義するもの:

- minimal operator UI の画面要素
- active/completed/detail の client-side polling contract
- preset queue action の最小 interaction
- delegated artifact access metadata の表示順と refresh 振る舞い
- local browser verification が依存してよい stable observation surface

この spec が定義しないもの:

- spectator replay/viewer
- real-time push update
- advanced filtering、search、pagination
- upload UI
- ranking / tournament UI
- Pages Functions や server-side rendering
- 特定 AI agent / interactive skill の導入必須化

## 参照関係

- `docs/specs/platform-frontend-architecture.md`: broader frontend の route / shared / API / auth boundary の正本
- `docs/specs/platform-service-operator-api.md`: operator-facing HTTP route の正本
- `docs/specs/platform-service-read-model.md`: compact row / detail view の正本
- `docs/specs/platform-service-skeleton.md`: first landing topology と Pages 配置方針の正本

## Delivery Shape

first landing の operator UI は `Cloudflare Pages` から配信する static app とする。

この surface は broader frontend の `operator` page family に属する 1 route / 1 page として扱う。

- rendering は browser 上の client-side application で完結してよい
- data fetch は operator-facing HTTP API へ直接行う
- server-side rendering、Server Components、Pages Functions は前提にしない
- implementation は後続 task の view 拡張を見越して component-based UI を採用してよい
- broader frontend の route-first rule に従い、page-specific state、polling、presentation は
  `operator` page 配下へ閉じてよい

## Screen Model

minimal operator UI は 1 画面内に少なくとも次を持つ。

- preset queue panel:
  server-known preset 一覧と enqueue action
- active matches panel:
  `queued|leased|running|persisting` の compact row 一覧
- completed matches panel:
  `completed|failed|canceled` の compact row 一覧
- detail panel:
  selected submission の compact summary、submitted players、
  replay input locator group、artifact access action

local browser verification は、少なくとも次の acceptance surface を同一画面で観測できなければならない。

- preset queue panel が visible で、少なくとも 1 件の enqueue action を押せる
- active matches panel が visible で、`queued|leased|running|persisting` item を 0 件以上表示できる
- completed matches panel が visible で、`completed|failed|canceled` item を 0 件以上表示できる
- completed detail panel が visible で、selected submission の `result_summary` と artifact access entry を表示できる

browser verification lane は少なくとも次の 3 系統で同じ acceptance surface を共有しなければならない。

- fixture local regression lane:
  fixture-seeded backend を使う軽量 regression lane
- real local inspection/capture lane:
  actual `arena-service` と actual `operator-ui` を contributor または AI agent が同一環境で起動し、
  preset queue から completed detail までを操作し、review artifact を保存する lane
- dedicated CI browser lane:
  actual operator API request により preset queue から started/completed state を作り、
  同じ panel / detail / artifact observation surface を継続検証する lane

初期表示では completed matches panel の先頭 item を自動選択してよい。
completed item がない場合は、detail panel は empty state を表示してよい。

## State Model

UI は少なくとも次の client state を持つ。

- operator API base URL
- preset catalog read model
- active match items
- completed match items
- selected submission identity
- selected detail response
- list fetch status:
  `idle|loading|ready|error`
- detail fetch status:
  `idle|loading|ready|error`
- preset enqueue status:
  `idle|submitting|success|error`

state は browser reload をまたいで永続化しなくてよい。
artifact access metadata や detail payload を local storage 等へ保存してはならない。

## Polling Contract

- active matches:
  5 秒 cadence で `GET /api/v1/matches/active` を poll してよい
- completed matches:
  10 秒 cadence で `GET /api/v1/matches/completed` を poll してよい
- selected detail:
  submission が選択されている間、15 秒 cadence で
  `GET /api/v1/matches/{submission_id}` を再取得してよい

list polling と detail polling は independent timer でよい。
1 回の request failure で polling loop 全体を停止してはならない。
失敗時は最後に成功した表示を維持しつつ、panel ごとに error state を表示してよい。

## Preset Queue Interaction

preset queue panel は、server-known preset catalog を operator が 1 click で enqueue できる surface とする。

- 1 action は 1 preset request だけを送る
- UI は request body に player list や `artifact_ref` override を含めてはならない
- successful enqueue 後は active matches panel を即時 refresh してよい
- `submission_id` や `match_id` の manual override は first landing では必須にしない

first landing では preset catalog 自体を static UI config として配ってよい。
ただし enqueue request の `preset_id` は API contract に従う server-known value と一致しなければならない。

successful enqueue 後、local browser verification は preset queue action により
active matches panel の row 増加または newly queued submission の可視化を観測できなければならない。

real local inspection/capture lane と dedicated CI browser lane では、
preset queue action のあと actual service backend が `queued|leased|running|persisting|completed`
のいずれかへ進むことを backend poll で待ち、
browser reload または subsequent polling により completed row / detail 表示まで到達できなければならない。

## Detail And Artifact Access

detail panel は `result-summary` を primary observation entry とし、
残りの artifact は secondary access として表示する。

artifact access entry は少なくとも次を表示してよい:

- artifact kind
- stable locator
- delegated access status
- issuer
- expiry
- delegated download action

`download_url` がある artifact kind は direct link action を表示してよい。
`download_url` がない artifact kind は unsupported / locator-only state を明示してよい。

したがって shared browser acceptance では、
file-backed lane は `locator-only` artifact entry でも合格としてよく、
durable artifact lane は delegated download action を少なくとも `result-summary` で表示できなければならない。

## Stable Observation Surface

browser automation は、semantic role / visible text を第一選択として観測してよい。
ただし panel 境界、queue action、row identity、artifact entry のように
styling や copy edit で壊れやすい箇所には、最小限の machine-readable hook を追加してよい。

first landing の operator UI は、少なくとも次の hook family を stable contract として持つ。

- panel root:
  `data-testid="operator-panel-<name>"`
- preset queue action:
  `data-testid="preset-queue-action-<preset-id>"`
- active / completed row:
  `data-testid="match-row-<submission-id>"`
- detail root:
  `data-testid="match-detail-<submission-id>"`
- artifact entry:
  `data-testid="artifact-entry-<artifact-kind>"`

ここで `<name>` は `preset-queue`、`active-matches`、`completed-matches`、
`completed-detail` のいずれかを使う。

machine-readable hook の役割は browser verification の stable targeting に限る。
style hook や analytics 用途と混在させてはならない。

CI の browser workflow は failure 時に少なくとも次を artifact として回収しなければならない。

- Playwright screenshot
- Playwright trace / video
- backend startup/runtime log
- frontend dev-server log

real local inspection/capture lane は success 時にも review artifact を保存できなければならない。

- completed detail screenshot
- Playwright trace
- backend startup/runtime log
- frontend dev-server log

これらの artifact は repo-local ignored path に保存してよい。
Git commit を前提にしてはならない。

## Expiry And Reload Behavior

delegated artifact access metadata は短寿命でよく、UI はそれを durable state として扱ってはならない。

- selected detail の periodic refresh 時は delegated access metadata を再取得してよい
- operator は manual refresh action で selected detail を再取得してよい
- expired link を client が再利用し続けることは保証しない
- UI は expiry 到達前後で link を local cache に退避しない

operator が link 失効を観測した場合の recovery は、
selected detail refresh による再取得を第一導線とする。

## Styling Constraint

first landing の UI は minimal operator surface に限定する。

- CSS framework は 1 つに固定してよい
- visual design より state visibility と action clarity を優先する
- `active`、`completed`、`detail`、`error` の区別が即座に読めることを優先する

## Deferred Follow-Ups

- server-driven preset catalog
- routing / deep-linking
- pagination、search、filter
- authentication / authorization
- replay viewer と ranking / tournament view
