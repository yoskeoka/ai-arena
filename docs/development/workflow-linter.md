# Workflow Linter

`ai-arena` は、vendored workflow repository 由来の local `tools/workflow-lint.sh` と
`.githooks/pre-push` を使う。

## Maintenance Contract

- local copy は `.claude/vendor/workflow/` の workflow 実装と近い状態を保つ
- repo-specific wording は `.claude/vendor/workflow/AI_WORKFLOW.md` と
  `.claude/vendor/workflow/docs/specs/ww-dogfooding-workflow.md` を参照してよい
- linter は warnings-only とする
- vendored workflow 更新時は、少なくとも以下を同期確認する
  - `.claude/vendor/workflow`
  - `tools/workflow-lint.sh`
  - `.githooks/pre-push`

## Workflow File Naming Contract

- active exec-plans under `docs/exec-plan/todo/` use `<sequence>-<name>.md`
- active issues under `docs/issues/` use `<sequence>-<name>.md`
- execution branches map by the `-<name>.md` suffix
- When execution completes, delete the matching plan and any resolved linked local issues. The linter reconstructs `Addresses:` from the deleted plan on the merge base to validate closure.
- Track completed task artifacts via the plan PR, implementation PR, or Git history. Record long-lived decisions/contracts/operational knowledge in specs, ADRs, code, or appropriate development docs.
