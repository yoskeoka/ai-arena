# operator-ui-browser-playwright-hang
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0035-operator-ui-browser-playwright-hang.md`

## Objective

`operator-ui-browser` workflow の Playwright hang について、
`PR276` で導入した repo-owned helper が direct cause か、
あるいは dedicated CI browser lane 自体を Docker runtime へ寄せるべきかを
切り分けたうえで、browser CI を安定に継続実行できる状態へ戻す。

最初の到達点は次の 2 つである。

- `operator-ui-browser-file-backed` / `operator-ui-browser-postgres` の両 lane で、
  現在どこで hang しているかを CI log 上で因果追跡できるようにする
- remediation を lane ごとに固定する
  - helper/bootstrap 修正で十分なのか
  - `file-backed` だけ Docker 化するのか
  - `postgres` も Docker 化するなら service topology をどう変えるのか

## Context

- `PR255` は `online-release-staging-verify` の host-runner browser install hang を避けるため、
  remote staging verify job を Playwright 公式 Docker image へ寄せた
- その fix は remote URL を叩くだけの lane だったため成立したが、
  `operator-ui-browser.yml` は local backend/frontend/bootstrap を伴う dedicated CI lane であり、
  topology が異なる
- `PR276` は token-heavy verification surface の改善として
  `verify:ci:file-backed` / `verify:ci:postgres` を
  `tools/dev/run-operator-ui-playwright.sh` 経由へ寄せた
- run `28283819740` の log では、両 lane とも
  Playwright helper が Chromium download progress 100% を出した後に停止し、
  test 本体や backend/frontend startup log に進んでいない
- `operator-ui-browser-postgres` lane は
  Postgres service container と `make seaweed-up` を使うため、
  `online-release-staging-verify` と同形の job container 化は追加設計なしには難しい

## Scope

- `operator-ui-browser` dedicated CI browser lane の current hang root cause を調査する
- `PR276` helper 導入前後で CI lane の browser provisioning contract が
  どう変わったかを整理する
- dedicated CI browser lane 向けに、
  opaque delegated browser bootstrap を避ける runtime contract を定める
- 必要なら `operator-ui-browser-file-backed` / `operator-ui-browser-postgres` を
  別 remediation に分ける
- docs/spec に
  `local canonical lane` と `dedicated CI browser lane` の runtime 役割差を反映する

この plan では以下を扱わない。

- local canonical lane の全面 Docker 化
- remote staging verify lane の再変更
- operator UI の新規 feature 拡張
- broader release workflow / deploy gate の再設計

## Options Considered

### Option A: host-runner browser CI を維持しつつ、CI では delegated browser bootstrap を使わない

- 利点:
  - `0069` で作った CI lane topology を大きく崩さずに済む
  - `postgres` lane の Postgres service / SeaweedFS harness をそのまま活かしやすい
  - direct cause が `PR276` helper なら最小修正で戻せる
- 欠点:
  - runner 上の browser provisioning を workflow step と helper で二重管理しないよう整理が必要
  - browser install 自体の flake を別の形で抱える可能性は残る

### Option B: `file-backed` lane だけ Playwright Docker 化し、`postgres` lane は host-runner で修正する

- 利点:
  - `PR255` の成功パターンを、topology が単純な lane にだけ再利用できる
  - `postgres` lane の service / SeaweedFS 制約を無理に container へ押し込まなくてよい
  - lane ごとの failure domain を分けやすい
- 欠点:
  - dedicated browser CI が 2 種類の runtime contract を持つ
  - docs/spec で lane 差を明確にしないと保守が混乱する

### Option C: `operator-ui-browser.yml` 全体を Playwright Docker runtime へ再設計する

- 利点:
  - browser runtime のばらつきを最も減らせる
  - install step を workflow から外しやすい
- 欠点:
  - `postgres` lane では Postgres service、SeaweedFS、Go/Node bootstrap、
    artifact upload まで含めた service topology を再設計する必要がある
  - 今回の direct cause が helper regression だけなら過剰変更になりうる

## Recommendation

Option A を first path とし、
実装中に `file-backed` lane だけは Option B へ切り替え可能な plan にする。

理由:

- 現在の job log は `playwright test` 本体ではなく helper 内 bootstrap で止まっており、
  `PR276` 由来の regression をまず外すのが最短である
- `postgres` lane は `make seaweed-up` を含むため、
  staging verify と同じ job container 化を即採用すると scope が不必要に広がる
- 一方で `file-backed` lane は topology が軽いので、
  helper fix だけで安定しない場合の Docker fallback 候補として plan に残す価値がある

## Spec Changes

### `docs/specs/platform-service-operator-ui.md`

- dedicated CI browser lane は、
  local canonical lane と同じ helper surface を共有してもよいが、
  browser provisioning だけは CI-observable contract として明示しなければならないことを追記する
- CI lane では browser runtime を
  workflow-managed install step か pinned Playwright container のいずれかで供給し、
  long-running delegated bootstrap を opaque helper 内へ隠さない方針を補足する
- lane ごとに runtime が異なる場合でも、
  acceptance surface と artifact contract は共通に保つことを明記する

### `docs/development/operator-ui-local-verification.md`

- local canonical lane の browser bootstrap helper は
  contributor / AI agent 向け contract として維持してよい一方、
  dedicated CI browser lane では別 runtime contract を取りうることを追記する
- CI diagnosis の正本は workflow log と artifact であり、
  helper 内で無音の長時間処理を抱え込まないことを補足する

### `docs/development/go-quality-gates.md`

- dedicated browser CI lane の browser provisioning は
  local quiet wrapper 契約とは別に、workflow 上で progress / timeout / failure point が追える必要があることを補足する

## Expected Code Changes

- `.github/workflows/operator-ui-browser.yml`
  - browser provisioning を helper 任せにせず、
    workflow-managed step か lane 別 container runtime へ整理する
  - 必要なら hang point を切り分ける timing / process / version logging を追加する
  - `file-backed` と `postgres` で runtime contract を分けるなら、その差を job 定義へ明示する
- `tools/dev/run-operator-ui-playwright.sh`
  - CI lane で local helper と同じ browser bootstrap を必ず走らせる前提を見直す
  - CI では delegated bootstrap を skip できるか、
    lane 別 contract を明示できるよう整理する
- `tools/dev/ensure-operator-ui-playwright-browser.sh`
  - browser presence check と install fallback の責務を見直し、
    CI diagnosis に必要な progress / timeout / failure point を出せるようにする
- `operator-ui/package.json`
  - `verify:ci:*` script が local-oriented bootstrap helper contract に
    不要に巻き込まれないよう必要最小限の調整を行う
- 必要なら related helper / docs path
  - `tools/dev/operator-ui-backend.sh`
  - `tools/dev/ensure-pnpm-install.sh`

## Sub-tasks

- [ ] run `28283819740` と `PR276` 差分から、
      helper/bootstrap が direct cause かを再構成する
- [ ] browser provisioning の current contract を
      local lane と CI lane に分けて棚卸しする
- [ ] [parallel] CI log で hang point が見えるように
      workflow / helper instrumentation 案を決める
- [ ] [parallel] `file-backed` lane と `postgres` lane の
      Docker 化難度と topology 制約を整理する
- [ ] [depends on: contract inventory] remediation 方針を lane ごとに固定する
- [ ] [depends on: remediation decision] docs/spec と workflow/helper 実装を更新する
- [ ] [depends on: implementation] dedicated CI browser lane の relevant verification と
      workflow-lint を通す

## Parallelism

- [parallel] log evidence の整理と lane topology 制約の棚卸しは並行できる
- [parallel] docs/spec wording 叩き台と workflow instrumentation 案の整理は並行できる
- remediation の最終選択は evidence と topology 整理の両方に depends on する

## Dependencies

- depends on: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`
- depends on: `0084-staging-verify-playwright-docker.md`
- context from: `0094-reduce-token-heavy-verification-command-surfaces.md`

