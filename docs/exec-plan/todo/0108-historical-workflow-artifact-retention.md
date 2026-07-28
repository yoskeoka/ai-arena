# 完了済み workflow artifact の保持廃止

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的

通常の検索対象を active な `docs/exec-plan/todo/` と未解決 `docs/issues/` に限定する。完了済み plan / local issue は実装 PR で削除し、plan PR・実装 PR・Git 履歴からだけ参照可能にする。

## 対象と変更

- (MODIFY) `AGENTS.md`、`docs/exec-plan/todo/README.md`、`docs/issues/README.md`、`.github/PULL_REQUEST_TEMPLATE.md`、`docs/development/workflow-linter.md`：`done/` 移動ではなく削除と Git/PR 履歴での検索を契約化する。
- (MODIFY) `tools/workflow-lint.sh` と既存テスト：active plan の必須性を保ち、完了時は merge-base 側で削除された matching plan の `Addresses:` を読み、linked local issue の削除と外部 issue の closure metadata を検証する。
- (DELETE) `docs/exec-plan/done/**`、`docs/issues/done/**` と空になるディレクトリ。永続判断は仕様・ADR・コードに残し、task tracker は複写しない。

## 実施順序

1. `docs/development/workflow-linter.md` を先に更新し、削除ベースの完了契約と監査経路を定義する。
2. workflow guide、template、linter を同一契約へ揃え、deleted plan の base-side 解析を実装・テストする。
3. 既存 done artifact を削除し、tracked current guidance に stale path がないことを確認する。

## 検証

- active `feat/*` / `fix/*` に matching todo plan が無い場合は従来どおり fixable warning になる。
- matching plan と linked local issue の同時削除は受理され、issue 未削除は fixable warning、external issue は `Closes` metadata を要求する。
- `go test ./...`、該当する workflow-linter tests、`git diff --check`、既存の lint/verify command を実行する。
- `rg` で current workflow guidance に `exec-plan/done` / `issues/done` が残らず、`git log --all -- docs/exec-plan` で削除済み plan を追跡できることを確認する。
