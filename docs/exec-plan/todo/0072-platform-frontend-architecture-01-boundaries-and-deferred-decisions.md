# platform-frontend-architecture-01-boundaries-and-deferred-decisions
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`operator-ui` の一時的な 1 file 分割方針ではなく、
今後の Phase 7 operator/admin 画面、game catalog / search、leaderboard、
将来の public watch/viewer をどういう frontend 境界で収めるかを先に固定する。
最初のゴールは、`ai-arena` に必要な範囲だけで frontend architecture contract を定義し、
今決めることと後で決めることを分けたうえで、以後の配置・分割ルールの正本を作ることに置く。

## Context

- `docs/issues/0026-operator-ui-component-state-refactor.md` は、
  `operator-ui/src/App.tsx` の肥大化を問題としているが、
  実際の論点は将来の画面追加に耐える責務境界を先に決めることにある
- `docs/specs/platform-service-operator-ui.md` は Phase 6 first landing の minimal operator surface を定義しているが、
  broader frontend architecture まではまだ固定していない
- user decision として、GraphQL は避け、frontend 専用 API と外部向け API は分け、
  backend 内部ロジックは共通化しつつ成形 layer を分けたい
- user decision として、browser 向け認証の第一選択は same-origin + http-only cookie とし、
  token を browser storage へ置く前提には寄せない
- user decision として、viewer の具体技術設計や auth mechanism の詳細は今は固定しない

## Scope

- `ai-arena` frontend の route / module / shared code の責務境界を定義する
- route-first 配置、page-local data fetch / state、shared primitive の範囲を定義する
- frontend 専用 API と外部向け API を分ける方針を spec に明記する
- browser 向け認証の第一選択を same-origin cookie に置くことを境界として明記する
- viewer、global session cache、詳細 auth flow の defer 範囲を明記する

この plan では以下を扱わない。

- `operator-ui` の実コード refactor
- login UI や session refresh の実装
- viewer の実装方式確定
- query cache library の導入
- external public API schema の実装

## Spec Changes

### 新規 spec

- `docs/specs/platform-frontend-architecture.md`
  - route-first frontend structure の正本
  - `routes/<page>`、`shared/ui`、`shared/layout`、将来の `features/<domain>` 導入条件
  - page-local fetch / local polling を基本とする data boundary
  - frontend 専用 API と external API を分け、backend 内部ロジックは共有しうること
  - browser auth は same-origin cookie first とする境界
  - viewer / auth / global session cache の deferred decision

### 更新 spec

- `docs/specs/platform-service-operator-ui.md`
  - operator UI を broader frontend の 1 route / 1 page family として位置づける
  - Phase 6 minimal operator surface と broader frontend architecture spec の参照関係を補足する

## Expected Code Changes

- なし。planning / spec only。

## Sub-tasks

- [ ] `0026` から見える frontend growth path を整理し、operator/admin、catalog/search、leaderboard、watch/viewer の責務境界を定義する
- [ ] route-first 配置と shared code ルールを定義する
- [ ] page-local data fetch / local polling を default とし、global cache を defer する条件を定義する
- [ ] frontend 専用 API と external API を分ける方針を定義する
- [ ] same-origin cookie first と deferred auth details を定義する
- [ ] viewer boundary と deferred technical decisions を明文化する

## Parallelism

- [parallel] route / directory boundary 整理と API / auth boundary 整理は並行できる
- deferred decision の棚卸しは、上記 boundary 案が固まった後に depends on する

## Dependencies

- depends on: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`
- depends on: `0070-platform-online-foundation-03-05-operator-ui-verification-03-real-local-browser-operator-lane.md`
- informs: `0073-platform-frontend-architecture-02-operator-ui-route-first-refactor.md`
- informs: `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md`

## Risks and Mitigations

- 先に library や framework trend を決め始めると、`ai-arena` に不要な complexity を背負う
  - mitigation: route boundary、shared boundary、API/auth boundary のような durable rule だけを先に固定する
- `shared` を広く取りすぎると、再利用名目の曖昧な component 置き場になる
  - mitigation: `shared/ui` は primitive のみ、cross-route の header/footer/menu は `shared/layout` へ分離し、page 固有物を置かない
- data fetch policy を広く抽象化しすぎると、単純な page fetch まで複雑になる
  - mitigation: default は page initial fetch + local polling とし、global cache は session-like data が本当に重複し始めてから検討する
- external API と frontend API を混ぜると、後方互換と UI 要件が同じ route schema に載って硬直化する
  - mitigation: public/external contract と frontend contract は transport layer を分け、backend 内部ロジックだけ共有する

## Design Decisions

- frontend 配置は route-first を第一選択とする
- `shared/ui` は primitive に限定し、header/footer/menu のような cross-route structure は `shared/layout` へ置く
- state と fetch は page 配下へ閉じ、default は initial fetch + local polling とする
- browser 向け認証の第一選択は same-origin + http-only cookie とする
- viewer の具体技術、query cache library、detailed auth flow は follow-up decision として defer する
