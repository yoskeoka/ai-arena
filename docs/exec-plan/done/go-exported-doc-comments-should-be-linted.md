# go-exported-doc-comments-should-be-linted
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の Go quality gate に exported doc comment / package comment の検査を追加し、公開 API と package surface の基本説明が欠けたまま CI を通過しない状態にする。

この plan は以下を成立させる。

- `make lint` で exported identifier と package comment の欠落を検出できる
- `golangci-lint` は導入せず、repo の既存方針どおり個別 tool pinning を維持する
- `games/dungeon/**` にある現行の exported comment 不足を execution PR 内で埋め、lint を即時有効化できる
- `docs/specs/go-quality-gates.md` と実際の lint suite を一致させる

Addresses:

- `docs/issues/go-exported-doc-comments-should-be-linted.md`

## Context

- `docs/project-plan.md` は platform 実装言語として Go を採用しており、ゲーム package や helper package を継続的に増やしていく前提にある
- `docs/design-decisions/core-beliefs.md` は correctness over speed と spec-first を優先しているため、comment policy も code 変更前に spec へ反映する
- `docs/design-decisions/adr.md` は Go 採用の trade-off として、仕様とテストで保守性を補強する必要があるとしている
- `docs/specs/go-quality-gates.md` と `docs/exec-plan/done/go-ci-make-targets.md` は `golangci-lint` を導入せず、必要な checker を個別 tool として pin する方針を既に固定している
- `docs/issues/go-exported-doc-comments-should-be-linted.md` は exported package / type / const / func の doc comment と package comment が `make lint` で未検出なことを問題としており、repo 全体へ適用できる一般的な外部 linter の採用を求めている

## Scope

- Go doc comment lint rule の追加
- lint 実行対象と rule 設定の repository-level 明文化
- `games/dungeon/**` の現行 lint failure 解消に必要な comment 追加
- `docs/specs/go-quality-gates.md` の lint suite 更新

この plan では以下は扱わない。

- `golangci-lint` への移行
- Go 以外の言語向け comment/style lint
- doc comment 以外の style rule の一括導入
- comment lint suppressions を前提にした段階導入

## Design Decisions

この plan では追加 ADR は作らない。

- exported doc comment と package comment の検査には、一般的に使われ保守されている外部 linter として `revive` を採用する
- `golangci-lint` は導入せず、既存の `make lint` 契約に沿って `revive` を個別 tool として pin する
- 有効化する `revive` rule は、今回の issue で必要な `exported` と `package-comments` を最小セットとして始める
- `main` package や test helper まで無差別に広げるのではなく、実際の package 構成に即して rule 対象と必要な除外を明文化する
- 段階導入ではなく、現行不足 comment を同一 execution PR で埋めた上で lint を有効化する

## Spec Changes

### `docs/specs/go-quality-gates.md`

以下を明記する。

- `make lint` に `revive` を追加し、Go comment policy の入口を同 target に揃える
- `revive` は個別 tool pinning の一部として扱い、`golangci-lint` は引き続き導入しない
- 常設 rule として `exported` および `package-comments` を有効化する
- package comment requirement の適用境界と、必要なら `foo_test` のような external test package や `cmd/**` の `package main` の扱いを明文化する

### Optional adjacent wording sync

- `docs/specs/README.md` などで `go-quality-gates.md` の説明が lint suite の中身に依存している場合は、一覧表現を最新化する

## Expected Code Changes

### `Makefile`

- `revive` 実行用の lint target を追加する
- `make lint` の既定 suite に新 target を組み込む
- 既存 cache / pinned tool invocation の流儀に合わせて `go tool` または同等の再現可能な呼び出しに揃える

### `go.mod` / `go.sum` / tool pinning files

- `revive` の version を明示的に pin する
- ローカルと CI が同じ version を再現できるようにする

### `games/dungeon/**`

- exported package / type / const / func / method の不足 comment を追加する
- package comment が別 file にある前提を保ちつつ、rule が期待する形に wording を揃える

### optional config file

- `revive.toml` などの設定ファイルを追加し、必要最小限の rule と除外境界を repo で固定する

## Sub-tasks

- [ ] Update `docs/specs/go-quality-gates.md` to document `revive` as part of the default lint suite
- [ ] Decide the minimal `revive` configuration needed for `exported` and `package-comments`
- [ ] Pin `revive` in the repo's Go toolchain
- [ ] Add `revive` execution to `make lint`
- [ ] Add the config file or equivalent invocation wiring needed to keep the rule set explicit
- [ ] Add missing doc comments in `games/dungeon/**` so the new lint passes immediately
- [ ] Verify `make lint` fails without the comments and passes after the fixes

## Parallelism

- [parallel] Spec wording and `revive` config drafting can proceed independently once the rule surface is fixed
- [parallel] Tool pinning and `Makefile` wiring can proceed independently of dungeon comment authoring
- Final verification depends on both lint wiring and in-scope comment fixes landing together

## Risks and Mitigations

- `revive` の default 振る舞いを広く有効化すると、今回の issue を超える style finding が大量に出る恐れがある
  - mitigation: 初回導入では `exported` と `package-comments` に scope を絞り、rule を明示設定する
- package comment rule は package ごとの file 構成に依存するため、期待する file に comment があっても設定次第で誤検出に見える可能性がある
  - mitigation: config file と対象 package の実データで確認し、必要なら narrow な除外や wording 修正で吸収する
- lint 追加だけを先に入れると main branch が直ちに赤くなる
  - mitigation: execution PR では lint wiring と `games/dungeon/**` comment 補完を同時に行う
- custom checker を自作しない代わりに external tool の設定理解が必要になる
  - mitigation: widely used な `revive` に寄せ、rule 数も最小限にして保守コストを抑える

## Verification

The execution PR is complete when the following are true.

- `make lint` includes the `revive`-based doc comment check
- Missing exported/package comments in `games/dungeon/**` are fixed and no longer fail lint
- The repo still does not depend on `golangci-lint`
- `docs/specs/go-quality-gates.md` matches the final lint suite and rule boundary
