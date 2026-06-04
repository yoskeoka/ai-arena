# platform-online-foundation-03-05-operator-ui-verification-03-real-local-browser-operator-lane
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0068` の local browser lane と `0069` の CI browser lane 群を補完し、
本物の local `arena-service` と `operator-ui` を起動したうえで、
AI Agent が browser automation を使って operator flow の確認・調査・証跡取得を行える
real local browser operator lane を整える。

最初のゴールは、preset queue action から completed detail 表示までの実運用寄り flow を
local で再現し、実装者が screenshot を保存して PR review artifact に使えることに置く。

## Context

- user expectation として、この line の Playwright harness は local regression だけでなく、
  real local service/frontend を使った実装確認・調査手段も持つべきである
- `0068` の現実装は `go run ./cmd/operator-ui-fixture` を browser lane 専用 backend として起動しており、
  本物の local `arena-service` を触る lane とはまだ分かれている
- local の queue / read model confidence を上げるには Postgres-backed lane を優先したい
- artifact storage は local object storage が整備済みならそれを使い、未整備なら file-backed を使う
  明確な二段判断にしたい
- `0069` は CI 側の file-backed lane と durable backend lane を整理する計画であり、
  local 実環境を agent が触る運用 lane とは別の concern である
- operator UI の review artifact として completed state の screenshot を残せると、
  実装者が「local で本当に動いた」ことを reviewer に渡しやすい

## Scope

- local Postgres harness、real `arena-service`、real `operator-ui` を 1 導線で起動する
- artifact storage は local object storage がある場合はそれを使い、ない場合は file-backed で起動する
- Playwright CLI または同等の ordinary browser automation から既存 local UI を操作できるようにする
- queue action -> active/completed visibility -> completed detail の実 flow を local で確認できるようにする
- completed 表示の screenshot / trace / 必要最小限の log を reviewer artifact として保存できるようにする
- 実装確認用途と調査用途の runbook を repo-owned 手順として残す

この plan では以下を扱わない。

- GitHub Actions 上の screenshot capture 自動化
- production / staging deploy 環境への browser attach
- visual diff testing や snapshot golden 比較
- ranking / tournament / general submission flow までの拡張

## Spec Changes

### `docs/specs/platform-service-operator-ui.md`

- `0068` の local regression lane と real local operator lane を役割分担つきで追記する
- browser automation が screenshot capture や investigation に使う最小 observation / operation seam を補足する

### `docs/specs/platform-service-skeleton.md`

- local verification lane を `0068` の regression lane と `0070` の real local service lane に分ける前提を補足する
- `0070` は Postgres-backed queue/state を第一候補にし、artifact storage は
  local object storage 優先、未整備時 file-backed fallback とすることを明記する
- real local lane では backend/frontend/browser が同一環境で協調起動し、agent/contributor が操作・観測してよいことを明記する

## Development / Runbook Changes

- `docs/development/operator-ui-local-verification.md` を `0068` lane と `0070` lane の 2 本立てに整理する
- real local lane の起動、停止、schema apply、seed、queue 操作、completed 待機、screenshot 保存手順を記録する
- local object storage がある場合の起動手順と、未整備時に file-backed へ切り替える判断基準を記録する
- PR 証跡として screenshot path / trace path / log path をどこへ残すかを runbook に書く

## Expected Code Changes

- Postgres-backed local queue/state を起動する helper / script / target
- real `arena-service` + real `operator-ui` を local object storage または file-backed artifact lane で扱う起動 helper
- browser automation が既存起動済み local URL に attach する Playwright command / scenario
- queue -> completed detail を待って screenshot を保存する scenario
- local investigation 時に使う artifact 保存先整理
- 必要なら `0068` lane と shared selector/assertion helper の分離整理

## Sub-tasks

- [ ] real local backend/frontend を 1 導線で起動・停止する helper を定義する
- [ ] local Postgres schema/bootstrap の手順を定義する
- [ ] local artifact storage が object storage で起動できるかを確認し、未整備なら file-backed fallback を定義する
- [ ] local verification data seed の扱いを決め、real service を queue から completed まで進める最小 flow を固定する
- [ ] Playwright から既存 local UI に attach する inspection/capture command を追加する
- [ ] preset queue action から completed detail まで待機して screenshot を保存する scenario を実装する
- [ ] screenshot / trace / backend log の保存先と reviewer handoff 手順を runbook に追記する
- [ ] `0068` lane と `0070` lane の役割分担を docs/specs に明文化する

## Parallelism

- [parallel] local bootstrap helper 設計と screenshot handoff runbook の叩き台作成は並行できる
- [parallel] selector/assertion helper の shared 化と local seed flow の整理は並行できる
- real queue -> completed scenario 実装は local bootstrap helper と seed 方針に depends on する

## Dependencies

- depends on: `0068-platform-online-foundation-03-05-operator-ui-verification-01-local-agent-browser-loop.md`
- depends on: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md` と独立 concern だが、lane 役割分担を揃えるため相互に inform する
- depends on: `0065-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- depends on: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`

## Risks and Mitigations

- real local lane は `0068` より重く、queue completion 待機も含むため flaky になりやすい
  - mitigation: `0068` は高速 regression として残し、real local lane は inspection/capture 用の narrow scenario に絞る
- reviewer artifact 導線まで 1 plan に入れると PR/comment/upload まで膨らみやすい
  - mitigation: first step は screenshot/trace/log を repo-local path に保存するところまでに留め、GitHub への貼り付け運用は runbook に整理する
- local bootstrap が arena-service の本物経路に密結合すると、調査 lane の保守コストが高くなる
  - mitigation: ordinary command line で再現できる起動手順を正本にし、Playwright は attach/operate/capture だけを担う
- local object storage が未整備な環境でも lane を動かせないと、実装確認 lane が使われなくなる
  - mitigation: queue/state は Postgres-backed を第一候補に固定しつつ、
    artifact storage だけ object storage 優先 / file-backed fallback の二段判断にする

## Design Decisions

- operator UI verification は 3 lane で分ける
  - `0068`: local regression lane
  - `0070`: real local inspection/capture lane
  - `0069`: CI browser lane 群
- `0070` の real local lane は Postgres-backed queue/state を正本にし、
  artifact storage は local object storage 優先、未整備時 file-backed fallback とする
- real local lane の browser automation は ordinary Playwright CLI を正本にし、MCP/browser-interactive は optional tactic として扱う
- reviewer artifact の first target は repo-local screenshot/trace/log path とし、PR body/comment への掲載はその上位運用とする
