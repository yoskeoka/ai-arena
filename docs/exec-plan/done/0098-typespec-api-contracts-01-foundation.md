# typespec-api-contracts-01-foundation
**Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

`ai-arena` に repo-owned な TypeSpec project を導入し、
operator-facing HTTP API の route / request / response schema を
Markdown ではなく TypeSpec source で管理する基盤を成立させる。

この plan のゴールは、operator API の wire-level contract を
`docs/specs/*.md` から外して TypeSpec へ移し、
`docs/specs/index.md` から typed source of truth を辿れるようにしたうえで、
`docs/specs/` には topology、責務境界、auth companion、polling/worker behavior のような
behavioral contract だけを残すことにある。

加えて、将来 `ai-arena` の API family が
operator 向け、public/external 向け、operator-ui 以外の private consumer 向けへ増えても、
同じ TypeSpec project 内で shared model と group-specific namespace/file を分けて
安全に拡張できる構成を先に固定する。

完了条件:

- `ai-arena` 配下に `docs/specs/` とは別の TypeSpec project が追加されている
- current operator API family の request/response/route schema が TypeSpec source of truth へ移っている
- operator group 向けに OpenAPI artifact と frontend-consumable client generation seam が定義されている
- `docs/specs/index.md` から operator HTTP contract と AI Arena JSON-RPC contract の typed source of truth を辿れる
- `docs/specs/platform-service-operator-api.md` は削除され、API 仕様を読む入口が Markdown 一覧ではなく TypeSpec / typed code へ揃っている
- `docs/specs/platform-service-operator-ui.md` は API 呼び出し詳細を重複保持せず、必要箇所だけ TypeSpec-owned contract を参照する
- `AGENTS.md` に
  「API schema は TypeSpec で管理し、`docs/specs/` は外形的振る舞いを書く」
  ルールが追記されている

## Context

- user は `spec code parity` 自体は維持したいが、
  frontend と API の関係を Markdown だけで保守するのは不向きなので、
  widely known な専用機構として TypeSpec を導入したいと明示している
- user は API spec source を `docs/specs/` 配下へ焼くことを望んでいない
- `docs/specs/README.md` は `docs/specs/` を product / platform / runtime / game contract の正本としつつ、
  contributor workflow や repo-local 運用の混入を避けるよう求めている
- `docs/specs/platform-frontend-architecture.md` は
  frontend API と external/public API の boundary を分けてよいと明記している
- current operator API contract は
  `docs/specs/platform-service-operator-api.md` と
  `operator-ui/src/api.ts` に二重化されている
- current Go implementation は
  `internal/platform/service/http.go` で route/handler を直接登録しており、
  TypeSpec 導入後も backend 実装と schema source の parity 管理が必要になる

## Option Snapshot

### Option A: 1 つの `main.tsp` に全 API family を置き、`@tag` だけで group を分ける

- 利点:
  初期導入の file 数が少ない
- 欠点:
  operator / public / future private consumer が増えたときに
  source ownership と review scope が混ざりやすい

### Option B: 1 つの TypeSpec project の中で、
### shared model + group-specific namespace/file split を採用し、
### emitted OpenAPI grouping には `@tag` も併用する

- 利点:
  source ownership と future API family split を最初から表現できる
- 欠点:
  初期導入時に directory 設計と compile entrypoint を決める必要がある

### Option C: API family ごとに別 TypeSpec project を作る

- 利点:
  group 間の accidental coupling を最小化しやすい
- 欠点:
  shared model の再利用、versioning、compile orchestration が過剰に重くなる

## Recommendation

Option B を採る。

- source tree は `docs/specs/` の外に独立配置する
- TypeSpec project は 1 つに保ち、
  shared model と group-specific namespace/file split で future API family seam を固定する
- emitted artifact 側では `@tag` を使って
  consumer-facing grouping も失わないようにする
- current execution では `operator` group を first landing とし、
  `public` や他 private consumer group は future expansion seam として directory/namespace だけ先に揃える

## Scope

- TypeSpec project の配置先と compile/configuration contract を決める
- shared model / operator group / future sibling group を置ける source tree を決める
- current operator API family の route / request / response schema を TypeSpec へ移す
- OpenAPI emit と frontend-consumable client generation の command/output seam を決める
- `docs/specs/index.md` を追加し、operator / JSON-RPC contract の参照入口を整理する
- `docs/specs/platform-service-operator-api.md` を削除し、Markdown 一覧を source of truth にしない状態へ寄せる
- `docs/specs/platform-service-operator-ui.md` から API call 詳細の重複記述を減らす
- `AGENTS.md` に API spec management rule を追加する

この plan では以下を扱わない。

- generated client を実際に `operator-ui` route codeへ組み込む refactor
- backend handler 実装を TypeSpec source から自動生成すること
- public/external API family 自体の新規設計

## Existing Implementation References

- `docs/specs/README.md`
  - spec placement rule, lines 1-21
- `docs/specs/platform-service-operator-api.md`
  - 削除前の route/response contract summary, lines 1-449
