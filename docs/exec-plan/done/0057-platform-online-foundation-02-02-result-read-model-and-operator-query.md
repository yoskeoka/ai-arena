# platform-online-foundation-02-02-result-read-model-and-operator-query
**Execution**: Use `/execute-task` to implement this plan.

## Objective

durable write model と file-backed artifact から、operator が match 結果と保存先を読める最小 read model を整える。
最初のゴールは、`result-summary.json` / `record.json` / stderr log locator を起点に、
result list と match detail を stable に引ける operator-facing query surface を定義することに置く。

## Context

- `0051` で最小 online flow の acceptance は通ったが、結果確認は individual artifact path を手作業で辿る前提に留まっている
- `0050` の `TerminalArtifacts` は `record_path` / `result_summary_path` / `player_stderr_paths` を返すが、一覧や detail 用の read model はまだない
- `0049` の queue record は durable backend 未確定のままなので、operator query は `0056` の write model 固定後に設計するのが自然である
- `0046-platform-online-foundation-02-persistence-and-read-model.md` 親 scope のうち、result/read model 側をこの child plan で受け持つ

## Scope

- match result list と match detail の最小 read model を定義する
- queued / leased / running / persisting / completed|failed|canceled を含む service lifecycle state を operator が参照できる query surface を定義する
- persisted queue/write model と file-backed artifact locator を結びつける
- CLI-first の operator-facing list/get/read entrypoint を整える

この plan では以下を扱わない。

- replay / resume 継続実行の orchestration
- public HTTP API
- spectator 向け public state delivery
- ranking / matchmaking 集計

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- operator-facing list/get/read の最小 query contract を追記する
- `result-summary.json` を既定の compact result view とし、詳細参照時だけ `record.json` / stderr locator を辿る順序を明記する

### `docs/specs/platform.md`

- service lifecycle state と runner terminal match status を、operator query 上でどう見せ分けるかを補足する

### 新規または更新 spec

- persistence/read-model spec を追加または更新し、result list、match detail、artifact locator view の schema を定義する

## Expected Code Changes

- result list / match detail read model builder
- operator-facing CLI query command
- durable store と artifact locator を束ねる query service
- list/get/read integration test

## Sub-tasks

- [ ] operator が確認したい最小 field を `0051` の acceptance artifact から棚卸しする
- [ ] result list / match detail / artifact locator view の schema を spec に落とす
- [ ] persisted write model と artifact layout を束ねる query service を実装する
- [ ] CLI-first の list/get/read entrypoint を追加する
- [ ] queued / running / completed / failed / canceled を横断する integration test を追加する

## Parallelism

- schema 整理と CLI surface 設計は並行できる
- `0056` 完了後は query service と integration test を分担できる

## Dependencies

- depends on: `0056-platform-online-foundation-02-01-durable-store-and-write-model.md`
- depends on: parent/base item `0046-platform-online-foundation-02-persistence-and-read-model.md` (now retired to `docs/exec-plan/done/` after split)
- informed by: `0051-platform-online-foundation-01-04-cli-proof-and-e2e-verification.md`

## Risks and Mitigations

- read model が `record.json` 全量の単なる再露出になると compact confirmation path の価値が薄れる
  - mitigation: default view は `result-summary` と lifecycle 要約に寄せ、詳細時だけ artifact locator を返す
- queue lifecycle と runner match status を混同すると failure interpretation がぶれる
  - mitigation: service lifecycle と runner terminal status を別 field として保持し、spec でも分離して説明する
- HTTP API 前提の shape を先に背負うと scope が過大化する
  - mitigation: CLI-first query contract を先に固定し、transport には依存させない

## Design Decisions

- operator の既定確認導線は compact result view から始め、必要時のみ詳細 artifact へ降りる
- read model は durable write model と file-backed artifact の join として扱い、どちらか一方へ責務を寄せすぎない
- public API ではなく CLI-first で contract を固定する
