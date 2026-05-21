# platform-online-foundation-01-02-submission-entry-and-queue-write
**Execution**: Use `/execute-task` to implement this plan.

## Objective

CLI から `match submission` を受け、queue へ載せる最小 operator entry を実装する。
最初のゴールは、matchmaking 済みの `game + players` の組を opaque artifact locator 付きの submission として受理し、
admission validation を通過したものだけを queue record として保存できるようにすることに置く。

## Context

- `0048-platform-online-foundation-01-01-service-contract-and-lifecycle.md` が service skeleton contract を固定する
- 0045 系では public HTTP API ではなく CLI-first で始める
- queue に未検証の submission を入れると DLQ / retry 運用を先に背負うため、この段階では queue 前 validation を採る

## Scope

- CLI から `match submission` を受ける entrypoint を追加する
- service command 経由で admission validation を実行する
- validation 済み submission だけ queue record を作成する
- `queued` までの lifecycle を扱う
- `queued` 中だけ cancel できる最小導線を扱う

この plan では以下を扱わない。

- worker dispatch
- match 実行
- terminal persist
- retry / DLQ
- HTTP API

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- CLI-first operator entry を追加する
- submission create / validate / queue write / queued-only cancel の contract を追記する
- opaque artifact locator の accepted forms と validation timing を明記する

### `docs/specs/ai-runtime.md`

- 必要なら online submission validation から見た sidecar / entrypoint 解決前提を補足する

## Expected Code Changes

- operator-facing CLI command for match submission
- submission store / queue write path
- admission validation orchestration
- queued-only cancellation path
- queue write path の integration test

## Sub-tasks

- [ ] CLI input から `match submission` を組み立てる
- [ ] artifact locator / sidecar / registry / dry-run validation を束ねる
- [ ] validation success 時だけ queue record を作成する
- [ ] `queued` 中 cancel を実装する
- [ ] success / rejection / cancel の integration test を追加する

## Parallelism

- `0048` の contract 固定後は、CLI entry / validation path と `0050` の worker path を並行で進められる

## Dependencies

- depends on: `0048-platform-online-foundation-01-01-service-contract-and-lifecycle.md`
- blocks: `0051-platform-online-foundation-01-04-cli-proof-and-e2e-verification.md`
- informs: `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md`

## Risks and Mitigations

- validation logic を CLI に寄せすぎると将来 API 化しづらい
  - mitigation: CLI は adapter に留め、service command が validation と queue write を所有する
- opaque locator を早く複雑化しすぎる
  - mitigation: URI shape を受けるが、初期実装は file-backed / local storage path に絞ってよい

## Design Decisions

- queue には validation 済み submission だけを入れる
- cancel は `queued` 中だけを対象にする
