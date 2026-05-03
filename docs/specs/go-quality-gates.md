# Go 品質ゲート

`ai-arena` の Go module は、ローカル開発と CI のどちらでも同じ quality-gate targets を入口として検証する。

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

## Cache Contract

- `Makefile` は local default として `/tmp/ai-arena-go-quality-gates` を cache root に使ってよい
- local default は `ww` で分かれた worktree 間でも再利用できる stable path として扱う
- `Makefile` は `GOPATH` / `GOMODCACHE` / `GOCACHE` を個別に override できなければならない
- ローカル開発では plain `make test` / `make fmt` / `make lint` が追加オプションなしで動かなければならない
- CI は workflow から `GOPATH` / `GOMODCACHE` / `GOCACHE` を上書きし、runner 標準の Go cache path を使ってよい
- CI の override 手段は workflow env または `make` の variable assignment でよい
- local default と CI override のどちらでも quality gate の入口は `make test` / `make fmt` / `make lint` に揃える
- GitHub Actions の Go cache strategy は `actions/setup-go` built-in cache と明示 cache を併用せず、1 系統に統一する
- GitHub Actions の cache entry は job ごとに分離してよい。`go-test` と `go-lint` は同じ dependency hash を共有しつつ、job suffix で最終 key を分けてよい

## CI Contract

- GitHub Actions 上の Go CI は `make test` と `make lint` を実行する
- `make test` と `make lint` は独立 job として並行に実行してよい
- CI は module/build/tool cache を持ってよいが、品質判定の入口は Makefile targets に揃える
- formatter drift は test failure ではなく lint failure として扱う

## Codex Hook Integration

- Codex の `PostToolUse` hook は `.go` edit の直後に `make fmt` を呼び出してよい
- Codex の `Stop` hook は turn 終了時に `make lint` と `make test` を呼び出してよい
- hook wiring と dispatch path は `docs/specs/codex-hooks.md` の契約に従う
