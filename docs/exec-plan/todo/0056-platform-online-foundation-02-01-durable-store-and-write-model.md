# platform-online-foundation-02-01-durable-store-and-write-model
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0049` / `0050` で導入した queue lifecycle と terminal artifact persist を、
process 境界をまたいでも保持できる durable write model へ進める。
最初のゴールは、in-memory 実装のまま残っている `QueueStore` を replaceable なまま durable backend に差し替え、
submission 受理から terminal artifact locator 保存までを再起動耐性のある service state として閉じることに置く。

## Context

- `0049` は `QueueStore` interface を切り、初期実装として `InMemoryQueueStore` を入れた
- `0050` は runner terminal artifact を `output_dir/<match_id>/` に保存し、`QueueRecord.Terminal` に locator を戻す形を固定した
- 現状は queue 順序、lease 状態、terminal locator がすべて process memory にしか残らず、service/worker を分けた運用へ進めない
- `0046-platform-online-foundation-02-persistence-and-read-model.md` の親 scope のうち、write path の正本化を先に閉じる必要がある

## Scope

- `MatchSubmission` / `QueueRecord` / `TerminalArtifacts` を durable に保持する write model を定義する
- queued / leased / running / persisting / terminal の lifecycle を durable backend 上で扱えるようにする
- terminal artifact 本体は既存どおり file-backed artifact layout に残し、write model 側には locator と最小 summary だけを持たせる
- single-node 前提の cross-process queue 共有を成立させる
- `0049` / `0050` の service command / worker path を durable backend に載せ替える

この plan では以下を扱わない。

- operator-facing list/get/read API
- replay / resume / audit 向け rebuild API
- distributed lock / multi-node fairness / Redis 等の外部 queue
- 長期 retention / archive policy

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- queue write / claim / update / cancel / terminal persist が durable backend 越しに進む contract を追記する
- `output_dir` 配下 artifact と service write model の責務分離を明記する
- single-node cross-process 共有までは扱うが、multi-node fairness は対象外であることを明記する

### `docs/specs/platform.md`

- service skeleton の write model が保持する最小 lifecycle state と、runner artifact 正本の関係を補足する

### 新規または更新 spec

- persistence model spec を追加または更新し、submission / queue lifecycle / terminal locator の保存単位を定義する

## Expected Code Changes

- durable queue/write-store interface の contract 整理
- single-node durable backend 実装
- command / worker の durable backend 対応
- lifecycle recovery / cross-process 共有の integration test

## Sub-tasks

- [ ] `0049` / `0050` 実装で実際に write model へ残すべき field を棚卸しする
- [ ] durable write model の保存単位と state transition contract を spec に落とす
- [ ] durable backend を追加し、既存 `QueueStore` 差し替え seam に接続する
- [ ] queued-only cancel / claim / terminal update が durable backend 上でも同じ contract を満たすことを確認する
- [ ] process 再起動または service/worker 分離を想定した integration test を追加する

## Parallelism

- write model の field 棚卸しと spec 追記は並行できる
- backend 実装着手後は command path と worker path の接続を分担できる

## Dependencies

- depends on: parent/base item `0046-platform-online-foundation-02-persistence-and-read-model.md` (to be retired to `docs/exec-plan/done/` after split)
- informed by: `0049-platform-online-foundation-01-02-submission-entry-and-queue-write.md`
- informed by: `0050-platform-online-foundation-01-03-worker-dispatch-and-terminal-persist.md`
- blocks: `0057-platform-online-foundation-02-02-result-read-model-and-operator-query.md`
- blocks: `0058-platform-online-foundation-02-03-replay-resume-audit-inputs.md`

## Risks and Mitigations

- in-memory 前提の helper が durable backend 詳細へ漏れると差し替え seam が崩れる
  - mitigation: command / worker は `QueueStore` 契約に留め、backend 固有処理を service orchestration へ漏らさない
- artifact 本体まで write model に重複保存すると source-of-truth が曖昧になる
  - mitigation: `record.json` / `snapshot.json` / `history.json` は file-backed artifact を正本に保ち、store には locator と lifecycle 要約だけを保持する
- durable 化の勢いで distributed queue 前提を背負うと scope が膨らむ
  - mitigation: single-node cross-process 共有を到達点に固定し、multi-node fairness は後続へ送る

## Design Decisions

- `0049` の replaceable store seam は維持し、durable 化はその差し替えとして行う
- artifact directory 自体を store backend に埋め込まず、service write model と runner artifact layout を分離する
- 最初の durable backend は single-node 運用を成立させることを優先し、分散 queue は対象外とする
