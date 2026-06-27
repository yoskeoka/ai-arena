# operator-ui-browser-playwright-hang-runtime-alignment
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0036-operator-ui-browser-runtime-alignment.md`

## Objective

`operator-ui` verification における local / CI / Postgres / SeaweedFS / Playwright browser の
runtime/bootstrap 責務を再整理し、将来 `Plan C` 相当の
dedicated browser CI Docker/runtime 再設計を無理なく実施できる土台を作る。

この plan は current `operator-ui-browser` hang の immediate fix ではない。
最初の到達点は、lane ごとの runtime ownership を揃え、
「何を workflow が用意し、何を helper が起動し、何を external service とみなすか」を
明文化して実装へ落とせる状態にすることにある。

## Context

- `0095-operator-ui-browser-playwright-hang.md` は
  current CI failure の stop-the-bleeding を優先し、helper/bootstrap regression の解消に集中する
- 現状は lane ごとに runtime responsibility が揺れている
  - Playwright browser bootstrap は helper が自動実行しうる
  - Postgres は workflow service / local compose / helper-side reset が混在する
  - SeaweedFS は一部 lane で helper/script 内起動を期待している
  - staging verify remote lane だけは Playwright Docker runtime を採用済み
- この asymmetry は current hang の直接原因でなくても、
  CI / local / remote の runtime contract を複雑にしている

## Scope

- local canonical lane、dedicated CI browser lane、remote staging verify lane の
  runtime/bootstrap responsibility を棚卸しする
- Postgres / SeaweedFS / Playwright browser ごとに、
  helper responsibility と workflow/service responsibility を再定義する
- `operator-ui-browser-file-backed` と `operator-ui-browser-postgres` を
  Docker runtime へ寄せる場合の service topology を設計する
- docs/spec/helper/workflow の責務境界を揃える

この plan では以下を扱わない。

- current `operator-ui-browser` hang の immediate stop-the-bleeding
- operator UI feature scope の拡張
- remote staging verify flow の product requirement 変更

## Options Considered

### Option A: current asymmetry を維持し、各 lane の helper を個別補修する

- 利点:
  - 変更範囲が小さい
  - 既存 command surface を保ちやすい
- 欠点:
  - local / CI / remote の runtime contract が読みづらいまま残る
  - Docker/runtime 再設計のたびに lane ごとの例外が増えやすい

### Option B: runtime ownership を service-first に再整理する

- 利点:
  - Postgres / SeaweedFS / browser を
    「workflow/service が供給するもの」と
    「helper が補助するもの」に分けやすい
  - `operator-ui-browser` の Docker 化可否を lane ごとに説明しやすい
- 欠点:
  - helper と workflow の両方に手が入る
  - local canonical lane の ergonomics を損なわない設計が必要

## Recommendation

Option B を採る。

local canonical lane の使いやすさは維持しつつ、
CI / remote 向け runtime は workflow/service-first に寄せる。
これにより、Playwright browser・Postgres・SeaweedFS の責務を
人間にも agent にも説明しやすい contract にできる。

## Spec Changes

### `docs/specs/platform-service-operator-ui.md`

- local / CI / remote lane ごとの runtime ownership を追記する
- browser provisioning、Postgres、SeaweedFS の responsibility boundary を固定する

### `docs/development/operator-ui-local-verification.md`

- local canonical lane が依存してよい self-bootstrap responsibility を明確にする
- local では helper が補助してよいが、
  CI/runtime contract と混線させない方針を追記する

### `docs/development/go-quality-gates.md`

- dedicated browser CI lane の runtime preparation responsibility を明文化する

## Expected Code Changes

- `.github/workflows/operator-ui-browser.yml`
  - Postgres / SeaweedFS / browser runtime の responsibility を明示する
  - 必要なら service-based topology へ寄せる
- related workflow/helper scripts
  - `tools/dev/run-operator-ui-playwright.sh`
  - `tools/dev/ensure-operator-ui-playwright-browser.sh`
  - `tools/dev/operator-ui-backend.sh`
  - any SeaweedFS/Postgres bootstrap helper touched by the ownership split
- `operator-ui/package.json`
  - lane entrypoint は維持しつつ、runtime responsibility の前提を整理する

## Sub-tasks

- [ ] local / CI / remote lane の runtime ownership を棚卸しする
- [ ] Postgres / SeaweedFS / browser provisioning の責務境界を決める
- [ ] `file-backed` と `postgres` lane の service topology 差を整理する
- [ ] docs/spec に ownership contract を記述する
- [ ] workflow/helper 実装を contract に揃える

## Parallelism

- [parallel] docs/spec の ownership 叩き台と現状棚卸しは並行できる
- [parallel] Postgres / SeaweedFS topology 整理と browser provisioning 境界整理は並行できる
- 最終実装は ownership decision に depends on する

## Dependencies

- depends on: `0095-operator-ui-browser-playwright-hang.md`
- depends on: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`
- context from: `0084-staging-verify-playwright-docker.md`

## Risks and Mitigations

- local ergonomics を壊すと canonical lane が使いにくくなる
  - mitigation: local command surface は維持し、ownership だけ整理する
- service-first に寄せすぎると local contributor setup が重くなる
  - mitigation: local helper bootstrap は残しつつ、CI/runtime contract と分離する
- topology 再設計を先にやりすぎると immediate fix と競合する
  - mitigation: `0095` を先に片付けてから着手する

## Design Decisions

- current failure stop-the-bleeding と broader runtime/topology 再整理は別計画に分ける
- Postgres と SeaweedFS の扱いの不揃いは incidental detail ではなく、
  future Docker/runtime redesign の主要論点として扱う
- local / CI / remote で runtime ownership を説明可能な contract にする

## Verification

- plan review では次が読めること
  - local / CI / remote lane の ownership 差が整理されている
  - Postgres と SeaweedFS の扱いの不揃いが future redesign の対象として明示されている
  - `0095` と scope が分離されている
