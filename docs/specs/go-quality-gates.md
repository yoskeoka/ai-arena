# Go 品質ゲート

`ai-arena` の Go module は、ローカル開発と CI のどちらでも同じ quality-gate targets を入口として検証する。

## Command Surface

- `make test`
  - fast default gate として `go test ./...` を実行する
- `make fmt`
  - tracked `.go` files に `goimports` を適用し、formatting と import ordering を auto fix する
- `make lint`
  - formatter check と、最小限の Go lint suite を実行する
- `make test-wasm-go`
  - dedicated Go-WASM verification lane として、Go-WASM `janken` e2e / helper tests を実行する
- `make test-wasm-rust`
  - dedicated Rust-WASM evaluation lane として、Rust-WASM `janken` e2e を実行する

## Formatter

- formatter は `goimports` を唯一の正規 formatter とする
- `make fmt` は tracked `.go` files のみを対象に auto fix する
- `make lint` は `goimports -l` 相当の check を行い、非空出力を lint failure として扱う
- CI は formatter を自動修正しない。修正が必要なファイル名を出力して fail する

## Lint Suite

`make lint` は少なくとも以下を順に実行する。

1. `goimports -l` over tracked `.go` files
2. `go vet ./...`
3. `go vet -vettool=<noctx> ./...`
4. `staticcheck ./...`
5. `gosec ./...`
6. `revive -config revive.toml ./...`

`golangci-lint` は導入せず、必要な checker を個別 tool として固定して使う。

`revive` は初回導入では comment policy の最小入口として使い、常設 rule は
`exported` と `package-comments` に限定する。repo-wide style ルールの一括導入は行わない。

`exported` rule は exported const / type / var / func / method の doc comment を要求する。
初回導入の対象は repo-external API / reusable library surface として扱う `games/**` 配下に絞る。

`package-comments` rule は library package に package comment を要求する。初回導入では
`games/**` 配下を対象とし、`cmd/**` の `package main` entrypoint、`internal/**` の repo-internal
package、`testdata/**` の fixture/helper package は対象から除外してよい。

## Tool Versioning

- `goimports`
- `noctx`
- `staticcheck`
- `gosec`
- `revive`

これらの tool version は module 側で明示的に pin し、CI とローカルで同じ version を使う。

## Cache Contract

- `Makefile` は local default として `/tmp/ai-arena-go-quality-gates` を cache root に使ってよい
- local default は `ww` で分かれた worktree 間でも再利用できる stable path として扱う
- `Makefile` は `GOPATH` / `GOMODCACHE` / `GOCACHE` を個別に override できなければならない
- ローカル開発では plain `make test` / `make fmt` / `make lint` が追加オプションなしで動かなければならない
- CI は workflow から `GOPATH` / `GOMODCACHE` / `GOCACHE` を上書きし、runner 標準の Go cache path を使ってよい
- CI の override 手段は workflow env または `make` の variable assignment でよい
- local default と CI override のどちらでも quality gate の入口は `make test` / `make fmt` / `make lint` に揃える
- GitHub Actions の Go cache strategy は `actions/setup-go` built-in cache と明示 cache を併用せず、1 系統に統一する
- GitHub Actions の明示 cache action は Node 24 runtime 対応版を使う。現行標準は `actions/cache@v5` とする
- GitHub Actions の cache entry は job ごとに分離してよい。`go-test` と `go-lint` は同じ dependency hash を共有しつつ、job suffix で最終 key を分けてよい

## CI Contract

- GitHub Actions 上の Go CI は `make test` と `make lint` を実行する
- `make test` と `make lint` は独立 job として並行に実行してよい
- CI は module/build/tool cache を持ってよいが、品質判定の入口は Makefile targets に揃える
- formatter drift は test failure ではなく lint failure として扱う
- default Go CI は Rust toolchain setup や WASM fixture build/e2e cost を持ち込まない
- WASM verification は dedicated workflow から `make test-wasm-go` / `make test-wasm-rust` を呼ぶ
- dedicated WASM workflow は `runtime` / `runner` / `e2e` / `testdata/ai/janken` / `Makefile` / workflow file など、WASM verification に影響する path 変更時だけ自動実行してよい

## Dedicated WASM Verification Lanes

Go 製 WASM sample build と `arena-runner` の `janken` verification は、default quality gate から分離し、
dedicated CI lane と manual helper に分けて維持する。

- `make test` には Go-WASM / Rust-WASM `janken` verification を含めない
- Go-WASM verification は supported path の targeted automated check として `make test-wasm-go` から継続実行する
- Rust-WASM verification は experiment / evaluation lane として `make test-wasm-rust` から CI 再現可能にするが、default Go gate には混ぜない
- WASM 専用 tests は default `go test ./...` から外れた dedicated selection mechanism で管理する。現行実装では dedicated env guard または build tag を使って分離してよい

### Manual Helpers

- `make build-janken-go-wasm`
  - `testdata/ai/janken/janken-go-wasm-ai` を `GOOS=wasip1 GOARCH=wasm` で build し、repo-local fixture path に `.wasm` を生成する
- `make run-janken-go-wasm`
  - 上記 build の後、`arena-runner` で `janken` match を起動し、subprocess bot と WASM bot が同じ game id で完走することを確認できる
- `make build-janken-rust-wasm`
  - `testdata/ai/janken/janken-rust-wasm-ai` を `wasm32-wasip1` target で build し、repo-local fixture path に `.wasm` を生成する
- `make run-janken-rust-wasm-eval`
  - 上記 build の後、`arena-runner` で `janken` match を起動し、Rust-WASM bot の evaluation lane を再現できる

方針:

- 常設 gate は引き続き `make test` / `make lint`
- Go-WASM path の継続的検証は dedicated CI lane と manual helper の両輪で担保する
- Rust-WASM path は `experiment-only` lane として dedicated CI lane / helper に留める
- default gate へ昇格させるのは、runtime matrix と CI cost を別途評価してからとする

## Codex Hook Integration

- Codex の `PostToolUse` hook は `.go` edit の直後に `make fmt` を呼び出してよい
- Codex の `Stop` hook は turn 終了時に `make lint` と `make test` を呼び出してよい
- hook wiring と dispatch path は `docs/specs/codex-hooks.md` の契約に従う
