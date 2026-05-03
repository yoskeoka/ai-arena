# Go 品質ゲート

`ai-arena` の Go module は、ローカル開発と CI で同じ command surface を使って検証する。

## Command Surface

- `make test`
  - `go test ./...` を実行する
- `make fmt`
  - tracked `.go` files に `goimports` を適用し、formatting と import ordering を auto fix する
- `make lint`
  - formatter check と、最小限の Go lint suite を実行する

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

`golangci-lint` は導入せず、必要な checker を個別 tool として固定して使う。

## Tool Versioning

- `goimports`
- `noctx`
- `staticcheck`
- `gosec`

これらの tool version は module 側で明示的に pin し、CI とローカルで同じ version を使う。

## CI Contract

- GitHub Actions 上の Go CI は `make test` と `make lint` を実行する
- `make test` と `make lint` は独立 job として並行に実行してよい
- CI は module/tool cache を持ってよいが、品質判定の入口は Makefile targets に揃える
- formatter drift は test failure ではなく lint failure として扱う
