# platform-online-foundation-01-04-cli-proof-and-e2e-verification
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`match submission -> validation -> queue -> worker dispatch -> runner match execution -> terminal persist` の
最小 online flow を CLI 主体で通し、0045 系の到達点を acceptance として固定する。

## Context

- `0049` が validated submission を queue に載せる
- `0050` が queued match を runner 実行して terminal persist を閉じる
- 0045 系の価値は、public API の整備より前に、runner の一段外側の運営 flow が実際に動くことを確認する点にある

## Scope

- CLI から submission を投入し、worker 実行で結果が保存される e2e を整える
- rejection path と queued-only cancel path を確認する
- artifact layout と stderr log 出力を確認する
- 0046 / 0047 に送る未実装事項を明確に残す

この plan では以下を扱わない。

- retry verification
- ranking / matchmaking verification
- replay / resume verification の厚い coverage
- HTTP API acceptance

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- acceptance scenario と non-goals を追記する
- CLI-first の検証導線を追記する

### 必要なら更新

- `docs/specs/platform.md`
  - service skeleton 経由の artifact 読取順や operator-facing confirmation を補足する

## Expected Code Changes

- end-to-end acceptance test or script
- rejection / cancel / success path の verification assets
- reviewer 向け最小 manual verification 手順

## Sub-tasks

- [ ] success path の e2e を追加する
- [ ] validation rejection path の e2e を追加する
- [ ] queued-only cancel path の e2e を追加する
- [ ] `record.json` / stderr log / summary artifact の確認手順を追加する
- [ ] 0046 / 0047 に残す TODO を明文化する

## Parallelism

- `0049` と `0050` が揃った後は、success / rejection / queued cancel の各 verification lane を分担して進められる

## Dependencies

- depends on: `0049-platform-online-foundation-01-02-submission-entry-and-queue-write.md`
- depends on: `0050-platform-online-foundation-01-03-worker-dispatch-and-terminal-persist.md`
- informs: `0046-platform-online-foundation-02-persistence-and-read-model.md`
- informs: `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md`

## Risks and Mitigations

- success path だけ通して failure-mode の contract が曖昧なまま残る
  - mitigation: rejection / queued cancel を同じ plan で押さえる
- e2e を厚くしすぎて 0045 系の planning scope を越える
  - mitigation: 最小 online flow の acceptance に限定し、resume/replay/ranking は後続へ送る

## Design Decisions

- 0045 系の completion bar は、CLI から queue system を動かして match 実行と terminal persist まで検証できることに置く
