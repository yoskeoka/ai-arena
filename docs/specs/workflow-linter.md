# Workflow Linter

`ai-arena` installs a local `tools/workflow-lint.sh` and `.githooks/pre-push` from the vendored workflow repository under `.claude/vendor/workflow/`.

The local copies are expected to track the vendored workflow implementation closely. Repo-specific wording may point at `.claude/vendor/workflow/AI_WORKFLOW.md` and `.claude/vendor/workflow/docs/specs/ww-dogfooding-workflow.md`, because those are the canonical workflow documents available in this repository.

The linter remains warnings-only. When the vendored workflow is updated, refresh both:

- `.claude/vendor/workflow`
- the local root copies at `tools/workflow-lint.sh` and `.githooks/pre-push`

`docs/issues/done/` is the archive location for resolved local issues referenced by the workflow linter.
