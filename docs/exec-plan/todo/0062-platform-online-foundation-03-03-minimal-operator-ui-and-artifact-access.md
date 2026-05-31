# platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 6 の online confirmation UI として、simple polling だけで
active/completed match の確認、preset queue、completed artifact access を行える最小 operator surface を整える。
最初のゴールは、表現にこだわらずに `Cloudflare Pages` 上で動く最小 UI を作り、
`result-summary` と主要 artifact を completed detail から辿れるようにすることに置く。

## Context

- user decision として、UI は Phase 6 から省かず、表現は後回しにしてまず運営確認面を持つ
- `0061` が remote polling API を提供できれば、UI は server-side push なしでも成立する
- completed artifact access は Phase 6 persisted artifact を online service として読めることの確認に直結する
- object bytes は backend を通さず、remote object storage provider の delegated download を使うべきである

## Scope

- `Cloudflare Pages` で配信する最小 operator UI を定義する
- active match list を polling で表示する
- completed match list を polling で表示する
- preset match queue 登録ボタンを提供する
- completed match detail から `result-summary` と主要 artifact への delegated access link を表示する

この plan では以下を扱わない。

- advanced design system
- spectator replay/viewer
- real-time push update
- upload UI
- ranking / tournament UI

## Spec Changes

### `docs/specs/platform-service-read-model.md`

- UI が期待する active/completed/detail payload shape を補足する
- completed detail の artifact access field を UI consumption 前提で明記する

### 新規または更新 spec

- minimal operator UI state model と polling cadence を定義する
- delegated artifact access link の表示ルールと expiry 取り扱いを定義する

## Expected Code Changes

- Pages app scaffold
- active/completed/detail polling client
- preset queue mutation client
- artifact access link rendering
- UI smoke / acceptance verification

## Sub-tasks

- [ ] minimal operator UI の画面要素と state model を spec に落とす
- [ ] active/completed polling cadence と error handling を定義する
- [ ] preset queue action と completed detail 表示を組み込む
- [ ] delegated artifact access link の expiry / reload behavior を整理する
- [ ] Pages 上での smoke verification を追加する

## Parallelism

- UI state model 整理と Pages scaffold 準備は並行できる
- API contract 固定後は list view と detail/action view を分担できる

## Dependencies

- depends on: `0060-platform-online-foundation-03-01-provider-bootstrap-and-remote-artifact-delivery.md`
- depends on: `0061-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- depends on: parent/base item `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md` (now retired to `docs/exec-plan/done/` after split)
- informs: `0063-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md`

## Risks and Mitigations

- UI を凝り始めると Phase 6 confirmation より見た目の調整に時間を使う
  - mitigation: Pages UI は minimal operator surface に限定し、CSS framework choice は execution 時に 1 つ選んで深入りしない
- delegated artifact access link が短命すぎると detail 表示中に失効しやすい
  - mitigation: detail refresh 時に再発行できる contract を持たせ、link 永続化はしない
- backend の compact summary より先に raw artifact 表示へ寄ると観測導線がぶれる
  - mitigation: default observation は `result-summary` first とし、詳細 artifact は secondary action に留める

## Design Decisions

- UI delivery は `Cloudflare Pages` first とする
- first operator UI は simple polling 前提でよい
- artifact access は delegated download link を表示するだけに留め、object bytes の proxy はしない
