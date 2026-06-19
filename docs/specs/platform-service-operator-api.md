# Platform Service Operator API 仕様

## 目的

このドキュメントは、Phase 6 の first remote landing で
`arena-service` が expose する operator-facing HTTP API を定義する。

対象は private / operator surface であり、spectator 向け public API ではない。
この spec が固定するのは route contract、preset queue 入口、
match request / run follow-up 操作、active/completed polling shape、
および completed detail の返却形である。

## この spec の責務範囲

この spec が定義するもの:

- backend process の HTTP route 一覧
- game registration / AI submission registration route
- match request route
- preset match enqueue request/response contract
- retry / rerun / correction route
- active/completed polling response shape
- completed match detail の response shape
- `Cloudflare Pages` hosted operator UI からの cross-origin access contract
- in-process worker loop と HTTP handler の責務境界

この spec が定義しないもの:

- public spectator API
- delegated artifact access の provider-specific signing 実装

## 参照関係

- `docs/specs/platform-service-skeleton.md`: submission / admission / queue lifecycle の正本
- `docs/specs/platform-service-read-model.md`: compact row / detail view の正本
- `docs/specs/platform-service-persistence.md`: durable write model と locator 保存境界の正本
- `docs/specs/platform-service-general-submission.md`: general operator lane entity / validation の正本
- `docs/specs/platform-service-match-request-scheduling.md`: match request / match-run-group の正本
- `docs/specs/platform-service-ranking-lifecycle.md`: official run / correction の正本
- `docs/specs/platform-product-auth.md`: login / session / operator authorization の正本

## Topology

first remote landing の `arena-service` は、1 つの backend process に次を同居させてよい。

- operator-facing HTTP API
- durable queue store client
- in-process worker loop

この process は single logical queue authority として振る舞う。
複数 process / 複数 worker を同時 support 済みとは扱わない。

HTTP request handling と queue progression は同じ process に同居してよいが、
責務は次のように分ける。

- HTTP handler:
  request decode、preset lookup、service command / query 呼び出し、JSON response
- service command:
  admission validation、retry / rerun / correction mutation
- worker loop:
  `Claim` 後の lifecycle progression と runner/persist orchestration

HTTP handler は `queued -> leased` 以降の状態を直接書き換えてはならない。

current remote landing では operator UI が `Pages`、backend が `Render` の別 origin になるため、
operator API は browser-based cross-origin access を許可しなければならない。

- allow 対象 origin は current canonical operator frontend URL に限定してよい
  - `https://staging.ai-arena.pages.dev`
  - `https://ai-arena.pages.dev`
- browser preflight (`OPTIONS`) を受け付けなければならない
- allow 対象 origin への credentials 付き request を許可しなければならない
- allow 対象外 origin に対して browser use を成功させる必要はない

## Health Route

- `GET /healthz`
  - 目的:
    process が HTTP request を受け付けていることを確認する
  - success response:
    `200 OK`

health route は queue backend や artifact backend の full readiness probe を保証しない。

## Auth Companion Routes

operator surface と同じ backend process は、auth companion route を同居させてよい。

- `GET /auth/session`
- `GET /auth/github/login`
- `GET /auth/github/callback`
- `POST /auth/logout`

これらの route の detailed contract は
`docs/specs/platform-product-auth.md` を正本とする。

## General Registration Routes

以下の operator route family は、auth-enabled mode では authenticated operator session を要求する。

### Game Registrations

- `GET /api/v1/game-registrations`
  - 目的:
    operator lane で利用可能な registered game metadata view を返す
- `POST /api/v1/game-registrations`
  - 目的:
    1 件の `game registration` を validation して保存する

request body は少なくとも次を受け付けてよい:

- `registration_id`
  - omit 時は service が deterministic default を補ってよい
- `game`

response は少なくとも次を返す:

- accepted `registration_id`
- `game`
- `build_mode`
- `builder_id`
- `supported_rulesets`

### AI Submissions

- `GET /api/v1/ai-submissions`
  - 目的:
    operator lane で利用可能な admitted AI artifact identity を返す
- `POST /api/v1/ai-submissions`
  - 目的:
    1 件の `AI submission` を validation して保存する

request body は少なくとも次を受け付けてよい:

- `ai_submission_id`
  - omit 時は service が一意値を生成してよい
- `game_registration_id`
- `artifact_ref`
- `display_name`

response は少なくとも次を返す:

- accepted `ai_submission_id`
- `game_registration_id`
- `game`
- `ai_id`
- `runtime_kind`
- `validation_state`

## Match Requests

- `GET /api/v1/match-requests`
  - 目的:
    operator lane で accepted 済みの match request を返す
- `POST /api/v1/match-requests`
  - 目的:
    1 件の `match request` を validation し、minimal scheduling policy で first run を queue へ流す

request body は少なくとも次を受け付けてよい:

- `request_id`
  - omit 時は service が一意値を生成してよい
