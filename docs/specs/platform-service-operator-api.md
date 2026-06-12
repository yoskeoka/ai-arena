# Platform Service Operator API 仕様

## 目的

このドキュメントは、Phase 6 の first remote landing で
`arena-service` が expose する operator-facing HTTP API を定義する。

対象は private / operator surface であり、spectator 向け public API ではない。
この spec が固定するのは route contract、preset queue 入口、
active/completed polling shape、および completed detail の返却形である。

## この spec の責務範囲

この spec が定義するもの:

- backend process の HTTP route 一覧
- preset match enqueue request/response contract
- active/completed polling response shape
- completed match detail の response shape
- `Cloudflare Pages` hosted operator UI からの cross-origin access contract
- in-process worker loop と HTTP handler の責務境界

この spec が定義しないもの:

- public spectator API
- authentication / authorization の最終方式
- general AI upload / matchmaking / ranking contract
- delegated artifact access の provider-specific signing 実装

## 参照関係

- `docs/specs/platform-service-skeleton.md`: submission / admission / queue lifecycle の正本
- `docs/specs/platform-service-read-model.md`: compact row / detail view の正本
- `docs/specs/platform-service-persistence.md`: durable write model と locator 保存境界の正本

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
  admission validation、queued-only mutation
- worker loop:
  `Claim` 後の lifecycle progression と runner/persist orchestration

HTTP handler は `queued -> leased` 以降の状態を直接書き換えてはならない。

current remote landing では operator UI が `Pages`、backend が `Render` の別 origin になるため、
operator API は browser-based cross-origin access を許可しなければならない。

- allow 対象 origin は current canonical operator frontend URL に限定してよい
  - `https://staging.ai-arena.pages.dev`
  - `https://ai-arena.pages.dev`
- `POST /api/v1/preset-matches` の JSON request に必要な browser preflight (`OPTIONS`) を受け付けなければならない
- allow 対象外 origin に対して browser use を成功させる必要はない

## Health Route

- `GET /healthz`
  - 目的:
    process が HTTP request を受け付けていることを確認する
  - success response:
    `200 OK`

health route は queue backend や artifact backend の full readiness probe を保証しない。

## Preset Match Enqueue

- `POST /api/v1/preset-matches`
  - 目的:
    server-known preset definition から 1 件の `match submission` を生成して queue へ積む

request body は少なくとも次を受け付けてよい:

- `preset_id`
- `submission_id`
  - omit 時は service が一意値を生成してよい
- `match_id`
  - omit 時は service が一意値を生成してよい
- `output_dir`
  - omit 時は preset default を使ってよい

response は少なくとも次を返す:

- accepted record の `submission_id`
- accepted record の `match_id`
- `lifecycle_state`
- compact row と同じ game metadata

request の制約:

- `preset_id` は server 側に登録済みの preset だけを受け付ける
- request は preset definition 自体の player list や `artifact_ref` を override してはならない
- queue へ積まれる時点では、具体化後の `match submission` が
  `docs/specs/platform-service-skeleton.md` の admission contract を満たしていなければならない

first landing の preset catalog は local JSON file や同等の repo-local config から読んでよい。
ただし deploy-shaped remote lane の catalog は、repo source checkout 上の `go run` 前提 entry ではなく、
`make render-build` などの build/start contract だけで実行可能な prepared artifact entry を指さなければならない。
`presets.example.json` のような contributor-facing lightweight example は、
staging / production の canonical preset catalog と同一でなくてよい。
general registration API や DB-backed preset registry は後続へ残す。

## Polling Routes

### Active List

- `GET /api/v1/matches/active`
  - 目的:
    `queued|leased|running|persisting` の submission だけを compact row で返す

response body は JSON object とし、少なくとも次を持つ:

- `items`
  - `docs/specs/platform-service-read-model.md` の compact row 配列

### Completed List

- `GET /api/v1/matches/completed`
  - 目的:
    `completed|failed|canceled` の submission だけを compact row で返す

response body は JSON object とし、少なくとも次を持つ:

- `items`
  - `docs/specs/platform-service-read-model.md` の compact row 配列

### Completed Detail

- `GET /api/v1/matches/{submission_id}`
  - 目的:
    1 submission の detail view を返す

response body は、`docs/specs/platform-service-read-model.md` の detail view に加えて、
artifact kind ごとの derived delegated access metadata を含んでよい。

delegated access metadata は少なくとも次の考え方を満たす:

- stable locator は write model / detail view 側に残す
- short-lived download URL や同等 token は request 時に派生させる
- object bytes 自体は backend process を経由しない

first landing では provider-specific URL signing 未対応の artifact kind があってよい。
その場合も detail view は stable locator を返し、unsupported であることを明示してよい。

## Error Contract

この milestone の route は JSON error object を返してよい。
少なくとも次を使い分ける。

- `400 Bad Request`
  - request body / query / path parameter が不正
- `404 Not Found`
  - unknown `submission_id` または unknown `preset_id`
- `409 Conflict`
  - duplicate `submission_id` など既存 record と衝突
- `500 Internal Server Error`
  - queue backend、artifact backend、unexpected service failure

## Worker Loop Behavior

backend process 内の worker loop は少なくとも次を満たす。

- loop は polling interval ごとに queued record を claim してよい
- queue が空のときは異常終了せず、次回 poll を待つ
- 1 件の match failure により process 全体を停止させない
- worker loop が lifecycle を進めた結果は、active/completed polling route から観測できなければならない

first landing では 1 backend process あたり 1 worker identity で十分とする。
