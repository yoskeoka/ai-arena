# Development docs

`docs/development/` には、`ai-arena` の開発運用文書を置く。
対象は contributor workflow、repo-local quality gate、CI / automation maintenance である。

`docs/specs/` との境界:

- `docs/specs/`: `ai-arena` が提供する platform / runner / runtime / game contract の正本
- `docs/development/`: contributor workflow、quality gate、CI maintenance、repo-local tooling 運用

現時点の主要文書:

- `docs/development/go-quality-gates.md`: Go module の quality gate と CI lane の運用
- `docs/development/japanese-textlint.md`: 日本語 `docs/**/*.md` 向け textlint CI と辞書運用
- `docs/development/github-actions-pinning.md`: GitHub Actions の `uses:` pinning 運用
- `docs/development/operator-ui-local-verification.md`: operator UI local browser verification の canonical Playwright lane
- `docs/development/platform-service-postgres.md`: durable queue backend 用 Postgres の schema/apply/query-generation workflow と local / CI harness
- `docs/development/platform-service-online-deploy.md`: provider inventory、staging / production release flow、internal surface protection、developer access inventory
- `docs/development/workflow-linter.md`: local workflow linter の maintenance 契約

online release workflow の dispatch / verification / rollback runbook も
`docs/development/platform-service-online-deploy.md` を正本とする。
