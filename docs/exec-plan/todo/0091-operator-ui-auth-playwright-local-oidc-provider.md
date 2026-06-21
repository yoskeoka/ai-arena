# operator-ui-auth-playwright-local-oidc-provider
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`
を、local / CI 専用の OIDC identity provider を追加することで解消する
execution plan を定義する。

この plan のゴールは、GitHub OAuth login そのものを mock するのではなく、
Playwright が simple password login で通せる local/CI 専用 identity provider を使って
`browser login -> callback -> session cookie -> protected route`
の repo-owned verification lane を持つことである。

加えて、local verification でも複数 user を切り分けて扱える
repo-owned auth seam を持ち、
`tester01` と `tester02` のような固定 account を使って
multi-user 観点の operator / participant / future role 差分テストへ広げられる土台を作る。

result として、この provider は login 手段として増える。
ただし public default で常時見せる hand ではなく、
local/CI verification のために only-visible な login hand として扱う。
将来 Google login/OIDC seam の regression lane としても再利用してよいが、
primary objective は local/CI の multi-user auth seam 確立に置く。

Addresses: `docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`

## Context

- user は「OAuth は本来 login protocol ではないので、generic local auth provider を足すなら OIDC の方が自然」
  という判断を置いている
- user は Google login 回帰に使える点を認めつつも、
  local verification 自体で複数 user を切り分ける必要があるため、
  local mock OIDC provider 自体が必要だと判断している
- current repo には `github.com/coreos/go-oidc/v3/oidc` verifier seam があり、
  OIDC provider を backend に足す土台はある
- current Playwright lane の gap は login provider 不在にあり、
  fixed tester account と fixed password で単純に通せる local/CI provider があると
  local verification と CI verification の両方で扱いやすい
- 一方で、この provider を public login hand と同格に見せると
  product boundary がぶれるため、visibility と positioning を慎重に定義する必要がある

## Option Snapshot

### Option A: local / CI 専用の OIDC provider を追加し、login page でも local/CI のときだけ表示する

- 利点:
  browser automation から通しやすく、generic provider seam として再利用できる
- 欠点:
  login 手段が 1 つ増えるため、product boundary と visibility 制御が必要になる

### Option B: OIDC provider は追加するが、public login hand としてではなく
### future OIDC provider の regression lane へ主目的を寄せる

- 利点:
  generic OIDC seam の検証価値を保ちながら product login hand の増加を抑えやすい
- 欠点:
  browser からどう入口を出すかがやや複雑になり、
  local/CI 専用 UI / route の contract を別途定義する必要がある

### Option C: local provider は入れず、GitHub login 回帰だけ別 child plan で担保する

- 利点:
  product login hand が増えない
- 欠点:
  generic local auth provider seam が得られず、
  OIDC capability の regression lane を別途作りにくい

## Recommendation

- Option A を採用する
  - local / CI only-visible な OIDC login hand を持つ
  - fixed tester account による multi-user verification を第一目的とする
  - public default では見せない
- local/CI mock OIDC provider の実装基盤は
  `https://github.com/luikyv/go-oidc` で固定する
  - repo-owned minimal provider を hand-rolled で組むのではなく、
    OIDC server library の上に fixed tester account / fixed password の
    minimal login surface を載せる
- Option B の価値は secondary benefit として残す
  - future Google login / OIDC seam の regression lane に流用してよい
  - ただし primary objective は local/CI の multi-user auth seam とする

## Scope

- local / CI 専用 OIDC provider の責務と最小機能を定義する
- fixed tester account (`tester01`, `tester02`) と fixed password の bootstrap を定義する
- MFA、passwordless、device flow、TOTP、mail-based challenge を持たない
  simple password auth lane を定義する
- local verification で複数 user を切り分ける UI / session / fixture contract を定義する
- Playwright から provider login、callback、session establishment を通す
- local / CI backend と frontend の visibility / entrypoint contract を定義する

この plan では以下を扱わない。

