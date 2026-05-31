# platform-online-foundation-03-02-remote-service-topology-and-polling-api
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`arena-service` を `Render` 上の first backend process として常駐させ、
Phase 6 の online confirmation に必要な最小 polling API を remote で成立させる。
最初のゴールは、preset match enqueue、active/completed list、completed detail を
`Neon Postgres + R2` を前提に remote service として通すことに置く。

## Context

- `0056` / `0057` / `0058` により、durable queue lifecycle、read model、artifact locator はすでに local/CLI では成立している
- user decision として、まずは `Render` で実際に service を動かし、free/low-cost lane の fit を見たい
- この段階で必要なのは本格 matchmaking ではなく、preset 組み合わせを queue へ積み、進行中と終了後を確認できる remote surface である
- UI 側は simple polling でよく、push/SSE/WebSocket はまだ不要である

## Scope

- `Render` 上で動く single logical queue authority としての backend topology を定義する
- first landing は `operator-facing HTTP API + in-process worker loop` の同居構成で始める
- preset match enqueue endpoint を定義する
- `queued|leased|running|persisting` を返す active list と、`completed|failed|canceled` を返す completed list を定義する
- completed match detail は compact summary と artifact locator / delegated access metadata を返す

この plan では以下を扱わない。

- general AI upload registry
- general game registration portal
- ranking 集計
- automatic matchmaking
- multi-worker fairness / distributed queue

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- remote first landing の topology と single-process first deployment shape を追記する
- `operator-facing HTTP API + in-process worker loop` を first remote lane として明記する

### `docs/specs/platform-service-read-model.md`

- active list、completed list、completed detail の polling contract を追記する
- default result observation は `result-summary` first であることを UI 向け surface にも反映する

### 新規または更新 spec

- preset match enqueue request/response contract を追加する
- backend process の HTTP route と state transition visibility contract を追加する

## Expected Code Changes

- HTTP server / routing surface for `arena-service`
- in-process worker loop or equivalent service-owned poller
- preset queue endpoint
- active/completed/detail query endpoints
- remote topology integration test

## Sub-tasks

- [ ] first landing backend topology と process responsibility を spec に落とす
- [ ] preset match enqueue surface を定義する
- [ ] active/completed/detail polling API を定義する
- [ ] in-process worker loop と HTTP API の干渉境界を整理する
- [ ] remote service と durable backend をまとめて検証する integration path を定義する

## Parallelism

- enqueue surface と read API surface の spec 整理は並行できる
- topology 固定後は HTTP server 実装と worker loop 接続を分担できる

## Dependencies

- depends on: `0064-platform-online-foundation-03-01-provider-bootstrap-and-remote-artifact-delivery.md`
- depends on: `0057-platform-online-foundation-02-02-result-read-model-and-operator-query.md`
- depends on: `0058-platform-online-foundation-02-03-replay-resume-audit-inputs.md`
- depends on: parent/base item `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md` (now retired to `docs/exec-plan/done/` after split)
- blocks: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`
- informs: `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md`

## Risks and Mitigations

- queue progression と HTTP request handling を同じ process に寄せると責務が混ざりやすい
  - mitigation: first landing は同居でよいが、API / queue authority / worker loop の責務境界を spec 上で先に切る
- preset queue 導線に AI upload registry や general matchmaking を混ぜると scope が膨らむ
  - mitigation: request は server-known preset/reference participants に限定し、general submission は後続へ残す
- completed detail で artifact bytes を返し始めると service が重くなる
  - mitigation: detail は compact summary + locator + delegated access metadata だけを返す

## Design Decisions

- `Render` first landing は backend process を常駐させやすい利点をそのまま使う
- first remote deployment は `service API + worker loop` 同居で始め、separate worker service は後続判断に残す
- Phase 6 remote confirmation に必要な queue visibility は polling API で十分とする