## Risks and Mitigations

- root cause を見誤って先に全面 Docker 化すると、`postgres` lane の topology 変更が過剰になる
  - mitigation: helper regression と lane topology を分けて判断する
- CI lane と local lane の browser provisioning contract が曖昧だと、
  将来また helper 側へ opaque bootstrap が戻りやすい
  - mitigation: docs/spec で local canonical lane と dedicated CI browser lane の差を固定する
- `file-backed` と `postgres` の remediation を分けると docs が複雑になる
  - mitigation: runtime だけ差分として扱い、acceptance surface と artifact contract は共通化する

## Design Decisions

- dedicated CI browser lane は、
  local canonical lane の quiet wrapper contract をそのまま流用してもよいが、
  browser provisioning は CI 上で観測可能な責務として分離する
- `operator-ui-browser-postgres` lane は、
  `SeaweedFS` と service bootstrap の制約を無視して
  staging verify と同形の Docker 化へ飛ばない
- remediation は lane ごとに分けてよく、
  最小変更で戻せるなら helper/bootstrap 修正を優先する

## Verification

- plan review では次の evidence が読めること
  - run `28283819740` で helper bootstrap が最後の出力であること
  - `PR276` で `verify:ci:*` が helper 経由へ変わったこと
  - `operator-ui-browser-postgres` lane が
    Postgres service と `make seaweed-up` を持つこと
- 実装時 verification では少なくとも次を通す想定とする
  - `./tools/workflow-lint.sh --mode=pre-push`
  - relevant local verification for changed helper/workflow surface
  - updated `operator-ui-browser` workflow run or equivalent targeted verification
