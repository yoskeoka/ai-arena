# platform-online-foundation-03-04-matchmaking-ranking-follow-up-01-phase6-release-flow
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 6 を local 検証、CI、staging deploy、staging verification、production release の
一連の release flow として閉じる。
最初のゴールは、online service first landing を「動く構成」だけでなく
「再現可能に出せる構成」として repo-local workflow に固定することに置く。

## Context

- `0064` / `0065` / `0066` と `0068` / `0069` / `0070` により、
  provider inventory、remote polling API、minimal operator UI、local/CI browser lane はそろっている
- user decision として、Phase 6 の意義は local 確認に留まらず、
  `local -> CI -> stg -> stg verification -> prod` の release flow を先に作ることにある
- `stg` / `prod` の URL、provider asset inventory、desired deploy policy は
  `docs/development/platform-service-online-deploy.md` に記録済みだが、
  repo-owned workflow / runbook としてはまだ閉じていない
- backend に危険な mutation surface はまだ薄いが、release flow が曖昧なまま
  Phase 7 を積み始めると online confirmation の価値が薄れる

## Scope

- repo-owned な staging deploy / production release workflow を定義する
- staging deploy 後に実施する verification contract を定義する
- local verification と CI browser lane を release gate の前段として接続する
- provider drift check、deploy artifact build、manual promotion 手順を文書化する

この plan では以下を扱わない。

- general AI submission / matchmaking / ranking の product 機能追加
- public signup / login の実装
- custom domain 導入そのもの

## Spec Changes

この plan では product spec を増やさず、development workflow を主対象とする。

### 更新 document

- `docs/development/platform-service-online-deploy.md`
  - local の次に読む section として、CI、staging deploy、staging verification、
    production release の順序を固定する
  - required secret / env / artifact / verification evidence を明記する

## Expected Code Changes

- staging Pages preview deploy workflow
- staging backend deploy workflow または documented trigger helper
- production release workflow または documented promotion helper
- release verification helper / checklist

## Sub-tasks

- [ ] local / CI / staging / production の release gate を棚卸しする
- [ ] staging deploy の canonical trigger と input/output を固定する
- [ ] staging verification の acceptance surface を local/CI lane とそろえる
- [ ] production release を staging 確認済み commit SHA 昇格に固定する
- [ ] deploy / release / rollback の runbook を整理する

## Parallelism

- [parallel] workflow inventory の棚卸しと runbook 叩き台作成は並行できる
- [depends on: staging deploy contract] staging verification helper と production promotion 手順を詰める

## Dependencies

- depends on: `0064-platform-online-foundation-03-01-provider-bootstrap-and-remote-artifact-delivery.md`
- depends on: `0065-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- depends on: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`
- depends on: `0068-platform-online-foundation-03-05-operator-ui-verification-01-local-agent-browser-loop.md`
- depends on: `0069-platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane.md`
- depends on: `0070-platform-online-foundation-03-05-operator-ui-verification-03-real-local-browser-operator-lane.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- release flow を workflow ではなく口頭運用に残すと、Phase 7 以降の online verification が再現不能になる
  - mitigation: repo-owned workflow と runbook の両方に落とす
- staging と production の deploy trigger が drift すると、検証済み artifact と本番反映物がずれる
  - mitigation: production は staging で確認済み commit SHA の明示昇格に固定する

## Design Decisions

- Phase 6 completion は local / CI success だけでなく、staging deploy と production promotion まで含む
- production release は auto deploy ではなく manual promotion を維持する
