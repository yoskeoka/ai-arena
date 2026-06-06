# Platform Frontend Architecture 仕様

## 目的

このドキュメントは、`ai-arena` の browser-facing frontend を
どの責務境界で増やしていくかの正本を定義する。

ここで固定するのは durable な route / module / API / auth boundary であり、
特定 framework feature、viewer 実装方式、query cache library の採用有無は固定しない。

## この spec の責務範囲

この spec が定義するもの:

- route-first frontend structure の基本形
- route 配下と shared 配下の責務境界
- page-local fetch / polling / state を default にする data boundary
- frontend 専用 API と external/public API を分ける boundary
- browser 向け auth の第一選択を same-origin + http-only cookie に置く boundary
- deferred decision として後続 plan に残す論点

この spec が定義しないもの:

- operator/admin、catalog/search、leaderboard、viewer 各 route の詳細 UI
- login form、session refresh、role matrix などの detailed auth flow
- SSR、Server Components、edge rendering など rendering strategy の選定
- GraphQL や query cache library の導入判断
- external/public API schema 自体

## Frontend Families

`ai-arena` の frontend は、少なくとも次の page family を受け止められる構造を前提にする。

- operator/admin:
  運営者が submission、match、artifact、queue、管理系 read/write を扱う route 群
- catalog/search:
  game、AI、match、artifact などの discoverability を扱う route 群
- leaderboard:
  ranking、standings、history の read-only view を扱う route 群
- watch/viewer:
  public または broader audience 向けの match watch / replay / spectator view を扱う route 群

この段階では各 family を実装しなくてよい。
ただし新しい page を追加するときは、既存 page の file 分割都合ではなく、
どの family に属する route かを先に固定して配置しなければならない。

## Route-First Structure

frontend の第一選択は route-first structure とする。

- entry point は app shell と route registration だけを持つ
- page 固有の UI、state、fetch、polling、selector は `routes/<page>/` 配下へ置く
- route をまたがない helper を `shared/` に逃がしてはならない
- page 固有の hook、mapper、table、detail panel、empty state は page 配下に残す

最低限の directory family は次を想定してよい。

- `routes/<page>/`:
  page 固有の component、hook、support logic、route-local test
- `shared/ui/`:
  button、badge、table shell、spinner のような primitive
- `shared/layout/`:
  header、footer、nav、page frame のような cross-route structure

`shared/ui` は primitive に限定する。
`operator` 専用 panel や `leaderboard` 専用 row のように route 文脈を前提にする UI は置かない。

route registration の実装方式はこの段階では固定しない。

- pathname dispatch を app shell 内で手書きしてもよい
- router library を採用してもよい
- ただし最初の refactor では、route 配置規則を先に固定し、
  deep-linking や nested route 機能要求が固まる前に library choice を contract にしてはならない

## Import And Export Rule

module 境界は、implementation jump のしやすさを優先する。

- `index.ts` / `index.tsx` は必須にしない
- barrel export は default にしない
- import は原則 direct import とする
- reusable module は named export を基本にする
- default export は framework 的に entry として自然な file に限定してよい

barrel export や re-export hub は、
public package API を整える必要がある場合の convenience としては使えるが、
`ai-arena` frontend では jump cost と依存追跡性を優先する。

したがって、単に import path を短く見せる目的だけで
`routes/<page>/index.ts` や `shared/*/index.ts` を増やしてはならない。

## Feature Module Rule

default は route-first であり、最初から `features/<domain>/` を必須にしない。

`features/<domain>/` の導入は、少なくとも次のすべてを満たすときに限って検討してよい。

- 2 つ以上の route family が同じ domain rule または workflow を共有する
- 共有対象が primitive ではなく、domain-specific read/write rule を持つ
- page 間 copy ではなく、明確な ownership を持つ単一 module として読める

逆に、単に file 数が増えたことだけを理由に `features/` を導入してはならない。

## Data Boundary

data fetch と state の default は page-local とする。

- page 初期表示に必要な fetch は、その page 配下で開始してよい
- polling が必要な場合も、default は page-local timer とする
- selected row、filter draft、tab state、detail refresh status は page-local state に置く
- page をまたいで参照しない data を global store へ昇格させてはならない

global session cache や cross-route query cache は deferred decision とする。
少なくとも次のいずれかが起きるまで、global cache 導入を default にしてはならない。

- same-origin session / viewer identity のように、複数 route が同じ freshness requirement を持つ
- 同じ fetch 結果を複数 route が同時に消費し、page-local fetch では重複が支配的になる
- page 遷移をまたぐ optimistic update または invalidation rule が product contract になる

## API Boundary

frontend が使う HTTP/API contract は、external/public API と同一である必要はない。

- operator/admin や browser-specific UX のための frontend API は、external/public API と別 route family として持ってよい
- public/external contract は backward compatibility と consumer readability を優先する
- frontend API は browser UX、polling cadence、detail aggregation、page action を優先して整形してよい
- backend 内部ロジック、storage access、domain rule は共有してよい
- transport layer と response shaping は分け、frontend 都合を public API contract へ直接漏らしてはならない

GraphQL はこの時点の既定にしない。
必要な read/write は HTTP route または同等の explicit contract で分けてよい。

## Browser Auth Boundary

browser 向け auth の第一選択は same-origin + http-only cookie とする。

- browser storage に bearer token を保持する前提を default にしない
- frontend route は same-origin deployment を第一選択として組み立ててよい
- session propagation は cookie-based request context を前提にしてよい
- auth failure handling や refresh rule の detailed flow は後続で固定する

この spec が今固定するのは boundary だけであり、
session rotation、CSRF 対策詳細、role-specific guard placement は別途 decision とする。

## Current Application

Phase 6 first landing の `operator-ui` は、
broader frontend architecture の最初の `operator` page family として位置づける。

- current minimal operator surface は `operator` route 配下の 1 page として扱ってよい
- first refactor では `operator` page を app shell 配下へ再配置し、
  current `/` entry から到達できる observation surface を維持してよい
- active/completed/detail/preset queue の state と polling は page-local default に従う
- `shared/ui` へ逃がしてよいのは primitive だけであり、
  operator 固有 panel や artifact entry は route/page 配下に残す
- 後続 refactor は current file size ではなく、この route-first boundary に従って進める

## Deferred Decisions

この段階では次を固定しない。

- viewer route の具体技術:
  live push、polling、event stream、canvas、DOM rendering の選択
- auth mechanism の詳細:
  login UI、session refresh、logout、role matrix、CSRF detail
- global cache / data library:
  query cache library、state library、invalidations の標準化
- app shell detail:
  deep-linking、breadcrumbs、global navigation、router library choice の最終形
- public/external API の schema と versioning detail

これらは、必要になった page family の execution plan で追加 decision として扱う。
