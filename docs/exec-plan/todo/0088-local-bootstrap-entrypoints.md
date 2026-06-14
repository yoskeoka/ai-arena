# local-bootstrap-entrypoints
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0029-local-playwright-and-worktree-dependency-bootstrap.md`

## Objective

fresh `ww create` worktree でも、repo-owned な canonical command を叩けば
`textlint` と local Playwright verification が必要最小限の self-bootstrap を自動で通過できる状態にする。

最初の到達点は、contributor / AI agent が setup runbook を先に暗記しなくても、
root の Markdown lint と `operator-ui` の local browser verify を
「entrypoint command を実行するだけ」で開始できることに置く。

## Context

- `docs/issues/0029-local-playwright-and-worktree-dependency-bootstrap.md` は、
  fresh worktree で root / `operator-ui` の `node_modules` 不在と
  local Playwright browser 不在が毎回の停止点になることを記録している
- 現在の repo-owned local verification 方針は、
  remote/staging lane は Playwright official image に寄せつつ、
  local lane は host-native command を canonical に保つ split を採っている
- `0068` / `0070` と `docs/development/operator-ui-local-verification.md` は
  ordinary Playwright CLI / helper を repo contract としており、
  hidden hook や agent-specific skill 前提には寄せていない
- user decision として、bootstrap は missing 時だけ自動実行し、
  every run で無条件 install する形は採らない

## Scope

- root `textlint` entrypoint を wrapper 化し、missing dependency を command 実行直前に self-bootstrap する
- `operator-ui` の local Playwright entrypoint を wrapper 化し、
  missing dependency と missing local browser を verify 実行直前に self-bootstrap する
- README / runbook / spec は setup 手順の列挙より canonical command を正本として短く保つ
- local bootstrap の発火条件、非発火条件、失敗時の案内を repo-owned contract として固定する

この plan では以下を扱わない。

- `ww create` hook や shell startup hook の導入
- remote/staging Playwright Docker lane の bootstrap 方針変更
- root と `operator-ui` 以外の package manager bootstrap 自動化
- global tool install を前提とする host mutation の常設化

## Viable Options

### Option 1: command entrypoint wrapper で missing 時だけ self-bootstrap する

- `pnpm run textlint:file -- ...` と local verify command が最初の実行時だけ bootstrap を挟む
- fresh worktree 問題を repo-owned command 側で吸収できる
- 通常時は no-op に近く、既存 contributor flow を重くしにくい

### Option 2: canonical docs/runbook へ毎回誘導し、manual setup を残す

- 実装は軽い
- ただし issue の本体である repeated bootstrap stop を解消しない

### Option 3: `ww create` hook や shared workspace setup に寄せる

- worktree 生成直後に dependencies を展開できる
- ただし repo 外の workflow 依存が強く、ai-arena 単体の repo-owned contract としては drift に弱い

## Recommended Decision

Option 1 を採る。

- `docs/project-plan.md` の AI-first 方針と、既存 verification plan 群の
  canonical-command-first 方針に最も整合する
- local lane は host-native のまま保ちつつ、fresh worktree 停止点だけを減らせる
- Option 2 は issue を実質的に残し、Option 3 は repo contract を `ww` 実装へ漏らしすぎる

## Spec Changes

### New: `docs/specs/contributor-bootstrap-entrypoints.md`

- repo-owned contributor command のうち self-bootstrap を許可する entrypoint を定義する
- bootstrap は command 実行直前にだけ発火し、missing condition でのみ走る contract を明記する
- root dependency bootstrap、`operator-ui` dependency bootstrap、
  local Playwright browser bootstrap の責務境界を定義する
- hidden startup hook を使わず、observable command path に閉じる decision を記録する

### Update: `docs/specs/platform-service-operator-ui.md`

- local browser verification contract に、
  canonical command が missing local dependency / browser を前段 bootstrap してよいことを追記する
- local lane は host-native command を正本とし、remote Docker lane とは bootstrap strategy を分けることを再確認する

## Documentation Changes

### `README.md`

- setup 手順の列挙を縮め、canonical command を先頭に置く
- bootstrap が必要なら command 側で解決されることを短く案内する

### `docs/development/operator-ui-local-verification.md`

- `pnpm install --frozen-lockfile` や `playwright install chromium` を
  pre-step の必須列挙から外し、wrapper command が必要時だけ実行する contract に書き換える
- 失敗時の fallback と manual debugging path は補助情報として残す

### `docs/development/japanese-textlint.md`

- root textlint command も self-bootstrap entrypoint になったことを反映する

### Optional: `AGENTS.md`

- 実装後も human / agent が誤りやすいなら、
  local verification / textlint の canonical command 一覧を短く追記する
- ただし docs/spec/runbook で十分に参照導線が閉じる場合は必須にしない

## Expected Code Changes

- root `textlint` wrapper script の追加または既存 `tools/run-textlint.sh` の前段 bootstrap 化
- `operator-ui` verification wrapper script の追加または package script の entrypoint 差し替え
- local Playwright browser existence check と install helper の追加
- missing 判定を行う shared shell helper の抽出、または root / `operator-ui` それぞれの小さな専用 helper
- package scripts / Makefile / runbook 参照先の更新

## Missing Detection Rules

- root dependency bootstrap:
  root `pnpm` entrypoint 実行時に、repo root の local package dependencies が未展開なら
  `pnpm install --frozen-lockfile` を実行してよい
- `operator-ui` dependency bootstrap:
  local verify entrypoint 実行時に、`operator-ui` の local package dependencies が未展開なら
  `operator-ui/` で `pnpm install --frozen-lockfile` を実行してよい
- local Playwright browser bootstrap:
  local verify entrypoint 実行時に host-native browser executable が未解決な場合だけ、
  repo-owned local browser install helper を実行してよい
- steady-state run:
  dependencies と browser が既に揃っている通常時は install を再実行してはならない

lockfile hash による再install 判定までは first step の必須 scope にしない。
最初は missing-only contract に留める。

## Sub-tasks

- [ ] self-bootstrap 対象 entrypoint を固定する
- [ ] root textlint bootstrap contract を spec / docs / script に落とす
- [ ] `operator-ui` local verify bootstrap contract を spec / docs / script に落とす
- [ ] dependency missing 判定の最小実装を決める
- [ ] Playwright browser missing 判定と install helper を決める
- [ ] package script / shell wrapper / helper の wiring を行う
- [ ] fresh worktree と warm worktree の両方で verification を定義する

## Parallelism

- [parallel] root textlint wrapper 設計と `operator-ui` wrapper 設計は並行できる
- [parallel] spec 叩き台と README/runbook の短文化は並行できる
- shared helper 抽出は、両 wrapper の責務境界が固まってから着手する

## Verification

- fresh worktree で root `textlint` command が manual install 前でも完走する
- fresh worktree で `operator-ui` local verify command が manual install 前でも完走する
- warm worktree では bootstrap step が再実行されず、通常 verify と同等の速度で通る
- remote/staging lane の Playwright Docker contract に影響しない

## Risks and Mitigations

- missing 判定が雑だと不要な reinstall が走り、通常時の開発速度を落とす
  - mitigation: first step は explicit な existence check に絞り、lockfile hash 判定は後続に回す
- Playwright browser bootstrap を local lane に入れすぎると、remote lane の container contract と混線する
  - mitigation: spec と docs で local host-native lane 専用 bootstrap であることを明記する
- wrapper が大きくなりすぎると保守しづらい
  - mitigation: root textlint と `operator-ui` verify に必要な最小分岐だけを入れ、共通化は小さく保つ
- host library 不足など browser install 以外の failure は自動解決できない
  - mitigation: runbook に fallback / manual diagnosis path を短く残す

## Design Decisions

- bootstrap は hidden hook ではなく observable command entrypoint でだけ走らせる
- bootstrap は missing 時だけ実行し、steady-state では no-op に近づける
- local Playwright verification は host-native canonical path を維持する
- `AGENTS.md` は primary contract にせず、必要なら補助導線としてだけ使う
