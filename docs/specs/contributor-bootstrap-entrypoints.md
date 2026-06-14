# Contributor bootstrap entrypoints 仕様

## 目的

この spec は、fresh worktree でも repo-owned な canonical command だけで
local quality gate と local browser verification を開始できる最小 bootstrap contract を定義する。

対象 entrypoint は次の 2 系統に限る。

- root Japanese textlint command
- `operator-ui` local Playwright verification command

この spec は hidden startup hook や `ww create` hook を定義しない。
bootstrap は observable な command entrypoint でだけ発火しなければならない。

## この spec の責務範囲

この spec が定義するもの:

- self-bootstrap 対象 entrypoint
- dependency / browser の missing 判定
- missing 時だけ install を許可する contract
- steady-state で再install してはならない rule
- failure 時の user-facing message boundary

この spec が定義しないもの:

- remote/staging Playwright Docker lane の bootstrap
- root と `operator-ui` 以外の package manager bootstrap
- global tool install や shell startup hook
- lockfile hash 差分による reinstall policy

## Canonical entrypoints

repo-owned な canonical command は次とする。

- root textlint:
  `pnpm run textlint`
- focused textlint:
  `pnpm run textlint:file -- <target.md>...`
- fixture local browser verify:
  `cd operator-ui && pnpm run verify:local`
- real local browser verify:
  `cd operator-ui && pnpm run verify:local:real`

contributor と AI agent は、manual setup 手順ではなくこの command 群を first entry として扱ってよい。

## Root dependency bootstrap

root textlint entrypoint は、repo root の local package dependencies が未展開なときだけ
`pnpm install --frozen-lockfile` を実行してよい。

first step の missing 判定は explicit な existence check に留める。
少なくとも `node_modules/.pnpm` が不在なら missing とみなしてよい。

dependencies が既に展開済みなら、textlint entrypoint は install を再実行してはならない。

## Operator UI dependency bootstrap

`operator-ui` local verify entrypoint は、`operator-ui/` の local package dependencies が未展開なときだけ
`operator-ui/` で `pnpm install --frozen-lockfile` を実行してよい。

first step の missing 判定は explicit な existence check に留める。
少なくとも `operator-ui/node_modules/.pnpm` が不在なら missing とみなしてよい。

dependencies が既に展開済みなら、verify entrypoint は install を再実行してはならない。

## Local Playwright browser bootstrap

`operator-ui` local verify entrypoint は、host-native Playwright browser executable が未解決なときだけ
repo-owned browser install helper を実行してよい。

browser install helper は `operator-ui/package.json` に pin された Playwright version に追従しなければならない。
first step の browser missing 判定は、local package runtime から解決した Chromium executable path の存在確認で足りる。

browser executable が既に存在する通常時は install を再実行してはならない。

remote/staging lane の Playwright Docker contract は、この local bootstrap helper の対象外とする。

## Failure contract

self-bootstrap で自動解決できない failure は、wrapper が短い補助メッセージを出して失敗してよい。

少なくとも次は自動解決対象外としてよい。

- host library 不足
- network / registry 到達不能
- package manager 自体が未導入
- browser install 以外の Playwright launch failure

message は manual debugging path と canonical runbook への参照を含めてよい。

## Steady-state contract

bootstrap は missing 時だけ走る。

- warm worktree では root textlint command が dependency install を再実行してはならない
- warm worktree では `operator-ui` verify command が dependency install を再実行してはならない
- warm worktree では `operator-ui` verify command が browser install を再実行してはならない

この spec は first landing では existence check のみを要求し、
lockfile hash を使った再install 判定までは要求しない。
