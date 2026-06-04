# platform-online-foundation-03-05-operator-ui-verification-01-local-agent-browser-loop
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`operator-ui/` の local verification を human manual check 依存から外し、
AI Agent が local backend/frontend を起動して browser automation で自己検証できる
最小 lane を整える。
最初のゴールは、preset queue / active/completed visibility / completed detail /
artifact access entry の回帰を、repo に置いた再利用可能な Playwright-based harness で
確認できる状態にすることに置く。

## Context

- `docs/issues/0025-operator-ui-local-and-ci-verification.md` は、
  local AI-agent verification と CI verification が未整備なことを問題にしている
- `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md` で
  minimal operator UI surface 自体は入ったが、AI Agent が画面を自律確認する seam はまだ薄い
- user suggestion として、Codex の browser-interactive 系 workflow は local 自己検証に有効そうである
- ただし repo contract を特定 agent 実装や実験的 skill 名に固定すると drift に弱いので、
  repo 側は ordinary Playwright harness を canonical とし、agent 側はそれを実行・補助できる形に留める

## Scope

- `operator-ui` の local browser verification foundation を定義する
- UI の stable observation seam を追加し、browser automation から判定しやすくする
- backend/frontend 起動と browser verification をまとめた local runbook を整える
- AI Agent が subagent や browser session を使って再現可能に確認できる最小 coverage を入れる

この plan では以下を扱わない。

- GitHub Actions 上の durable browser verification lane
- Postgres-backed CI orchestration
- operator UI の責務分割 refactor 自体
- `playwright-interactive` skill 自体の repo 導入や contributor 前提化

## Spec Changes

### `docs/specs/platform-service-operator-ui.md`

- local browser verification で観測する最小 acceptance surface を追記する
- preset queue panel、active/completed list、detail panel、artifact access entry の
  stable observation rule を明記する
- browser automation が依存してよい role / text / machine-readable hook の扱いを明文化する

### `docs/specs/platform-service-skeleton.md`

- operator UI verification lane を default file-backed lane と durable CI lane に分ける前提を補足する
- local lane では backend/frontend/browser verification を contributor/agent が同一環境で起動してよいことを追記する

## Development / Runbook Changes

- `README.md` または `docs/development/*` に local operator UI verification runbook を追加する
- browser automation の起動方法、backend/frontend の前提、失敗時の観測 artifact を記録する
- OpenAI `playwright-interactive` skill は optional tactic として脚注してよいが、
  repo canonical 手順としては扱わない

## Expected Code Changes

- `operator-ui` への Playwright-based test harness 導入
- browser automation が安定して読むための UI observation seam 追加
- local backend/frontend/browser をまとめて起動する helper / script / target
- first coverage 用の local verification scenario
- run artifact / screenshot / trace の保存先整理

## Sub-tasks

- [ ] Playwright 系 harness 前提で、repo canonical な browser verification command / helper を固定する
- [ ] operator UI の QA inventory を作り、first coverage を明文化する
- [ ] UI observation seam を追加し、selector strategy を固定する
- [ ] local backend/frontend/browser verification の起動 helper を整える
- [ ] health backend 起動確認、active/completed panel、preset queue action、
      completed detail、artifact access entry を含む最小 scenario を実装する
- [ ] AI Agent 向け local self-verification runbook を整える

## Parallelism

- [parallel] QA inventory 整理と local runbook の叩き台作成は並行できる
- [parallel] UI observation seam 追加と backend seed/helper 整理は並行できる
- browser scenario 実装は selector strategy と local bootstrap helper に depends on する

## Dependencies

- depends on: `0065-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- depends on: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`
- informs: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`
- informs: `docs/issues/0026-operator-ui-component-state-refactor.md`

## Risks and Mitigations

- Codex 固有の interactive skill に寄せすぎると、agent/runtime drift で repo の verification が壊れやすい
  - mitigation: repo contract は Playwright harness を正本とし、interactive browser session はその上位利用に留める
- `playwright-interactive` は `js_repl` と `danger-full-access` 前提が強く、repo contributor contract にしにくい
  - mitigation: skill の導入有無と無関係に動く Playwright command / helper を canonical path にする
- selector が styling や文言に密結合すると、見た目変更だけで tests が壊れやすい
  - mitigation: role/text 優先 + 必要最小限の machine-readable hook を spec で固定する
- local bootstrap が重すぎると AI Agent が再利用しなくなる
  - mitigation: first lane は最小 fixture と最小 scenario に絞り、durable DB 前提は次 plan に分離する

## Design Decisions

- browser automation の canonical foundation は Playwright 系に一本化する
- repo に残す contract は ordinary Playwright 実行で再現できるものとし、
  agent-specific interactive tooling は optional execution tactic として扱う
- OpenAI `playwright-interactive` skill は採用必須にしない。使う場合も local 自己検証の補助手段に限る
- `0026` の UI component/state refactor は、この local verification seam ができてから着手する
