# Go 1.27 toolchain
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

`ai-arena` をビルド、テスト、lint、Go-WASM fixture のすべてで Go 1.27 系として扱い、
module の言語バージョン下限と、main module を操作するときに選ばれる Go toolchain を同じ
1.27 系のリリース `go1.27.1` に固定する。ローカルと GitHub Actions は既存どおり
`go.mod` を唯一の Go version input とし、toolchain 自動選択が有効な環境では 1.27.1 を選ぶ。

完了時には `go.mod` が `go 1.27` と `toolchain go1.27.1` を宣言し、開発用 quality-gate
契約がその役割（前者は最低言語/実行要件、後者は推奨する exact toolchain）を明記する。
file-backed、Postgres、lint、Go-WASM、Rust-WASM の既存検証が選択された toolchain で成功し、
module graph に意図しない差分を残さず、その実施結果から将来の Go version upgrade 用 skill を
作成することまでを本計画の範囲とする。

次は範囲外とする。

- Go 1.27 の新 API / 言語機能を利用するためのアプリケーション変更
- Go tool・通常依存の version 更新（`go get` / `tidy` が必要最小限に生む module metadata 以外）
- cache key 形式、GitHub Actions action revision、CI job topology、Render build image の変更
- Go 1.27.x の後続 patch への追随
- 実施前に Go version upgrade skill の詳細手順・文面を設計すること。skill は実装で得た
  確定済みの手順と検証結果から組み立てる

## 背景と既存根拠

- `go.mod:1-9` は現在 `go 1.26` のみを宣言し、`toolchain` directive がない。Go の `go`
  directive は main module を使う最低 Go version と言語 version を定め、`toolchain`
  directive は default toolchain より新しい場合に選ぶ exact toolchain を提案する。
- Go 公式配布 API（2026-09-13 確認）は stable `go1.27.1` を提供する。この初期 patch
  release を固定値にする。
- `.github/workflows/go-ci.yml:31-123`、`.github/workflows/wasm-verification.yml:34-80`、
  `.github/workflows/operator-ui-browser.yml:50-218` はいずれも `actions/setup-go` の
  `go-version-file: go.mod` を使う。このため workflow ごとに version literal を重複させず、
  module manifest を更新すれば既存 lane の version source は一貫する。
- `docs/development/go-quality-gates.md:1-18,72-104` は local/CI 共通 quality-gate、tool
  pin、cache/CI の運用契約を所有する。`docs/specs/` は local harness/CI mechanics を
  所有しないため、この toolchain 運用契約は開発文書に置く。
- `docs/specs/ai-runtime.md:136-153` も language-specific build 手順と CI verification
  lane を development harness 文書側で扱う境界を明記している。

## 仕様・契約変更

### 開発 toolchain contract

`docs/development/go-quality-gates.md` に次を追加する。

- `go.mod` の `go` directive は module の最低 Go version と package の言語 version を
  `1.27` に固定する。
- `go.mod` の `toolchain` directive は local と CI が main module を操作するときの
  preferred exact toolchain を `go1.27.1` に固定する。`GOTOOLCHAIN` で switching を禁じる
  環境では、実行者がこの version を事前に提供しなければならない。
- GitHub Actions とローカルは別個の version literal を持たず、既存の `go-version-file:
  go.mod` と Go command の選択規則を通じて同じ manifest を source of truth とする。

### Black-box product/API specification

N/A。これは public API、runtime input/output、artifact、ゲーム規則を変えない開発・CI の
build contract 変更である。`docs/specs/` に CI mechanics を追加しない。

## 変更マップ

- `(MODIFY) docs/development/go-quality-gates.md:1-18,72-104`:
  Go version と exact toolchain の責務、single source of truth、`GOTOOLCHAIN` を固定した
  環境での前提を明文化する。既存 command / cache / CI lane の契約は変更しない。
- `(MODIFY) go.mod:1-9`:
  `go 1.26` を `go 1.27` に上げ、その直後に `toolchain go1.27.1` を追加する。既存
  `tool` block と dependency version は意図して変更しない。
- `(MODIFY, only if Go tooling changes it) go.sum`:
  Go 1.27.1 で module tidy / verification が正当な checksum 差分を生成した場合だけ、その
  最小差分を含める。差分が不要なら変更しない。
- `(NEW) .claude/skills/go-version-upgrade/SKILL.md`:
  実施済みの Go version/toolchain upgrade を再利用可能にする repo-local skill を追加する。
  内容は事前に固定せず、directive 更新、module metadata の精査、toolchain selection、quality
  gates、CI evidence について実装で確認できた手順と境界だけを記録する。
- `(NO CHANGE, verify) .github/workflows/go-ci.yml:31-123`,
  `.github/workflows/wasm-verification.yml:34-80`,
  `.github/workflows/operator-ui-browser.yml:50-218`:
  `go-version-file: go.mod` が更新後 manifest を読んで selected Go version を揃えることを
  CI 実行で確認する。version literal を追加しない。

## 実施手順

1. Go 公式 toolchain documentation と release API を実行時点で再確認し、stable 1.27.x が
   `go1.27.1` であることを確認する。より新しい patch を採用する必要が生じた場合は、plan
   本文・開発 contract・検証期待値を同じ exact version に更新してから進める。
