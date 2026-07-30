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
- 実行完了時は matching plan と解決済みの linked local issue を削除する。linter は merge-base 側の削除済み plan から `Addresses:` を復元して検査する。
- 完了済み task tracker は plan PR、実装 PR、または Git 履歴で追跡する。永続的な判断は仕様、ADR、コード、または開発文書に残す。
