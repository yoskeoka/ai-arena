# platform-online-foundation-01-01-service-contract-and-lifecycle
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`platform-online-foundation-01-service-skeleton` の最初の child plan として、
matchmaking 済みの `game + players` を試合実行へ流す最小 online contract を固定する。
最初のゴールは、runner の 1 段外側に置く service skeleton について、
`match submission -> validation -> queue -> worker claim -> runner invoke -> terminal persist` の責務境界と
state machine を spec-first で確定することに置く。

## Context

- parent plan `0045-platform-online-foundation-01-service-skeleton.md` は、service skeleton を child plan 群へ分割する前提で作られていた
- `docs/specs/platform.md` には single-match runner / record artifact contract があるが、queue / worker / service lifecycle は未定義である
- Phase 6 の最初の到達点は、matchmaking 後に得た `game + players` の組を queue 経由で runner へ渡し、結果を保存する flow を成立させることである
- operator flow / matchmaking / ranking は `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md` の責務であり、この plan では扱わない

## Scope

- `match submission` の最小 schema を定義する
- admission validation の責務範囲を定義する
- queue / execution lifecycle state machine を定義する
- service / worker / runner / registry / game master の責務境界を定義する
- terminal persist 時点で最低限保存する正本と summary を定義する

この plan では以下を扱わない。

- public HTTP API
- AI upload registry の durable contract
- matchmaking, ranking, rerun policy
- replay / resume read model の詳細
- distributed worker 運用や retry 自動化

## Fixed Decisions

- `submission` は AI upload ではなく、1 回の試合実行要求を表す `match submission` とする
- `artifact_ref` は file path 固定ではなく opaque locator / URI として表現する
- queue に入る前に admission validation を完了させる
- admission validation では metadata / manifest / registry check に加えて、runner 側 dry-run 入口の導入方針を取る
- cancel は `queued` 中のみ許可し、`leased` 以降は 0045 系では扱わない
- terminal persist は既存 `output-dir` を使う file-backed first とする
- AI stderr は `output-dir` 配下へ player ごとの `*-stderr.log` として出力する
- retry は実装しない。`attempt_count` は持ってよいが、初期は `1` 固定運用とする

## Spec Changes

### `docs/specs/platform.md`

- runner が single-match execution engine であり、queue ownership を持たないことを明記する
- service / worker / runner の責務境界を追記する
- terminal `record.json` と player stderr artifact が service skeleton の保存対象になることを追記する

### 新規 spec

- `docs/specs/platform-service-skeleton.md`
  - `match submission` schema
  - admission validation scope
  - queue / execution lifecycle state machine
  - worker claim / runner invoke / terminal persist boundary
  - cancel / retry の初期制約

## Expected Code Changes

- service skeleton 向けの internal contract package または module
- `match submission` と queue record の型
- validation / lifecycle / runner invocation の interface 定義
- spec contract を検証する unit test

## Sub-tasks

- [ ] `match submission` の最小項目と opaque locator rule を定義する
- [ ] admission validation と runner dry-run の責務分離を定義する
- [ ] queue / execution lifecycle state machine を定義する
- [ ] service / worker / runner / registry / game master の責務境界を定義する
- [ ] terminal persist minimum と stderr artifact rule を定義する

## Parallelism

- schema 整理と lifecycle state machine の比較は並行できる
- ただし最終 spec の統合は同一 plan 内で順に確定する
- 後続の `0049` と `0050` は、この plan の contract が固まった後なら並行で進められる

## Dependencies

- depends on: parent/base item `0045-platform-online-foundation-01-service-skeleton.md` (now retired to `docs/exec-plan/done/` after split)
- blocks: `0049-platform-online-foundation-01-02-submission-entry-and-queue-write.md`
- blocks: `0050-platform-online-foundation-01-03-worker-dispatch-and-terminal-persist.md`
- informs: `0046-platform-online-foundation-02-persistence-and-read-model.md`
- informs: `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md`

## Risks and Mitigations

- runner dry-run を広く取りすぎると、本実行と別の複雑な入口を増やしやすい
  - mitigation: admission validation で必要な最小確認に限定し、full match 実行は worker path に残す
- service lifecycle と match lifecycle が混線すると責務が崩れる
  - mitigation: queue / execution state と `record.json.status` を別 contract として明記する

## Design Decisions

- 0045 系の最小 scope は、matchmaking 後の組を queue 経由で runner に渡せるようにすることに置く
- online service の public contract はまだ固定せず、internal service contract から始める
