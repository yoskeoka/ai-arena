# AI Runtime 仕様

## 目的

このドキュメントは、AI player runtime の公式 contract を定義する。platform / runner / AI sample は
この spec を正本として、runtime kind、manifest schema、標準 stream 契約、sandbox と resource limit
の境界を共有する。

## この spec の責務範囲

この spec が定義するもの:

- AI sidecar manifest の runtime schema
- `local-subprocess` / `wasm-wasi` runtime kind
- runtime kind ごとの entrypoint 解決
- `stdin` / `stdout` / `stderr` の使い方
- sandbox と host capability の最小契約
- deadline / memory limit / shutdown の責務分離
- runtime failure の監査観点

この spec が定義しないもの:

- game 固有 payload schema
- game metadata 互換性判定の詳細
- `arena-runner` の artifact layout
- 提出 API や online service への upload 手順

## Sidecar Manifest

AI 実行物の横に `<entry>.arena.json` を置く。

online service skeleton の admission validation が local artifact locator を受ける場合も、sidecar 発見規則はこの
`<entry>.arena.json` を使う。validation 段階では manifest / runtime の互換性確認と entrypoint 解決だけを行い、
実際の match 実行開始は queue claim 後の worker / runner 責務に留める。

最小 schema:

```json
{
  "ai_id": "sample-janken-bot",
  "protocol": {
    "transport": "stdio-jsonrpc-ndjson",
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular"
  },
  "runtime": {
    "kind": "wasm-wasi",
    "module": "./bot.wasm",
    "args": ["./bot.wasm"],
    "memory_limit_pages": 64
  }
}
```

### 共通項目

- `ai_id`: AI の識別子。省略時は runner が entrypoint basename などの local fallback を使ってよい
- `protocol.transport`: 現時点では `stdio-jsonrpc-ndjson` のみを許可する
- `protocol.game_id`
- `protocol.game_version`
- `protocol.ruleset_version`
- `runtime.kind`: 起動方式の識別子

### `local-subprocess`

`local-subprocess` は開発用・比較用の runtime kind として維持する。

必須項目:

- `runtime.command`: shell 展開しない tokenized command array

任意項目:

- `runtime.args`: 将来拡張予約とし、現時点では使わない
- `runtime.memory_limit_pages`: 使用しない

運用補足:

- contributor example では `go run` entry を使ってよい
- ただし deploy/release lane では、`local-subprocess` participant が build-time に用意された executable を
  `runtime.command` で指してよい
- その prepared executable は、repo source checkout や runtime toolchain download がなくても
  `make render-build` のような build contract だけで再生成できなければならない

### `wasm-wasi`

`wasm-wasi` は正式な WASM/WASI 実行経路である。

必須項目:

- `runtime.module`: `.wasm` module への path

任意項目:

- `runtime.args`: module に渡す argv。省略時は `[runtime.module]` とみなしてよい
- `runtime.memory_limit_pages`: linear memory max pages。未指定時は platform 既定値を使ってよい

制約:

- module は command-style WASI program として起動できなければならない
- module path は sidecar path 基準または runner cwd 基準のローカル file path とする
- network や repo/workspace への暗黙 access を前提にしてはならない

## 標準 Stream 契約

- `stdin`: platform から AI への JSON-RPC request / notification を NDJSON で送る唯一の公式入力
- `stdout`: AI から platform への JSON-RPC response を NDJSON で返す唯一の公式出力
- `stderr`: AI の自由ログ出力。platform は capture するが protocol channel として解釈しない

不変条件:

- runtime kind が subprocess でも WASM でも `stdout` へ JSON 以外を混在させてはならない
- runtime kind が subprocess でも WASM でも `stderr` は debug / audit 用の自由ログとして扱う
- transport 継続不能は runtime kind を問わず `runtime-stopped` として監査する

## Sandbox と Host Capability

deny-by-default を基本方針とする。

- network access は与えない
- repo/workspace への暗黙 file access は与えない
- `wasm-wasi` では最小限の stdio と clock/time 相当だけを前提にしてよい
- 任意 host function や環境依存 capability を AI contract に含めてはならない

`local-subprocess` は OS process で動いても、platform contract 上は上記制約を満たす運用前提とする。

## Resource Limit と Deadline

- request deadline は session/request 側の責務であり、各 `init` / `turn` / `game_over` request の締切として扱う
- request deadline は request kind ごとに独立でよく、`init` / `turn` / `game_over` を同一値へ固定しない
- 競技上の厳しい締切は通常 `turn` に置き、`init` は一度きりの起動コストを吸収するためより大きい上限を持ってよい
- runtime 側は memory 上限と host capability 制限を担う
- runtime shutdown は platform が主導し、cooperative shutdown を試みた後に必要なら強制停止してよい

`wasm-wasi` では少なくとも以下を platform が管理する。

- module instantiation 成否
- memory limit の適用
- cooperative shutdown 後の forced shutdown

## Language-Specific Guides and Verification

この spec は、runtime kind、manifest、stream、sandbox、resource limit の共通契約だけを定義する。

以下はこの spec の責務外とする。

- 特定言語向けの build 手順
- sample AI の配置
- CI verification lane の構成
- fixture / helper / golden の ownership
- language support 拡張の評価メモ

それらの development harness 文書は `docs/development/` や issue / plan / PR artifact 側で扱う。

## 監査対象

platform は少なくとも以下を distinguish して記録できなければならない。

- runtime 起動失敗
- malformed response
- timeout
- forced shutdown
- transport 継続不能

失敗分類との対応:

- deadline まで response が来なければ `invalid-timeout`
- deadline 前に stdout close / module exit / stdin write failure など transport 継続不能が起きたら `runtime-stopped`
- JSON 破損、JSON-RPC envelope 不正、error response は `invalid-protocol-malformed`
- shutdown 猶予後に platform が停止を強制した場合は audit 上 forced shutdown として残してよい

## Security Checklist

`wasm-wasi` を読み込む host 実装と review では、少なくとも以下を確認する。

- [ ] host capability は deny-by-default で、未使用の host function / filesystem / network access を渡していない
- [ ] `stdin` / `stdout` / `stderr` 以外の通信経路を AI runtime contract に含めていない
- [ ] memory limit は未設定時も platform default を適用し、unbounded growth を許していない
- [ ] request deadline と shutdown timeout は host 側で強制できる
- [ ] transport 継続不能、timeout、malformed output、forced shutdown を区別して監査できる
- [ ] module path / artifact path は host 側で明示解決し、repo/workspace 全体 access を暗黙許可していない
- [ ] 追加の host capability を導入する場合、その capability が必要な理由と影響範囲を spec か ADR に残す

この checklist は `wasm-wasi` だけでなく、将来 custom host function や限定的 filesystem access を足すときの review gate として使う。
