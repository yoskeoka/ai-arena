# Development Docs

`docs/development/` は、`ai-arena` の contributor workflow、repo-local quality gate、
CI / automation maintenance のような development harness 文書を置く。

`docs/specs/` との境界:

- `docs/specs/`: `ai-arena` が提供する platform / runner / runtime / game contract の正本
- `docs/development/`: contributor workflow、quality gate、CI maintenance、repo-local tooling 運用

現時点の主要文書:

- `docs/development/go-quality-gates.md`: Go module の quality gate と CI lane の運用
- `docs/development/github-actions-pinning.md`: GitHub Actions の `uses:` pinning 運用
- `docs/development/workflow-linter.md`: local workflow linter の maintenance 契約