- `game_registration_id`
- `participants[]`
  - `player_id`
  - `ai_submission_id`
- `output_dir`
- `match_id`
  - omit 時は service が logical match id を一意生成してよい

response は少なくとも次を返す:

- accepted `request_id`
- `game_registration_id`
- `game`
- `participants[]`
- `source`
- `match_id`
- `latest_run_id`
- `official_run_id`
- `lifecycle_state`

## Preset Match Enqueue

- `POST /api/v1/preset-matches`
  - 目的:
    server-known preset definition から 1 件の logical match と first run を生成して queue へ積む

request body は少なくとも次を受け付けてよい:

- `preset_id`
- `run_id`
  - omit 時は service が一意値を生成してよい
- `match_id`
  - omit 時は service が logical match id を一意生成してよい
- `output_dir`
  - omit 時は preset default を使ってよい

response は少なくとも次を返す:

- accepted record の `run_id`
- accepted record の `match_id`
- `lifecycle_state`
- `official`
- compact row と同じ game metadata

request の制約:

- `preset_id` は server 側に登録済みの preset だけを受け付ける
- request は preset definition 自体の player list や `artifact_ref` を override してはならない
- queue へ積まれる時点では、具体化後の first run が
  `docs/specs/platform-service-skeleton.md` の admission contract を満たしていなければならない

preset lane は queue mutation の直前に、
対応する `game registration` と participant `AI submission` を materialize してよい。
この materialize は same-process 同期処理でよく、
その後は `docs/specs/platform-service-match-request-scheduling.md` が定義する
general `match request` と同じ scheduling 入口へ正規化されなければならない。

## Run Follow-Up Routes

### Retry

- `POST /api/v1/runs/{run_id}/retry`
  - 目的:
    failed run を対象に same `match_id` へ new retry run を append する

response は少なくとも次を返す:

- created retry run の `run_id`
- `match_id`
- `attempt_count`
- `official`
- `lifecycle_state`

制約:

- target run は `failed` でなければならない
- target run が `completed` の場合は `409 Conflict` を返す

### Rerun

- `POST /api/v1/runs/{run_id}/rerun`
  - 目的:
    completed run を対象に same `match_id` へ new rerun candidate を append する

response は少なくとも次を返す:

- created rerun run の `run_id`
- `match_id`
- `attempt_count`
- `official`
- `lifecycle_state`

制約:

- target run は `completed` でなければならない
- created rerun candidate は success 後も automatic official replacement を行ってはならない

### Correction / Promote

- `POST /api/v1/runs/{run_id}/promote`
  - 目的:
    completed run を same `match_id` の `official_run_id` へ切り替える

response は少なくとも次を返す:

- promoted run の `run_id`
- `match_id`
- resulting `official_run_id`

制約:

- target run は `completed` でなければならない
- same `match_id` に属する他 run の `official` は false へ戻さなければならない

## Polling Routes

### Active List

- `GET /api/v1/matches/active`
  - 目的:
    `queued|leased|running|persisting` の run だけを compact row で返す

response body は JSON object とし、少なくとも次を持つ:

- `items`
  - `docs/specs/platform-service-read-model.md` の compact row 配列

### Completed List

- `GET /api/v1/matches/completed`
  - 目的:
    `completed|failed|canceled` の run だけを compact row で返す

response body は JSON object とし、少なくとも次を持つ:

- `items`
  - `docs/specs/platform-service-read-model.md` の compact row 配列

### Completed Detail

- `GET /api/v1/runs/{run_id}`
  - 目的:
    1 run の detail view を返す

response body は、`docs/specs/platform-service-read-model.md` の detail view に加えて、
artifact kind ごとの derived delegated access metadata を含んでよい。

detail response は少なくとも次の識別子を返さなければならない。

- `run_id`
- `match_id`
- `attempt_count`
- `official`

delegated access metadata は少なくとも次の考え方を満たす:

- stable locator は write model / detail view 側に残す
- short-lived download URL や同等 token は request 時に派生させる
- object bytes 自体は backend process を経由しない

## Error Contract

この milestone の route は JSON error object を返してよい。
少なくとも次を使い分ける。

- `400 Bad Request`
  - request body / query / path parameter が不正
- `404 Not Found`
  - unknown `run_id` または unknown `preset_id`
- `409 Conflict`
  - invalid lifecycle target、duplicate `run_id`、同一 match 内 official conflict など
- `500 Internal Server Error`
  - queue backend、artifact backend、unexpected service failure

## Worker Loop Behavior

backend process 内の worker loop は少なくとも次を満たす。

- loop は polling interval ごとに queued run を claim してよい
- queue が空のときは異常終了せず、次回 poll を待つ
- 1 件の run failure により process 全体を停止させない
- worker loop が lifecycle を進めた結果は、active/completed polling route から観測できなければならない
- completed run が automatic official promotion 対象かどうかは、
  same `match_id` の existing official 状態を見て判断しなければならない

first landing では 1 backend process あたり 1 worker identity で十分とする。
