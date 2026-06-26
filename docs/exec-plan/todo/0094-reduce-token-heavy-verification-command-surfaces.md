# reduce-token-heavy-verification-command-surfaces
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の local / CI verification 導線を、
「短い repo-owned command surface」と「成功時は quiet、失敗時だけ詳細」に寄せる。

対象は `Makefile`、`operator-ui/package.json`、`tools/dev/*`、
`docs/development/*` の repo-owned verification entrypoints であり、
shared workflow 側の `gh-pr-followup poll` tier 分離は別 plan / 別 PR で扱う。

Addresses: `docs/issues/0034-reduce-token-heavy-verification-command-surfaces.md`

## Current State

- `docs/issues/0034-reduce-token-heavy-verification-command-surfaces.md` が、
  token 消費の主因を次の 2 つに整理している。
  - success 時も verbose な verification output
  - 実行入口に長い env var 列が露出した command surface
- `operator-ui/package.json` の
  `verify:local:auth` / `verify:local:real` / `verify:remote` は、
  scenario 切替と artifact path 指定を長い env var 列で直接表現している。
- `tools/dev/run-operator-ui-playwright.sh`、
  `tools/dev/operator-ui-backend.sh`、
  `tools/dev/operator-ui-frontend.sh`、
  `tools/dev/github-oauth-test-double.sh` は、
  既に mode 吸収の責務を持っているが、
  command surface 側から見ると lane 名より env var 実装詳細が前面に出ている。
- `Makefile` の `lint` / `test` / browser verification 関連入口は、
  成功時 summary より各 tool の通常出力がそのまま表に出やすい。
- `docs/development/operator-ui-local-verification.md` と
  `docs/development/go-quality-gates.md` も、
  現状 command 名は揃っている一方、
  short command を正本として押し出す整理がまだ弱い。

## Spec Changes

### `docs/development/operator-ui-local-verification.md`

- local fixture lane、real local lane、auth lane の canonical command を、
  実装詳細の env var 列ではなく short repo-owned command 名で記述する。
- success 時 artifact 保存の既定、verbose/debug opt-in の入口、
  failure 時にどの log/artifact path を見ればよいかを明文化する。

### `docs/development/go-quality-gates.md`

- `make test` / `make lint` の default UX を、
  「成功時 summary、failure 時 diagnosis」という contract で補足する。
- quiet default と explicit verbose opt-in の責務を明文化する。

### `docs/specs/platform-service-operator-ui.md`

- operator UI browser verification lane が依存してよい canonical entrypoint を、
  short command 名ベースで補足する。
- lane ごとの artifact 保存責務は残しつつ、
  operator/human/AI agent が長い env var 列を覚えなくてよい前提を明記する。

## Expected Code Changes

### `Makefile`

- よく使う local verification lane へ short target を追加または整理する。
  例:
  - `make verify-auth-local`
  - `make verify-operator-ui-local`
  - `make verify-operator-ui-real`
- `test` / `lint` / browser lane wrapper の成功時出力を summary 寄りへ寄せる。
- verbose/diagnostic 出力が必要なときの opt-in 入口
  (`VERBOSE=1` や同等の repo-owned switch) を整理する。

### `operator-ui/package.json`

- long env var 列を直接露出する script を、
  repo-owned lane 名中心の script surface へ寄せる。
- scenario / artifact path / auth/mock distinction は、
  可能な限り shell helper か Make target 側へ閉じ込める。

### `tools/dev/run-operator-ui-playwright.sh` と関連 helper

- lane ごとの既定値、artifact dir、report dir、auth/mock/real-local 切替を
  helper 側で吸収できるよう整理する。
- success 時は quiet summary、
  failure 時は relevant log/artifact path を短く返す wrapper へ寄せる。
- `ensure-*` 系 helper の bootstrap message は、
  success 常時表示が不要なものを quiet default に寄せる。

### `tools/dev/operator-ui-backend.sh` と関連 process wrapper

- backend/frontend/mock provider wrapper の success 時常時ログを抑え、
  artifact file へ逃がす設計に寄せる。
- lane 既定値を script 側へ集約し、
  caller から mode 実装詳細が見えすぎないようにする。

## Design Decisions

Past decisions reviewed before planning:

- `docs/design-decisions/core-beliefs.md` は
  AI-first と correctness over speed を最優先にしている。
- `docs/issues/0034-reduce-token-heavy-verification-command-surfaces.md`
  自体が、
  repo-owned verification surface を短くし、
  success 時 output を抑える方針を既に整理している。
- `docs/development/operator-ui-local-verification.md` は
  fixture / real-local / auth regression を別 lane として定義済みであり、
  lane の意味は維持したまま surface だけ短くするのが自然である。

Apply the same reasoning here:

- verification lane の種類は増やさず、
  既存 lane の入口だけを短くする。
- repo-owned helper 内で既定値と mode 分岐を吸収し、
  caller が毎回 env var 実装詳細を組み立てない形を優先する。
- source-of-truth artifact path は残し、
  default observation だけ quiet/compact にする。

Viable implementation directions:

1. `package.json` scripts を増やして lane ごとの env var を隠す
2. `Makefile` を canonical entrypoint にして、
   Node/Playwright lane も Make targets から呼ぶ
3. shell helper に lane 名引数を持たせ、
   `make` / `pnpm` は薄い facade にする

Recommended option: option 3 を中心に、
人が最初に叩く surface は `make` または短い `pnpm` script に揃える。
lane ごとの設定責務を shell helper に寄せると、
`package.json` だけに env var 実装詳細を抱え込まずに済み、
将来 lane が増えても surface を保ちやすい。

ADR update は不要と見込む。
これは verification UX と docs/spec parity の改善であり、
project-plan の product architecture 判断を変えない。

## Sub-tasks

- [ ] [parallel] `docs/development/operator-ui-local-verification.md`、
      `docs/development/go-quality-gates.md`、
      必要なら `docs/specs/platform-service-operator-ui.md` の
      command-surface contract を更新する。
- [ ] [parallel] 既存 verification lane を棚卸しし、
      short command 名、quiet default、verbose opt-in の責務分担を決める。
- [ ] [depends on: lane inventory] `Makefile`、
      `operator-ui/package.json`、
      `tools/dev/run-operator-ui-playwright.sh` と関連 helper を更新する。
- [ ] [depends on: helper update] success 時 summary / failure 時 log path の
      出力 contract を verification docs と一致させる。
- [ ] [depends on: all above] relevant local verification と docs/spec parity
      check を実行し、
      `docs/issues/0034-reduce-token-heavy-verification-command-surfaces.md`
      を `done/` へ move できる状態にする。

## Parallelism

- docs/spec 側の contract 更新と lane inventory は並行に進められる。
- helper / Make / package script の実装は lane inventory に依存する。
- issue close 判断は docs と helper surface が揃った後に行う。

## Verification

- `make test`
- `make lint`
- browser verification の代表 lane を少なくとも必要範囲で確認
  - fixture local regression lane
  - auth-enabled local regression lane または real-local lane
- docs/spec parity check:
  - short command 名が docs と実装で一致する
  - verbose/quiet の既定値説明が helper behavior と一致する
- `./tools/workflow-lint.sh --mode=pre-push`

## Expected Outcome

- human と AI agent が、
  長い env var 列ではなく short repo-owned verification command を正本として使える。
- success 時の verification output は判断に必要な summary 中心になり、
  failure 時だけ log/artifact path を辿ればよい。
- shared workflow helper の改善とは独立に、
  `ai-arena` repo 内だけで token-heavy な verification surface を減らせる。
