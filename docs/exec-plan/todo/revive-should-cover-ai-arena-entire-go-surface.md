# revive-should-cover-ai-arena-entire-go-surface
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の `revive` 適用境界を `games/**` 限定から repo の Go surface 全体へ広げ、`exported` と
`package-comments` の 2 rule を `make lint` / CI の常設 gate として一貫して扱える状態にする。

この plan は以下を成立させる。

- `games/**` だけでなく `cmd/**`、`internal/**`、`e2e/**`、`testdata/**` を含む全 Go package が
  同じ comment policy で検査される
- `golangci-lint` を導入せず、既存方針どおり個別 tool として pin した `revive` を維持する
- rule 追加は行わず、現行の `exported` と `package-comments` だけを repo-wide に適用する
- `revive.toml` 単独変更時も CI が漏れずに走る
- `docs/specs/go-quality-gates.md` と実際の lint / workflow 境界が一致する

## Context

- `docs/project-plan.md` は platform 実装言語として Go を採用し、platform / game / helper / fixture を継続追加する前提にある
- `docs/design-decisions/core-beliefs.md` は correctness over speed と spec-first を優先しており、quality gate 拡張も spec を先に揃える
- `docs/design-decisions/adr.md` の Go 採用判断は、保守性を仕様とテストで補強することを前提にしている
- `docs/specs/go-quality-gates.md` は `golangci-lint` 非導入、個別 tool pinning、`make lint` を唯一の lint 入口とする契約を固定している
- `docs/issues/go-exported-doc-comments-should-be-linted.md` は repo 全体へ適用できる comment checker を求めており、前段 plan `go-exported-doc-comments-should-be-linted.md` は最小 2 rule の `revive` 導入と `games/dungeon/**` での初回有効化を定義している
- 現状の `go-ci.yml` path filter は `.github/workflows/go-ci.yml`、`Makefile`、`go.mod`、`go.sum`、`**/*.go` のみを監視しており、`revive.toml` 単独変更を取りこぼす

## Scope

- `revive` の repo-wide 適用境界を `cmd/**`、`games/**`、`internal/**`、`e2e/**`、`testdata/**` に広げる
- `exported` と `package-comments` の 2 rule だけを維持したまま、各 package の不足 comment を補完する
- `make lint` と CI workflow の path trigger を repo-wide 運用に合わせて更新する
- `docs/specs/go-quality-gates.md` を final boundary に更新する

この plan では以下は扱わない。

- `golangci-lint` の導入
- `exported` / `package-comments` 以外の `revive` rule 追加
- suppressions 前提の恒久運用
- Go 以外の lint suite 拡張

## Design Decisions

この plan では追加 ADR は作らない。

- repo-wide 化の対象は、tracked な Go code が存在する `cmd/**`、`games/**`、`internal/**`、`e2e/**`、`testdata/**` とする
- rule surface は前段導入と同じ `exported` と `package-comments` に固定し、repo-wide 化と同時に別 rule を増やさない
- 既存 comment 未整備の解消単位は file 単位ではなく package 単位とする
  - 各 directory ごとに package comment と exported symbol comment をまとめて揃え、partial cleanup を残さない
- 実装順は 2 段階とする
  - 第1段: `cmd/**`、`internal/**`、`e2e/**` を repo-wide gate に載せる
  - 第2段: fixture-heavy な `testdata/**` を同じ 2 rule で通す
- `testdata/**` は永続除外しない
  - `fixturebot` のような helper package は通常 package と同じ基準で comment を揃える
  - `package main` fixture は package comment を追加し、exported symbol がある場合だけ exported comment を揃える
- CI trigger は lint 実行結果に影響する config file を source code と同等に扱い、`revive.toml` 単独変更でも Go CI が走るようにする

## Spec Changes

### `docs/specs/go-quality-gates.md`

以下を明記する。

- default `make lint` に含まれる `revive` の適用境界は `games/**` 限定ではなく repo-wide である
- 常設 rule は引き続き `exported` と `package-comments` の 2 つのみである
- `cmd/**`、`internal/**`、`e2e/**`、`testdata/**` の `package main` / helper package も同じ comment policy に含める
- `revive.toml` のような lint config 変更は Go CI を再実行すべき relevant change として扱う