2. 先に `docs/development/go-quality-gates.md` を更新し、上記「開発 toolchain contract」を
   記録する。これは implementation より先に運用上観測可能な build contract を固定する手順
   である。
3. Go が提供する module-aware 操作（例: `go get go@1.27 toolchain@1.27.1`）で `go.mod` を
   更新し、`go 1.27` と `toolchain go1.27.1` の組を得る。手編集で directive syntax を推測
   しない。続けて `go mod tidy` を Go 1.27.1 で実行し、dependency graph / checksum が整合
   するか確認する。
4. `git diff -- go.mod go.sum` を精査する。directive 変更と tidy が必須とした最小 metadata
   以外の dependency/tool update は本計画外なので revert して原因を分離する。workspace の
   `go.work` が存在しないことも確認し、別の workspace directive が main module の選択を
   上書きしていない状態で検証する。
5. selected toolchain を明示して確認する。通常の `go version` が `go1.27.1` を返し、
   `go env GOTOOLCHAIN` と `go env GOMOD` が意図しない固定設定や別 module を示さないことを
   記録する。必要なら `GODEBUG=toolchaintrace=1` を使い、toolchain selection の根拠を診断する。
6. 既存 Makefile entrypoint を Go 1.27.1 で実行する。順序は module/checksum 確認、
   file-backed test、Postgres test、lint、Go-WASM、Rust-WASM とし、失敗時は Go 1.27 由来の
   compile/test/tool incompatibility を最小の scope で直す。既存 Go tool pin の更新が必要なら、
   その互換性修正だけを同じ PR に含め、理由と version を開発 contract に追記する。
7. 上記の実装と local verification が確定した後に `.claude/skills/go-version-upgrade/SKILL.md`
   を作成する。skill の詳細内容はこの時点で初めて、実際に成功した command、必要だった
   prerequisite、許容される module metadata 差分、local/CI の検証証跡から構成する。計画段階の
   仮説を手順として書かず、今回の upgrade 以外の依存更新・CI topology 変更を skill の標準手順
   に含めない。
8. GitHub Actions では既存の `go-version-file: go.mod` lanes を変更せず、PR の `go-ci`、
   `wasm-verification`、`operator-ui-browser` の最新 head 実行が Go 1.27.1 で開始し成功することを
   job log で確認する。setup-go が 1.27.1 を準備しない、または Go command が別 version を選ぶ
   なら、version source を増やさず `go.mod` directive と action の version-file 解釈に限定して
   原因を解消する。

## 依存関係と並行性

| 順序 | 作業 | 依存 | 並行可否 |
| --- | --- | --- | --- |
| 1 | stable exact release の再確認と development contract 更新 | なし | 不可 |
| 2 | `go` / `toolchain` directives と module metadata 更新 | 1 | 不可 |
| 3 | selected toolchain と module diff の検査 | 2 | 不可 |
| 4 | local quality gates | 3 | `make test`、`make lint`、WASM lanes は cache/DB 資源を分離できる場合のみ並行可 |
| 5 | 実施結果から Go version upgrade skill を作成 | 4 | 不可 |
| 6 | PR CI と latest-head follow-up | 5、PR 作成 | CI jobs は並行、follow-up は順次 |

## 検証

- `go.mod` に `go 1.27` と `toolchain go1.27.1` が 1 行ずつあり、dependency と `tool` の
  version diff が意図したものだけであることを確認する。
- `go version` が selected `go1.27.1` を返すこと、`go env GOMOD` が repository の `go.mod` を
  返すことを確認する。toolchain download を禁止する CI/local 設定でも必要な toolchain を
  供給すれば同じ version で動くことを確認する。
- Go 1.27.1 で `go mod tidy` 後に `git diff --exit-code -- go.mod go.sum` が、承認済みの
  directive/checksum 差分以外を示さないことを確認する。
- `make test`、`make test-postgres`、`make lint`、`make test-wasm-go`、`make test-wasm-rust`
  を実行する。Postgres lane は既存の repository 手順で DSN と service を用意して実行する。
- `.claude/skills/go-version-upgrade/SKILL.md` が追加され、今回の成功した upgrade の実施内容と
  evidence を再利用できる一方、未実施の手順を事実として扱わないことをレビューで確認する。
- PR では `go-ci`、`wasm-verification`、`operator-ui-browser` の current head を確認し、各
  setup-go step と Go command が manifest 由来の 1.27.1 を使っていること、および required
  checks が成功していることを確認する。

## 実行時の判断

- `go 1.27` は module の最低要件/言語 version、`toolchain go1.27.1` は exact preferred
  release という分離を採る。`go 1.27.1` のみで patch を最低要件として強制する案は採らない。
- workflow ごとに `go-version: 1.27.1` を書く案は採らない。既存の `go-version-file: go.mod`
  が version input を一元化しており、version literal の重複は将来の patch update を不整合にする。
- Go 1.27.1 が古くなっている場合にも実装時に無断で latest patch へ切り替えない。exact
  release を変えることは reviewed plan の contract 変更なので、plan を更新してレビューを
  受ける。