- `docs/specs/platform-service-operator-ui.md`
  - API-referencing UI contract, lines 35-40, 42-72, 180-297
- `docs/specs/platform-frontend-architecture.md`
  - API family boundary and operator route-first boundary, lines 120-131, 152-190
- `AGENTS.md`
  - spec-writing discipline and child-repo doc policy, lines 1-38
- `internal/platform/service/http.go`
  - `OperatorAPI`, `Handler`, route handlers, CORS/error contract, lines 14-160, 175-213, 239-513
- `operator-ui/src/api.ts`
  - handwritten DTOs and `OperatorApiClient`, lines 1-456
- `operator-ui/package.json`
  - current frontend script surface, lines 1-32

## Code Change Map

- `typespec/package.json` (NEW)
  - TypeSpec compiler/emitter dependencies and repo-owned generation scripts
- `typespec/tspconfig.yaml` (NEW)
  - project boundary, emitter/output configuration, shared output dirs
- `typespec/main.tsp` (NEW)
  - project entrypoint and shared imports
- `typespec/namespaces/shared.tsp` (NEW)
  - DTOs and reusable models shared across API groups
- `typespec/namespaces/operator/*.tsp` (NEW)
  - operator group routes, payloads, response models, and tags
- `typespec/namespaces/public/*.tsp` (NEW)
  - future public/external namespace seam placeholder kept distinct from operator group
- `typespec/generated/openapi/operator/openapi.json` (NEW)
  - checked-in emitted OpenAPI artifact for the operator group
- `operator-ui/src/generated/operator-api/*` (NEW)
  - checked-in emitted frontend client/types for the operator group, without adoption refactor yet
- `docs/specs/index.md` (NEW)
  - TypeSpec と typed Go contract の lookup index
- `docs/specs/platform-service-operator-api.md` (DELETE)
  - remove duplicated Markdown API inventory after TypeSpec/index migration
- `docs/specs/platform-service-operator-ui.md` (MODIFY)
  - remove duplicated request/response detail prose and point to TypeSpec-owned operator contract
- `AGENTS.md` (MODIFY)
  - add API schema management rule that TypeSpec owns wire contracts outside `docs/specs/`

## Spec Changes

- `docs/specs/index.md`
  - route/path/body/response schema は TypeSpec、AI Arena JSON-RPC payload は typed Go code を見る導線を追加する
- `docs/specs/platform-service-operator-ui.md`
  - UI が依存する route family や polling cadence は残しつつ、
    body/query/response field の重複説明は TypeSpec source を参照する形へ寄せる

## Sub-tasks

- [ ] TypeSpec project の配置先を `docs/specs/` 外で固定する
- [ ] shared model と group-specific namespace/file split の source tree を固定する
- [ ] emitted OpenAPI と emitted frontend client の output path を固定する
- [ ] current operator API family の route / request / response schema を TypeSpec へ移す
- [ ] `docs/specs/index.md` を追加し、typed contract の lookup entry を固定する
- [ ] `docs/specs/platform-service-operator-api.md` を削除し、TypeSpec / typed code を source of truth に揃える
- [ ] `docs/specs/platform-service-operator-ui.md` から API call detail の重複を外す
- [ ] `AGENTS.md` へ API spec management rule を追加する

## Parallelism

- [parallel] TypeSpec project layout の整理と
  Markdown spec の責務再整理は並行できる
- [parallel] shared model 設計と emitted artifact output path 設計は並行できる
- generated client adoption を伴う UI refactor は別 child plan に分離し、
  この plan の完了に depends on させる

## Dependencies

- context from: `docs/specs/platform-service-general-submission.md`
- context from: `docs/specs/platform-service-match-request-scheduling.md`
- context from: `docs/specs/platform-service-ranking-lifecycle.md`
- context from: `docs/specs/platform-product-auth.md`

## Verification

- `typespec/` project が compile できる
- operator group 向け OpenAPI artifact が repo-owned command で再生成できる
- operator group 向け frontend client artifact が repo-owned command で再生成できる
- `docs/specs/` 配下に operator API の field inventory Markdown が残っていない
- `AGENTS.md` から
  API wire contract は TypeSpec、
  `docs/specs/` は observable behavior という境界が読める

## Risks and Mitigations

- TypeSpec source を 1 file に集約しすぎると future API family split が難しくなる
  - mitigation:
    source は shared + group-specific namespace/file split を最初から採る
- generated artifact の出力場所を曖昧にすると、
  `docs/specs/` へ drift したり frontend import が不安定になったりする
  - mitigation:
    `docs/specs/` 外で source と generated output path を明示する
- Markdown spec を削りすぎると topology/behavior contract まで失う
  - mitigation:
    TypeSpec へ移すのは wire-level schema に限定し、
    behavioral/operator-surface contract は Markdown に残す

## Design Decisions

- API wire contract の source of truth は TypeSpec に置く
- TypeSpec source は `docs/specs/` 配下に置かない
- source layout は 1 project + shared/common + group-specific namespace/file split を基本にする
- emitted artifact grouping には `@tag` を併用してよいが、
  source ownership の分離は file/namespace split で先に表現する
