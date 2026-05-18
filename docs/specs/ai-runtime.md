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

## Support Policy

WASM/WASI runtime contract 自体は言語非依存で共通とする。一方で、公式の guide・sample・verification assets
を整備して継続的に動作保証する範囲は、toolchain ごとに段階的に管理する。

Phase 4 時点の区分:

- `supported`: Go
- `experiment-only`: Rust
- `future candidate`: TypeScript, Python

区分基準:

- `supported`
  - repo 内に公式 sample、build helper、targeted verification path がある
  - `janken` で再現可能な verification asset があり、継続確認の責務を負う
- `experiment-only`
  - WASM/WASI contract に沿う module の提出自体は許可する
  - repo 内に最小評価用 sample または再現可能 artifact を置いてよい
  - build/runtime 成否の観測は残すが、常設 gate や外部向け guide 整備までは約束しない
- `future candidate`
  - support 拡張候補として意図だけ残す
  - sample、helper、verification asset はまだ持たない

運用ルール:

- `supported` 以外を、公式サポート済み言語として告知してはならない
- `experiment-only` の評価結果は恒久 spec へ詳細に埋め込まず、必要に応じて appendix / issue note / PR artifact に逃がしてよい
- runtime host 側は language ごとの差別扱いをせず、manifest と module の contract だけを正本として扱う

## Go 参照フロー

Phase 4 の first supported reference path は Go source から `GOOS=wasip1 GOARCH=wasm` で build した
WASM/WASI module とする。`janken` sample AI はこの参照フローを固定するための正本 fixture であり、
binary artifact ではなく source + reproducible build step を正本として扱う。

参照 build 例:

```sh
GOOS=wasip1 GOARCH=wasm go build \
  -o ./testdata/ai/janken/janken-go-wasm-ai.wasm \
  ./testdata/ai/janken/janken-go-wasm-ai
```

対応する sidecar manifest 例:

```json
{
  "ai_id": "janken-go-wasm-ai",
  "protocol": {
    "transport": "stdio-jsonrpc-ndjson",
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular"
  },
  "runtime": {
    "kind": "wasm-wasi",
    "module": "./janken-go-wasm-ai.wasm",
    "args": ["./janken-go-wasm-ai.wasm"],
    "memory_limit_pages": 64
  }
}
```

運用ルール:

- checked-in fixture の正本は `.go` source と `.arena.json` manifest であり、`.wasm` binary は commit しない
- local helper / targeted verification / CI は必要に応じて `.wasm` を都度 build して使う
- sidecar manifest は build output と同じ directory に置き、`runtime.module` は sidecar 基準で解決できる相対 path を使う
- e2e helper / CI helper は caller から AI player entry 名だけを受け取り、build 済み module を指す temp sidecar を生成して runtime kind 差分を隠蔽してよい

Go sample で期待する runtime 振る舞い:

- `stdout`: `init` / `turn` / `game_over` への JSON-RPC response だけを NDJSON で返す
- `stderr`: `janken-go-wasm-ai init`, `turn <n>`, `game_over` のような debug/audit 用ログを出してよい
- exit/shutdown: `game_over` に ACK 後は clean exit してよく、platform が `stdin` close 後に cooperative shutdown を完了できること

## Game-Owned Additional Lanes

`ai-arena` が継続して canonical asset を持つのは platform fixture と runtime contract の検証 lane までとする。
repo 外へ切り出した game が Go-WASM や独自 fixture lane を持つ場合、その source / manifest / golden / CI は
game 開発側 repo が ownership を持つ。

その場合でも守る契約:

- checked-in の正本は `.go` source と `.arena.json` manifest とし、`.wasm` binary は commit しない
- local subprocess bot と WASM bot が shared decision layer を共有する場合、transport / runtime 差分は entrypoint 側へ閉じ込める
- runtime ごとの entrypoint や test helper が `full_state` や hidden information 全体を shared policy へ直接渡してはならない
- runner host を versioned dependency として使う場合、game 開発側 repo が採用する ai-arena version を明示して verification を固定する

## Rust Evaluation Lane

Rust は Phase 4 時点では `experiment-only` とする。最初の non-Go candidate として、
`janken` で最小の WASM/WASI evaluation lane を持ってよい。

前提:

- module は command-style WASI program として起動できること
- `stdout` / `stderr` / shutdown contract は Go 参照フローと同じであること
- toolchain 前提や blocker 切り分けは helper / appendix / issue note 側へ残し、外部 developer guide には昇格しないこと

この lane が意味するもの:

- Rust module の提出可否を runtime contract 上で評価できる
- `janken` の targeted verification path で build/runtime 観測を再現できる
- ただし official support は Go のみであり、Rust の成功は将来サポート判断の材料に留まる

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