この plan では spec 更新対象を `docs/specs/go-quality-gates.md` に限定する。ADR 追加や別 spec への契約変更は行わない。

## Expected Code Changes

### `revive.toml`

- repo-wide 適用に必要な path boundary を明文化する
- `exported` と `package-comments` 以外の rule は追加しない
- `cmd/**`、`internal/**`、`e2e/**`、`testdata/**` を含む最終対象境界を config 上でも明示する

### `Makefile`

- `make lint` の `revive` 実行対象が repo-wide になっていることを明確にする
- `revive.toml` を前提にした再現可能な invocation に揃える

### `.github/workflows/go-ci.yml`

- `revive.toml` 単独変更でも workflow が起動するよう `paths` を更新する
- 既存の `make lint` / `make test` 入口は変えず、repo-wide `revive` 化に追随する

### Go packages

- `cmd/**`: 各 command package の package comment と exported symbol comment を補う
- `internal/**`: platform/game package の exported API comment と package comment を directory 単位で揃える
- `e2e/**`: package comment が必要なら補う
- `testdata/**`: fixture helper package と fixture `main` package の comment 不足を directory 単位で揃える

## Sub-tasks

- [ ] Update `docs/specs/go-quality-gates.md` to declare repo-wide `revive` coverage and retain only `exported` + `package-comments`
- [ ] Update `revive.toml` so the final target boundary covers `cmd/**`, `games/**`, `internal/**`, `e2e/**`, and `testdata/**`
- [ ] Update `Makefile` wiring if needed so `make lint` runs the same repo-wide `revive` invocation locally and in CI
- [ ] Update `.github/workflows/go-ci.yml` path filters so `revive.toml` changes trigger Go CI
- [ ] Add missing package comments and exported comments in `cmd/**`, `internal/**`, and `e2e/**`
- [ ] [depends on: Add missing package comments and exported comments in `cmd/**`, `internal/**`, and `e2e/**`] Add missing package comments and exported comments in `testdata/**`
- [ ] Verify repo-wide `revive` passes under `make lint` without adding new rules or persistent exclusions

## Parallelism

- [parallel] `docs/specs/go-quality-gates.md` と `go-ci.yml` の boundary 更新は、final target boundary が固まれば並行で進められる
- [parallel] `cmd/**` と `internal/**` の comment 補完は、同じ rule set 前提で package 群ごとに分担できる
- `testdata/**` の comment 補完は、repo 本体 package の整理後に着手する
- Final verification depends on spec sync, CI trigger sync, and all in-scope package comments landing together

## Risks and Mitigations

- `internal/**` には platform core package が多く、repo-wide 有効化で想定以上に exported comment 不足が出る可能性がある
  - mitigation: package 単位で comment debt を潰し、directory 単位で完了判定する
- `testdata/**` には fixture `main` package が多く、package comment 追加の量が見積もりより増える可能性がある
  - mitigation: 第2段として切り分け、repo 本体 package の gate 拡張と fixture cleanup を混線させない
- `revive.toml` と workflow path filter の同期が漏れると、config 変更だけが CI 未検証で流れる
  - mitigation: execution PR で `revive.toml` と `.github/workflows/go-ci.yml` を同時に更新する
- rule を増やしながら repo-wide 化すると、comment rollout の作業量と policy change の原因切り分けが崩れる
  - mitigation: 今回は `exported` と `package-comments` 以外を追加しない

## Verification

The execution PR is complete when the following are true.

- `make lint` runs `revive` repo-wide and passes with only `exported` + `package-comments`
- `cmd/**`, `games/**`, `internal/**`, `e2e/**`, and `testdata/**` all satisfy the same comment policy
- `.github/workflows/go-ci.yml` runs for `revive.toml`-only changes
- `make test` still passes after the comment-only cleanup and lint wiring updates
- `docs/specs/go-quality-gates.md` matches the final lint suite, repo-wide boundary, and CI trigger contract
