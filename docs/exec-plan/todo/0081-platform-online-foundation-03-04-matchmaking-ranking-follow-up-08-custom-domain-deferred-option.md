# platform-online-foundation-03-04-matchmaking-ranking-follow-up-08-custom-domain-deferred-option
**Execution**: Use `/execute-task` to implement this plan.

## Objective

custom domain を導入する deferred option を保留 plan として記録する。
最初のゴールは、今は custom domain なしで進めつつ、後から導入判断するときに
Render / Pages / Access / naming 候補の再調査を不要にすることに置く。

## Context

- user decision として、current path は custom domain なしで進める
- Render free でも custom domain は利用可能であり、Cloudflare Registrar で廉価な domain を取得する余地はある
- ただし current backend には高価値 mutation surface がまだ少なく、今すぐ domain 購入が必須ではない
- future product auth の shape 次第では custom domain 不要判断に戻る可能性もある

## Scope

- custom domain 導入時の provider facts と naming criteria を文書化する
- Pages / Render / Cloudflare Access と custom domain の接続点を整理する
- `.com` / `.net` / `.dev` を中心に naming evaluation ルールを定義する

この plan では以下を扱わない。

- 実際の domain purchase
- DNS cutover
- custom domain を前提にした auth 実装

## Spec Changes

この plan では product spec を増やさず、development / deploy runbook を主対象とする。

### 更新 document

- `docs/development/platform-service-online-deploy.md`
  - custom domain deferred option と provider facts を記録する

## Expected Code Changes

- deploy / access runbook 更新のみ、または最小 helper

## Sub-tasks

- [ ] Render free custom domain support と cost boundary を記録する
- [ ] Cloudflare Registrar / custom domain cost expectation を記録する
- [ ] naming criteria を整理する
- [ ] trigger condition を定義する

## Parallelism

- [parallel] provider fact 記録と naming option 整理は並行できる

## Dependencies

- depends on: `0075-platform-online-foundation-03-04-matchmaking-ranking-follow-up-02-internal-surface-protection-and-developer-access.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Risks and Mitigations

- deferred option を残さないと後続で provider fact を再調査するコストが発生する
  - mitigation: current docs に research result を過剰気味に残す

## Design Decisions

- custom domain は current path では導入しない
- ただし deferred option として plan を保持し、後続判断の再調査コストを下げる
