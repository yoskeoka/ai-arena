# local-object-storage-harness-for-platform-service
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0024-local-object-storage-harness-for-platform-service.md`

## Objective

Phase 6 の local verification を deploy-shaped に寄せるため、
local の `arena-service` artifact backend を S3-compatible object storage で起動できる
repo-owned harness を整える。

最初のゴールは、`SeaweedFS` を local object storage の主導線として固定し、
`Postgres + SeaweedFS + arena-service` の組み合わせで
artifact write と delegated download URL を local で確認できることに置く。

## Context

- current local docs は Postgres harness まで持つが、artifact backend は local filesystem default のままである
- Phase 6 / 7 の first landing shape は `Cloudflare Pages + Render + Neon Postgres + Cloudflare R2` であり、
  local confirmation でも artifact lane を object storage shape へ寄せる価値がある
- user judgement として、local S3-compatible 候補は次で整理された
  - `SeaweedFS`: 第一候補
  - `LocalStack`: S3 用途としては重く、他の AWS emulator を当面使う予定も薄いので候補外
  - `MinIO`: 配布/運用形態が変わっており、repo-owned default harness には据えづらいため候補外
- local lane の acceptance bar は、AWS 完全互換そのものではなく、
  `arena-service` が object storage 向け delegated download URL を発行し、
  browser/CLI がその URL で artifact を取得できることである

## Scope

- `SeaweedFS` を local S3-compatible artifact harness の主導線として定義する
- local bootstrap で必要な起動コマンド、bucket 作成、credential、endpoint contract を固定する
- local object storage data dir は worktree/repo 内の ignored path に閉じ込める
- `arena-service` が local object storage lane で artifact write と delegated download URL 発行を行えるようにする
- seed object が必要な場合も、backend 内部 filesystem を直接触らず
  S3-compatible API 経由で投入する
- local verification / operator investigation / browser harness から再利用できる runbook を残す

この plan では以下を扱わない。

- AWS 全体の local emulator 導入
- production/staging の Cloudflare R2 credential 自動同期
- visual diff や heavy browser exploration
- object storage gateway の比較ベンチマーク

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- local verification lane の artifact backend として `SeaweedFS` を主導線にできることを追記する
- local filesystem lane と local S3-compatible lane の役割分担を明記する

### `docs/specs/platform-service-persistence.md`

- local durable verification では `Postgres` を metadata backend に使い、
  artifact backend は `SeaweedFS` を優先できることを補足する
- local object storage lane の delegated download verification は、
  provider-specific 完全互換ではなく repo contract verification を目的にすることを明記する

### `docs/specs/platform-service-read-model.md`

- local object storage lane でも `read` / delegated artifact access metadata の確認導線を持てることを補足する

## Development / Runbook Changes

- `README.md` から辿れる local object storage harness 手順を追加する
- `docs/development/platform-service-postgres.md` または同等の local verification runbook に
  `SeaweedFS` 起動・停止・bucket/bootstrap・teardown を記録する
- data dir の default を repo/worktree 内の ignored path に固定し、
  stale object を残さない reset-first 運用と必要 seed object の投入手順を記録する
- seed object の投入は `cp` や backend data dir への直書きではなく、
  S3-compatible API 経由で行うことを明記する
- `docs/development/platform-service-online-deploy.md` には、
  local harness が `Cloudflare R2` の代替ではなく deploy-shaped verification lane であることを補足する

## Expected Code Changes

- local object storage harness 用 helper / script / make target / compose asset の追加
- default data dir を `.local/seaweed/` のような repo-local ignored path に固定する local runtime layout の追加
- `SeaweedFS` endpoint / bucket / credential を `arena-service` へ渡す local env contract の整理
- artifact backend bootstrap と minimal verification command の追加
- reset -> bootstrap -> optional seed object upload を一連で行う helper の追加
  - seed object upload は S3-compatible API client/CLI を使って行う
- delegated download URL を local で検証するための narrow test or verification helper

## Sub-tasks

- [ ] `SeaweedFS` を local S3-compatible harness の第一候補として固定する docs/spec wording を整理する
- [ ] local data dir を repo/worktree 内の ignored path へ固定し、default を `.local/seaweed/` とする
- [ ] local bootstrap command を定義し、bucket / credential / endpoint の default contract を決める
- [ ] reset-first の bootstrap/teardown 方針を定義し、初回起動時に必要な seed object 投入を S3-compatible API 経由の script として用意する
- [ ] `arena-service` local lane が `Postgres + SeaweedFS` で起動する手順を整える
- [ ] artifact write と delegated download URL の minimal verification 手順を整える
- [ ] local filesystem lane と local object storage lane の役割分担を runbook と spec に反映する

## Parallelism

- [parallel] docs/spec wording 整理と `SeaweedFS` bootstrap command 叩き台作成は並行できる
- [parallel] env contract 整理と verification helper の叩き台作成は並行できる
- `arena-service` 実起動確認は bootstrap command と env contract に depends on する

## Dependencies

- depends on: `0056-platform-online-foundation-02-01-durable-store-and-write-model.md`
- depends on: `0064-platform-online-foundation-03-01-provider-bootstrap-and-remote-artifact-delivery.md`
- informs: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`
- informs: `0070-platform-online-foundation-03-05-operator-ui-verification-03-real-local-browser-operator-lane.md`

## Risks and Mitigations

- local object storage harness が重すぎると local verification で使われなくなる
  - mitigation: `SeaweedFS` の single-node / single-command 起動を正本にし、AWS 全体 emulator は導入しない
- local object storage data が worktree をまたいで残留すると、古い object が verification noise になりやすい
  - mitigation: data dir は worktree 内の ignored path に閉じ、default bootstrap は reset-first にする
- backend data dir に直接 `cp` して seed すると、実際の object storage contract を通らず検証の意味が薄れる
  - mitigation: seed object が必要な場合は必ず S3-compatible API 経由で投入する
- provider-specific 完全互換を最初から求めると scope が膨らむ
  - mitigation: first target は delegated download URL と artifact write/read の repo contract verification に絞る
- local filesystem lane と object storage lane の説明が混ざると docs が再び曖昧になる
  - mitigation: local filesystem lane は軽量 default、`SeaweedFS` lane は deploy-shaped confirmation と明記する

## Design Decisions

- local S3-compatible artifact harness の第一候補は `SeaweedFS` とする
- `LocalStack` は S3 用途としては重く、他の AWS emulator 需要も薄いため採用しない
- `MinIO` は current distribution / maintenance shape から repo-owned default harness には採用しない
- local object storage lane の acceptance bar は、
  AWS 完全互換ではなく `arena-service` の artifact write と delegated download URL contract verification とする
- local object storage data dir の default は repo/worktree 内の ignored path とし、
  `.local/seaweed/` を第一候補にする
- default bootstrap は stale object を引き継がない reset-first 運用とし、
  必要な seed object は script で毎回投入する
- seed object script は backend data dir を直接触らず、
  S3-compatible API を通すことで local lane でも object storage contract を崩さない
