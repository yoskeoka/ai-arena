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
6. `revive -config revive.toml ./cmd/... ./games/... ./internal/... ./e2e/... <explicit testdata package dirs>`

`golangci-lint` は導入せず、必要な checker を個別 tool として固定して使う。

`revive` は comment policy の最小入口として使い、常設 rule は
`exported` と `package-comments` に限定する。`exported` rule の stuttering subcheck のような
rename/style 指摘は今回の常設 gate に含めない。repo-wide style ルールの一括導入は行わない。

`exported` rule は exported const / type / var / func / method の doc comment を要求する。
対象は tracked な Go code が存在する `cmd/**`、`games/**`、`internal/**`、`e2e/**`、`testdata/**`
全体とする。

`package-comments` rule は package ごとに package comment を要求する。`package main`
entrypoint、repo-internal package、fixture/helper package も同じ comment policy に含める。
`testdata/**` を最終対象に含めるため、`make lint` の `revive` invocation は `./...` だけに依存せず、
`./cmd/... ./games/... ./internal/... ./e2e/...` に加えて、`testdata/**` と
`internal/platform/runtime/testdata/**` の package dir を明示的に検査しなければならない。

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
