# platform-frontend-architecture-02-operator-ui-route-first-refactor
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0026-operator-ui-component-state-refactor.md`

## Objective

`0072-platform-frontend-architecture-01-boundaries-and-deferred-decisions.md`
で固定した frontend architecture contract に従い、
current `operator-ui` を route-first / page-local state / shared-minimal の形へ寄せる。
最初のゴールは、minimal operator surface と既存 browser verification contract を壊さずに、
`App.tsx` の state、polling side effect、presentation を読みやすい単位へ分けることに置く。

## Context

- current `operator-ui/src/App.tsx` は active/completed polling、detail refresh、
  preset enqueue、base URL handling、helper UI、panel rendering が 1 file に集まっている
- `docs/issues/0026-operator-ui-component-state-refactor.md` は、
  verification lane が整ったら責務分割方針を先に決めて refactor したいとしている
- `0068` / `0069` / `0070` により、minimal operator surface を検証する browser lane はすでにある
- user decision として、今の refactor は「500 行をどう切るか」ではなく、
  将来の frontend 配置ルールに従って first page を置き直すことが目的である

## Scope

- `operator-ui` の current entry を route-first layout へ再配置する
- operator page 固有の data fetch / polling / selection state を page 配下へ閉じる
- shared primitive と page-specific presentation を分離する
- existing browser verification selector / acceptance surface を維持する
- future pages を追加しやすい app shell / route seam を作る

この plan では以下を扱わない。

- login / session UI の実装
- global query cache 導入
- public watch/viewer route の実装
- leaderboard / catalog / search page の実装
- backend API schema の拡張

## Spec Changes

### `docs/specs/platform-frontend-architecture.md`

- first execution で `operator` page へどう適用するかを補足する
- page-local state と shared primitive の具体境界を execution-ready にする

### `docs/specs/platform-service-operator-ui.md`

- route-first 再配置後も変わらない stable observation surface を確認する
- page split 後も維持すべき active/completed/detail/preset queue contract を必要に応じて補足する

## Expected Code Changes

- `operator-ui/src/` の entry / route structure 再編
- operator page 配下への fetch / polling hook 分離
- page-specific panel / table / detail presentation 分離
- `shared/ui` と必要なら `shared/layout` の最小導入
- selector / acceptance surface を維持した browser verification update

## Sub-tasks

- [ ] `operator-ui` の route / app shell 入口を作り、current operator surface を page として収める
- [ ] active/completed/detail/enqueue の fetch / polling logic を page 配下へ分離する
- [ ] preset queue、match list、detail、shared primitive の presentation 境界を整理する
- [ ] base URL normalization や connection hint のような page support logic を view から分離する
- [ ] browser verification selectors と acceptance surface が維持されることを確認する
- [ ] refactor 後の layout / import rule を docs または code structure で読み取りやすく保つ

## Parallelism

- [parallel] app shell / route scaffold と primitive extraction は並行できる
- [parallel] active/completed list 分離と detail panel 分離は並行できる
- verification update は page split と selector drift 確認に depends on する

## Dependencies

- depends on: `0072-platform-frontend-architecture-01-boundaries-and-deferred-decisions.md`
- depends on: `0068-platform-online-foundation-03-05-operator-ui-verification-01-local-agent-browser-loop.md`
- depends on: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`
- depends on: `0070-platform-online-foundation-03-05-operator-ui-verification-03-real-local-browser-operator-lane.md`
- informs: `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md`

## Risks and Mitigations

- early refactor で page boundary より component split を優先すると、将来の route 拡張に効かない
  - mitigation: まず app shell / route seam を作り、その下で page-local state と presentation を分ける
- selector 名や panel contract を壊すと、既存 browser lane が回帰する
  - mitigation: `platform-service-operator-ui` spec の stable observation surface を維持し、必要なら tests を先に合わせる
- `shared` へ page 固有 UI を移しすぎると、再び責務が曖昧になる
  - mitigation: preset queue、match row、detail panel のような operator 固有 UI は route/page 配下に残す
- global state を先に導入すると、単純な minimal operator page に対して過剰な抽象化になる
  - mitigation: first refactor では page-local state に留め、global cache / auth context は次段判断にする

## Design Decisions

- first refactor は route-first の最小適用として `operator` page を再配置する
- page 固有の data / state / support logic は page 配下へ閉じる
- `shared/ui` は primitive のみ、cross-route layout が必要なら `shared/layout` を最小導入する
- browser verification seam は refactor 後も durability contract として維持する