- production での local OIDC provider 提供
- Google login の本実装
- full-featured external IdP の導入

## Spec Changes

- `docs/specs/platform-product-auth.md`
  - OIDC-capable provider seam と local/CI-only provider contract を追記する
  - local provider visibility の条件
    (`local/CI only`、public default hidden など) を明記する
  - local/CI mock OIDC provider は `luikyv/go-oidc` ベースで構築する方針を明記する
- `docs/specs/platform-service-operator-ui.md`
  - browser verification lane に
    `local OIDC auth lane` を追加する
  - login page または local-only entrypoint の acceptance surface を定義する
  - multi-user tester account を切り替える acceptance surface を定義する
- `docs/specs/platform-service-operator-api.md`
  - auth companion route に generic OIDC login/callback をどう置くかの contract を補足する
- `docs/development/operator-ui-local-verification.md`
  - tester account、password、provider bootstrap、Playwright scenario を追記する

## Expected Code Changes

- OIDC provider bootstrap
  - local / CI 用 issuer
  - discovery
  - authorize
  - token
  - jwks
  - user login form
  - `luikyv/go-oidc` の provider/server 構成
- backend auth route / provider config
  - local OIDC provider descriptor
  - callback handling
  - ID token verification
  - normalized identity mapping
- login page / local-only entrypoint
  - local/CI のときだけ見せる provider start action
- tester fixture
  - fixed tester account seed
  - fixed password
  - multi-user role / identity variation の seed

## Sub-tasks

- [ ] local/CI-only OIDC provider の positioning を
  `local/CI only-visible login hand` として固定する
- [ ] minimal provider feature set を固定する
- [ ] `luikyv/go-oidc` 上で必要な provider/server 構成と login surface を整理する
- [ ] tester account / password bootstrap を決める
- [ ] multi-user tester switching contract を決める
- [ ] backend 側の OIDC route / verifier / claim mapping を定義する
- [ ] login page または local-only entrypoint の visibility contract を定義する
- [ ] Playwright lane に provider login / callback / session establishment を追加する

## Parallelism

- [parallel] provider minimal feature set の整理と tester bootstrap の整理は並行できる
- [parallel] spec 上の visibility contract と backend verifier seam の整理は並行できる

## Dependencies

- depends on: `docs/specs/platform-product-auth.md`
- depends on: `docs/specs/platform-service-operator-ui.md`
- depends on: `docs/issues/0030-operator-ui-playwright-auth-coverage-gap.md`
- depends on: `docs/exec-plan/done/0089-oauth-library-adoption.md`

## Verification

- local / CI で only-visible な OIDC login entrypoint がある
- Playwright が fixed tester account / fixed password で login できる
- callback 後に session cookie が成立し、protected operator route に到達できる
- OIDC provider は MFA / passwordless / device flow / TOTP を要求しない
- provider visibility が public default で漏れず、local/CI contract が明示されている

## Risks and Mitigations

- local provider が product login hand の一部に見えて境界が曖昧になる
  - mitigation:
    local/CI-only visibility と positioning を spec で固定する
- multi-user fixture が曖昧だと single-user happy path しか検証できない
  - mitigation:
    `tester01`, `tester02` の identity 差分と切り替え手順を最初から contract に含める
- generic OIDC provider を作り込みすぎると current issue より大きい auth project になる
  - mitigation:
    `luikyv/go-oidc` を使いつつも、
    fixed tester account、fixed password、minimal OIDC endpoints に限定する
- Google login regression lane と local-only login hand の目的が混ざる
  - mitigation:
    Option A / B を plan に残し、implementation で選んだ positioning を spec に明記する

## Design Decisions

- generic local auth provider を持つなら OIDC の方が自然という判断を採る
- local/CI mock OIDC provider の server library は `luikyv/go-oidc` で固定する
- provider 機能は browser automation 通過に必要な最小面へ絞る
- local verification の multi-user seam を first-class requirement として扱う
- product login hand と regression lane の位置づけを曖昧にしたまま実装を始めない
