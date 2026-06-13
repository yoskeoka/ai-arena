# staging-verify-playwright-docker
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0028-online-release-staging-verify-playwright-install-hang.md`

## Objective

`online-release-staging-verify` の Playwright browser install hang を、
remote verification job を Playwright 公式 Docker image ベースへ寄せることで解消する。
最初の到達点は、staging verify の canonical remote lane から
`pnpm exec playwright install --with-deps chromium` 依存を外し、
同じ commit SHA の verify rerun が stale install step に詰まらない状態へ戻すことに置く。

## Context

- current workflow は host runner 上で `node-version: 24` を設定し、
  `pnpm exec playwright install --with-deps chromium` を実行してから remote verify を始めている
- issue `0028` では、この install step が長時間 `in_progress` のまま止まり、
  同一 SHA の後続 verify run も concurrency group で `pending` のまま詰まる事象が観測されている
- `operator-ui` の remote verify lane は staging frontend/backend URL を直接叩く構成であり、
  local backend/frontend を同居起動する lane ではない
- current Playwright config では managed-backend lane の `video` は `off` であり、
  remote verification の required artifact は `trace + screenshot + HTML report` が中心になっている
- user decision:
  - CI remote verify は Docker 化する
  - local verification contract は canonical command を変えずに維持する
  - local で同種の hang が出た場合の fallback は docs に残す

## Scope

- `online-release-staging-verify.yml` の remote verify job を Playwright 公式 Docker image ベースへ移す
- browser install step を workflow から削除する
- image version と repo の `@playwright/test` version の整合ルールを docs/spec に残す
- local verification は host-native contract を維持したまま、Node 22 fallback を docs に追加する
- staging verification artifact contract を current remote lane 実態に合わせて明文化する

この plan では以下を扱わない。

- local verification lane 自体の Docker 化
- Playwright video capture の再有効化
- remote verify 対象 API や UI flow の追加
- production release lane の containerization

## Options Considered

### Option A: remote verify job を Playwright Docker image へ移す

- 利点:
  - CI 中の browser download / unzip / host dependency install を job 開始前に排除できる
  - remote verify lane は外部 URL を叩くだけなので、container 化しても service topology が複雑化しにくい
  - current `video=off` 方針とも整合し、`ffmpeg` の host 差異を workflow から切り離しやすい
- 欠点:
  - image tag と `@playwright/test` version の pin 管理が増える
  - local lane と CI lane の実行環境差は残る

### Option B: host runner のまま Node version / install command だけを絞って回避する

- 利点:
  - workflow 差分が小さい
  - local と CI の実行形が近いまま保てる
- 欠点:
  - browser install step 自体は残り、同系統の runner / unzip / host dependency 問題が再発しうる
  - issue `0028` の根にある「install step が verify lane を占有する」構造は残る

### Recommendation

Option A を採る。`0074` で固定した release contract は staging verify を acceptance gate として扱うため、
この lane は最小限の環境差で安定に回ることを優先する。
local lane は `docs/development/operator-ui-local-verification.md` の repo-owned command を正本として残し、
CI だけを Playwright 公式 image へ寄せる。

## Spec Changes

### `docs/development/platform-service-online-deploy.md`

- `online-release-staging-verify` の remote verify job が Playwright 公式 Docker image を使うことを追記する
- image tag は repo の `operator-ui/package.json` にある `@playwright/test` version と揃える rule を明記する
- browser install step を workflow 内で実行しないことを明記する
- verify artifact は current remote lane では `Playwright trace zip`, `screenshot`, `HTML report` を正本とする

### `docs/development/operator-ui-local-verification.md`

- local canonical lane は host-native のまま維持することを追記する
- local で browser install hang を踏んだ場合の fallback として、
  `Node 22 LTS` で `pnpm exec playwright install chromium` を再試行する案内を追加する
- optional note として、local Docker 実行は debugging fallback であり canonical lane ではないことを明記する

### `docs/specs/platform-service-operator-ui.md`

- staging verification の CI artifact contract を current remote lane 実態に合わせて
  `trace + screenshot + HTML report` 中心で明記する
- CI lane では browser runtime を repo 外の pinned Playwright image に載せてもよいことを補足する

## Expected Code Changes

- `.github/workflows/online-release-staging-verify.yml`
  - verify job に Playwright 公式 container 設定を追加する
  - browser install step を削除する
  - container 上で必要な checkout / pnpm install / remote verify / artifact upload が継続することを確認する
- `operator-ui/package.json`
  - 必要なら plan 実装時に Playwright version と container tag の対応が追跡しやすい形へ補足する

## Sub-tasks

- [ ] current remote verify workflow の host-runner assumptions を洗い出す
- [ ] Playwright 公式 Docker image tag と current `@playwright/test` version の対応を決める
- [ ] `online-release-staging-verify.yml` を container ベースへ移し、browser install step を削除する
- [ ] artifact upload と workflow summary が container 実行後も維持されることを確認する
- [ ] `platform-service-online-deploy.md` に CI container contract を追記する
- [ ] `operator-ui-local-verification.md` に local fallback を追記する
- [ ] `platform-service-operator-ui.md` に remote verify artifact contract の明文化を追加する

## Parallelism

- [parallel] doc/spec wording の叩き台作成
- [depends on: Playwright image tag decision] workflow container 化
- [depends on: workflow container 化] artifact path と summary の整合確認

## Dependencies

- depends on: `0074-platform-online-foundation-03-04-matchmaking-ranking-follow-up-01-phase6-release-flow.md`
- depends on: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`

## Risks and Mitigations

- container tag と repo の `@playwright/test` version がずれて browser executable 解決に失敗する
  - mitigation: image tag pin rule を docs に残し、実装時に version を明示的に揃える
- local lane まで Docker 前提だと contributor / agent の real-local verification が重くなる
  - mitigation: local contract は維持し、Docker は remote CI lane に限定する
- current artifact contract と spec wording がずれて reviewer が期待する evidence を誤解する
  - mitigation: remote verify artifact を `trace + screenshot + HTML report` で揃えて docs/spec を更新する

## Design Decisions

- remote staging verify は host-runner 上の browser install ではなく、Playwright 公式 Docker image を canonical runtime とする
- local verification は existing host-native commands を canonical runtime のまま維持する
- video/ffmpeg の扱いはこの issue では再拡張せず、artifact contract は current remote lane の実態を正本にする

## Verification

- workflow definition 上で `Install Playwright browser` step が消え、container 化された verify job から
  `pnpm run verify:remote` と artifact upload が追跡できること
- docs/spec 上で CI remote lane と local lane の責務差が読めること
- local fallback が「canonical lane を変えない」方針のまま追加されていること
