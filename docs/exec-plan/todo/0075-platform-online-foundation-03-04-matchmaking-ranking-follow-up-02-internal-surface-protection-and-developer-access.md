# platform-online-foundation-03-04-matchmaking-ranking-follow-up-02-internal-surface-protection-and-developer-access
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 6 の staging / production operator surface を、一般公開前の internal surface として
どう保護し、どう開発者がアクセスするかを固定する。
最初のゴールは、free-tier 前提で現実的な保護方法と、repo に置いてよい access inventory を整理することに置く。

## Context

- operator API / UI は private surface として扱っているが、
  auth / authorization の最終方式はまだ deferred である
- user decision として、現時点では custom domain なしで進める
- Render free では backend に IP allowlist を期待できず、
  Pages / preview だけを閉じても `onrender.com` backend は別問題として残る
- 現在の backend は高価値 mutation surface をほぼ持たない一方で、
  env や process memory 露出につながる脆弱性は避けなければならない

## Scope

- staging / production の internal operator surface protection option を比較し、第一選択を固定する
- Cloudflare Access を staging / internal surface protection としてどう使うかを整理する
- backend 直アクセス問題を documented risk として切り出す
- 開発者アクセス runbook に、公開してよい URL / ID / credential 取得方法だけを残す

この plan では以下を扱わない。

- public product auth の実装
- custom domain 導入
- Render backend の network allowlist 実装

## Spec Changes

この plan では product spec を増やさず、development / ops runbook を主対象とする。

### 更新 document

- `docs/development/platform-service-online-deploy.md`
  - Cloudflare Access、Render custom domain、free-tier 制約、developer access inventory を追記する
  - repo に commit してよい情報と secret manager にのみ置く情報を分ける

## Expected Code Changes

- staging preview protection workflow または運用手順
- access inventory / secret handling runbook
- 必要なら minimal backend hardening issue / deferred plan へのリンク

## Sub-tasks

- [ ] current internal surface と protected surface の境界を棚卸しする
- [ ] Cloudflare Access / app auth / no-auth-yet の比較を整理する
- [ ] staging / production developer access runbook を書く
- [ ] backend direct exposure の現時点判断と将来の closure 条件を記録する

## Parallelism

- [parallel] provider facts の整理と developer access runbook 叩き台は並行できる
- [depends on: protection option decision] workflow / env naming を最終化する

## Dependencies

- depends on: `0074-platform-online-foundation-03-04-matchmaking-ranking-follow-up-01-phase6-release-flow.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- Pages preview を閉じただけで backend protection ができたと誤認すると、Phase 7 の攻撃面を見誤る
  - mitigation: backend direct access を別論点として明記する
- credential 配布方法を repo に書き過ぎると secret leakage を招く
  - mitigation: repo には URL / ID / issuance path だけを残し、secret value 自体は記録しない

## Design Decisions

- custom domain なしを current path とし、Cloudflare Access は staging / internal surface の保護候補として扱う
- public GA 前の true product auth は後続 child plan に分離する
