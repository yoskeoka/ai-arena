# go-ci-cache-tuning
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の Go quality gates で、ローカル開発時の worktree 横断 cache reuse は維持しつつ、GitHub Actions では CI runner に自然な cache layout を使うように整理する。

この plan は以下を成立させる。

- ローカル開発では `ww` で新しい worktree を切っても、`make test` / `make lint` / `make fmt` が同じ `/tmp` cache を再利用できる
- CI では `Makefile` のローカル向け `/tmp` path に縛られず、runner 標準の Go cache path を使える
- CI cache は `go-test` と `go-lint` で競合せず、それぞれの job が必要な module/build/tool state を再利用できる
- `docs/specs/go-quality-gates.md` に local/CI cache contract を明記し、quality gate の入口は引き続き `make` targets に揃える

## Context

- `docs/project-plan.md` は Go を platform 実装言語として採用しており、Phase 2 ではローカル反復速度を維持したまま platform core を積み上げる必要がある
- `docs/design-decisions/core-beliefs.md` は spec-first と correctness を優先しているため、cache 挙動の変更も先に spec へ反映する
- `docs/design-decisions/adr.md` の Go 採用判断は、単一実装系での継続的な検証容易性を重視している
- `docs/exec-plan/done/go-ci-make-targets.md` で `make test` / `make lint` / `make fmt` と Go CI の契約は既に導入済み
- 現状の `.github/workflows/go-ci.yml` は `go-test` と `go-lint` が同じ cache key を共有し、`Makefile` は `/tmp/ai-arena-go-quality-gates` を CI でも強制しているため、lint 側 tool cache が十分に育たない

## Scope

- `Makefile` の cache path override 設計見直し
- `docs/specs/go-quality-gates.md` の local/CI cache contract 更新
- `.github/workflows/go-ci.yml` の Go cache 設計見直し
- local と CI の command surface 一貫性維持

この plan では以下は扱わない。

- lint suite 自体の追加・削除
- `golangci-lint` 導入
- `go-test` と `go-lint` の job 統合
- nightly security scan など別 workflow への分離

## Design Decision

この plan では追加 ADR は作らない。

- ローカル既定値として `/tmp/ai-arena-go-quality-gates` を維持する
- CI では workflow から `GOPATH` / `GOMODCACHE` / `GOCACHE` を runner 標準 path に上書きして使う
- CI cache は job ごとに独立させ、`go-test` と `go-lint` の cache 競合を避ける

これは新しいアーキテクチャ方針ではなく、既存の Go quality-gate contract の運用調整として扱う。

## Spec Changes

### `docs/specs/go-quality-gates.md`

以下を明記する。

- `Makefile` の cache path は local default を持ってよい
- local default は worktree をまたいで再利用できる stable path として `/tmp/ai-arena-go-quality-gates` を使ってよい
- CI は workflow から `GOPATH` / `GOMODCACHE` / `GOCACHE` を上書きし、runner 標準 path を使ってよい
- CI cache は `make` target の behavioral contract を変えない範囲で job ごとに分離してよい
- `actions/setup-go` built-in cache と明示 cache の責務が衝突しないよう、CI cache strategy は一貫した 1 系統に揃える

## Expected Code Changes

### `Makefile`

- `CACHE_ROOT` だけでなく `GOPATH` / `GOMODCACHE` / `GOCACHE` も workflow から override できる形にする
- local default は現状どおり `/tmp/ai-arena-go-quality-gates` を維持する
- `make test` / `make fmt` / `make lint` の command surface は変えない

### `.github/workflows/go-ci.yml`

- CI では runner 標準の Go cache path を使うよう `make` 実行環境を明示する
- `go-test` と `go-lint` が cache key を共有しないようにする
- `setup-go` built-in cache を使うか、明示 `actions/cache` に寄せるかを 1 つに統一する
- `make test` / `make lint` の入口は維持する

### `docs/specs/go-quality-gates.md`

- local cache と CI cache の役割分担を追記する

## Sub-tasks

- [ ] Update `docs/specs/go-quality-gates.md` with the local-versus-CI cache contract
- [ ] Update `Makefile` so cache-related Go environment variables can be overridden while preserving the local `/tmp` default
- [ ] Update `.github/workflows/go-ci.yml` so CI uses runner-native Go cache paths
- [ ] Split CI cache ownership so `go-test` and `go-lint` do not compete for the same cache entry
- [ ] Verify `make test` and `make lint` still work locally with the default `/tmp` cache
- [ ] Verify the workflow definition still keeps `make` as the only quality-gate entrypoint

## Parallelism

- [parallel] Spec wording and workflow YAML drafting can proceed independently once the override policy is fixed
- [parallel] `Makefile` override support and CI cache-key separation can proceed independently
- Workflow verification depends on both the `Makefile` update and the final CI cache strategy choice

## Risks and Mitigations

- CI cache strategy may become harder to understand if local and CI path rules are mixed implicitly
  - mitigation: make the override boundary explicit in both spec and workflow env configuration
- If `Makefile` keeps hard-coded derived variables, workflow overrides may appear to work while still writing to `/tmp`
  - mitigation: make `GOPATH`, `GOMODCACHE`, and `GOCACHE` individually overrideable and verify them in execution
- Cache-key separation can reduce sharing too much if keyed too narrowly
  - mitigation: keep the base dependency hash shared, but add a job-role suffix rather than unrelated content

## Verification

The execution PR is complete when the following are true.

- Local `make test` succeeds with the default `/tmp/ai-arena-go-quality-gates` layout
- Local `make lint` succeeds with the default `/tmp/ai-arena-go-quality-gates` layout
- CI workflow definition clearly routes Go cache paths through runner-native locations instead of the local `/tmp` default
- CI workflow definition no longer makes `go-test` and `go-lint` share the same cache entry
