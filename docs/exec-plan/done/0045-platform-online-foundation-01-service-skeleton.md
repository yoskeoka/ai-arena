# platform-online-foundation-01-service-skeleton
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 6/7 の入口として、ai-arena を online platform として動かす最小 service skeleton の到達点を定義する。
最初のゴールは、`submission -> validation -> queue -> match run -> result persist` を 1 本の運営フローとして通し、
CLI 主体でもよいので「人が運営できる最小の platform」を成立させることに置く。

## Context

- `docs/project-plan.md` では、Phase 6 が永続化基盤と service skeleton、Phase 7 が online 運営基盤を扱う
- Phase 5 までで、game 固有実装を別 repo へ逃がせる platform / SDK / artifact / registry 境界はおおむね整った
- 一方で、現状は local runner / fixture / replay 契約が中心で、online 運営の流れはまだ 1 本通っていない
- 現段階では interface 契約の細部を先に固めるより、全体フローを通して必要 surface を見極める方を優先する

## Scope

- AI submission と registered game を使って match を投入する最小 service entrypoint を定義する
- validation / queueing / worker dispatch / result persist の責務境界を定義する
- 認証、課金、本格 UI、複数 tenant 運用のような production feature はこの plan では扱わない
- この plan は親 plan として扱い、実装前に service API、worker loop、storage boundary などの child plan へ分解する

## Spec Changes

### `docs/specs/platform.md`

- online service から見た runner / worker の責務境界を追記する
- match 実行の source-of-truth artifact が persist 前提で扱われることを明記する

### 新規または更新 spec

- submission / queue / execution lifecycle を扱う service skeleton spec を追加する
- operator が扱う最小 match state machine と failure / retry 方針の spec を追加する

## Expected Code Changes

- service entrypoint または CLI から online match を投入する最小 surface
- queued match を実行する worker loop
- validation, queue record, execution result persist を束ねる orchestration 層
- integration test または e2e で最小 online flow を確認する検証導線

## Sub-tasks

- [ ] 現行 runner / registry / artifact contract のうち service skeleton が前提にする部分を棚卸しする
- [ ] service entrypoint と worker の責務境界を spec に落とす
- [ ] [parallel] queue / lifecycle / persist の state model 候補を比較する
- [ ] 実装前に child plan へ分解し、execution order を確定する

## Parallelism

- service entrypoint の整理と queue / lifecycle state model の比較は並行できる
- child plan 分解後は API surface、worker loop、verification path を別 lane に分けられる可能性が高い

## Risks and Mitigations

- online flow を一度に広げすぎると、service skeleton ではなく未確定の運営機能まで抱え込む
  - mitigation: 最初の到達点を 1 試合投入して結果を保存できる最小フローに限定する
- 既存 runner 契約をそのまま service 化できるとは限らない
  - mitigation: 先に end-to-end flow を通し、必要になった public / internal seam だけを次段で締める

## Design Decisions

- 現時点では、public contract hardening より service skeleton 先行を優先する
- この plan 自体は parent/base plan であり、実装着手前に同じ `platform-online-foundation-01` 系の child plan へ分割する前提とする
