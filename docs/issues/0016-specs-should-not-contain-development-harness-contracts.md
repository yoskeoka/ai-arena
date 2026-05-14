# Product specs should not contain development harness contracts

## Problem

`docs/specs/` には ai-arena platform / game / runtime など、product contract を置きたい。
一方で Codex hook、local quality gate、AI agent workflow のような development harness の契約が
`docs/specs/` に混ざると、product behavior と contributor workflow が同じ層に見えてしまう。

今回の `docs/specs/codex-hooks.md` はその典型で、repo-local Codex hook wiring のような
開発環境依存の話が product spec として扱われていた。

## Why it matters

- product contract と local tooling contract の境界が曖昧になる
- review 時に「何が user-facing / platform-facing contract なのか」が読み取りにくくなる
- Codex / CI / local helper の変更が product spec diff に見えてしまう

## Requested direction

- `docs/specs/` には product behavior / platform contract だけを残す
- development harness の契約は別の置き場に移す
- 既存の spec のうち harness 寄りのものを棚卸しし、どこまでを product spec に残すか整理する

## Candidate follow-up

- `docs/specs/` 配下で development harness 寄りの文書を洗い出す
- 移設先の候補を決める
- spec index と related references を整理する
