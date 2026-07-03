# Architectural Decision Records (ADR)

## [2026-03-07] AI実行環境: WASM採用、Docker却下

### Context
AIプログラムの提出・実行形式を決定する必要がある。候補はDocker（コンテナ）とWASM（WebAssembly/WASI）。

project-planの要件:
- AIプログラムの実装言語自由
- プラットフォーム側でのコンパイル・ビルドなし
- 外部通信禁止（不正防止）
- stdin/stdoutでの通信
- 高速起動（ターン制ゲーム）

### Decision
**WASM (WASI + wazero) を採用。Dockerは却下。**

### Consequences

**WASM採用の利点:**
- サンドボックスがデフォルト。外部通信禁止が設計上保証される（Dockerはiptables等で自前構築が必要）
- 起動がミリ秒単位（Dockerは秒単位）
- wazero（pure Go）でプラットフォームにネイティブ統合。外部依存なし
- クロスプラットフォーム。提出者がOS/アーキテクチャを意識不要
- stdin/stdoutがWASI標準でサポート済み

**Docker却下の理由:**
- セキュリティ確保が実質「小さなPaaS構築」になり、開発コストが高い
- ネットワーク隔離を自前で構築・検証する必要がある（WASMはデフォルトで不可能）
- Dockerデーモン依存はデプロイ環境の制約になる
- 起動オーバーヘッドがターン制ゲームに不向き
- WASMでカバーできない言語（Python等）のために残す案もあったが、WASMコンパイル可能な言語で十分なカバレッジがあり、複雑さに見合わない

**トレードオフ:**
- WASMにコンパイルできない言語（Python, Java等）は非対応。ただしAI競技で高速判断を求めるコンテキストでは、これらの言語はそもそも不利
- AIプログラムからの外部通信が一切不可能になるため、プラットフォーム側でstderr捕捉・ログ提供が必要（デバッグだけでなく、AI改善サイクル支援のため）

---

## [2026-03-08] Locale・言語・Timezone: 英語 + UTC

### Context
プラットフォームのUI言語、日時表示のタイムゾーン、ログ・API応答のロケールを統一する必要がある。
ai-arenaはAI競技プラットフォームであり、国際的な参加者を想定。運営コストを最小化しつつ、参加者にわかりやすいUXを提供したい。

### Decision
- **UI言語**: 英語のみ（多言語対応なし）
- **Timezone**: UTC固定（サーバーサイド・クライアント表示ともにUTC）
- **Locale**: en-US

### Consequences

**利点:**
- 実装が単純。i18nフレームワーク不要
- AI競技プラットフォームは技術者向けであり、英語+UTCで十分な共通言語
- 対戦ログ・ランキングの日時がUTC統一で混乱なし
- サーバーサイドのタイムゾーン変換処理が不要

**トレードオフ:**
- 英語を読めない参加者にはハードルが高い。ただしAIプログラムを書ける人は英語の技術文書を読めることが多い
- UTC表示はローカルタイムに慣れたユーザーには不便だが、対戦スケジュール等で曖昧さがなくなる利点が上回る

---

## [2026-04-26] AI通信プロトコル: stdin/stdout + JSON-RPC 2.0

### Context
AI Arena はゲーム非依存の実行基盤を目指しており、各ゲームで異なるのは
状態とアクションの中身だけにしたい。AI との通信は以下を満たす必要がある。

- 言語非依存であること
- ターンごとの request/response 対応が明確であること
- 長期プロセスで複数ターンを安全に扱えること
- 将来のゲーム追加時にも transport を変えずに済むこと

### Decision
**AI との公式通信は stdin/stdout 上の JSON-RPC 2.0 に統一する。**

### Consequences

**利点:**
- ほぼ全言語で実装可能。特定 SDK を強制しない
- `id` により各ターンの request/response 対応が明確
- `method` と `params` を分けることで、共通 transport とゲーム固有 payload を分離できる
- ローカルプロセス実行から WASM 実行へ移行しても、AI 側の通信契約を維持しやすい
- NDJSON framing を採用しやすく、人間がローカルデバッグ時に入出力を追いやすい

**却下した代替案:**
- 独自の行指向テキストプロトコル: 最初は軽いが、ゲーム追加時に拡張性と検証性が落ちる
- HTTP/gRPC: サンドボックス内 AI との接続モデルとして重く、短いターン制ゲームには過剰

**トレードオフ:**
- JSON シリアライズコストはあるが、Phase 1-3 のスコープでは支配的コストではない
- stdout を JSON-RPC 専用に縛る必要があるため、デバッグログは stderr に分離する必要がある

---

## [2026-04-26] プラットフォーム実装言語: Go

### Context
プラットフォームはマッチ運営、AI 実行管理、ログ収集、将来の API 提供まで担う。
WASM ランタイムである wazero を自然に統合でき、Phase 2 で小さく実装を始められる
実装言語が必要だった。

### Decision
**プラットフォーム実装言語として Go を採用する。**

### Consequences

**利点:**
- wazero が pure Go で提供されており統合が自然
- 単一バイナリで配布しやすく、開発・運用が単純
- goroutine と context により、複数 AI のターン締切管理を実装しやすい
- 将来の API サーバーや運営ツールも同じ言語で揃えやすい

**却下した代替案:**
- Rust: 性能面は優秀だが、Phase 2 で最小実装を立ち上げる速度よりも実装負荷が高い
- Node.js: JSON 処理はしやすいが、WASM 実行基盤との親和性と運用一貫性で Go に劣る

**トレードオフ:**
- 厳密な型表現では Rust より弱い場面があるため、仕様とテストで補強する必要がある

---

## [2026-05-27] Phase 6 first infra target は Pages + Render + Neon + R2 とする

### Context
Phase 6 の `0056` では、online service skeleton の durable write model を最初に成立させたい。一方で、
`world state`、`snapshot`、`history`、stderr、AI/game master executable のような artifact 本体は、
queue や registration manifest metadata より大きく、public watch/read API からも参照されうる。

この段階で必要なのは、

- queue / registration / locator metadata を durable に共有すること
- latest world state を必要なら即時 read しやすい場所へ置けること
- large artifact を DB 本体から切り離すこと
- first deploy を低コストかつ簡潔に始められること

であり、multi-node fairness や distributed queue 自体はまだ対象外である。

また、local CLI、CI、external game 開発で runner を直接起動する lane は引き続き file-backed first で運用したいが、
online service infra では同じ artifact contract を remote object storage へ差し替えたい。

### Decision
Phase 6 の first infra target は次で固定する。

- frontend / public watch UI: `Cloudflare Pages`
- match 実行を担う backend process: `Render`
- durable metadata backend: `Neon Postgres`
- large artifact backend: `Cloudflare R2`

保存責務は次で分離する。

- `Neon Postgres`: queue lifecycle、registration manifest metadata、artifact locator metadata、必要なら latest world state
- `Cloudflare R2`: snapshot、history、stderr、large world-state artifact、AI/game master executable payload

artifact contract の default lane は file-backed first とする。local CLI、CI、external game 開発では local filesystem を使い、
online service infra では同じ artifact shape を `R2` へ差し替える。

public read/watch API は当初 backend process と同じ lane に置いてよいが、後続で `Cloudflare Workers` へ切り出しても
この storage split は変えない。

### Consequences

**利点:**
- `Neon` は DB 単体を軽く持てるため、queue や registration の coordination を素直に扱える
- `R2` は large artifact と watch/read の object delivery を DB から切り離せる
- `Render` は first backend process を分かりやすく常駐させやすく、match 実行 lane の運用を単純に始められる
- file-backed default lane を残すことで、local CLI、CI、external game 開発の検証を複雑化せずに済む
- `Pages` は watch UI を storage/backend と独立に配信できる

**トレードオフ:**
- vendor は 1 社に寄らず、DB と object storage は分離される
- first deploy は single logical queue authority 前提であり、multi-node fairness や distributed worker coordination は後続に残る
- latest world state を DB に置く場合は read は簡潔になるが、snapshot/history と保存場所が分かれる
- file-backed default lane と object storage lane の両方を保つため、artifact path/prefix の扱いは spec で明示し続ける必要がある

**制約:**
- durable metadata と object artifact の責務分離は崩さない
- public watch/read の実装場所を後で `Workers` や `Cloud Run` に移しても、`Neon + R2` の保存境界は維持する

---

## [2026-05-29] Phase 6 Postgres schema/query stack は Atlas + sqlc + pgx を採用する

### Context
`0056` で導入した Postgres durable store は、startup 時の inline DDL と手書き `pgx` query で最小実装を成立させた。
しかし Phase 6 以降は durable state と operator/read path が広がり、schema versioning、migration plan 生成、
query 数の増加、review 時の差分可読性を早めに揃える必要がある。

今回ほしいのは、現状 DB schema と desired schema の差分を migration plan として作れること、および SQL を review しやすい形で
query/source-of-truth として保てることである。一方で production apply workflow、approval gate、migration lint/drift detection の自動化は、
この段階では Atlas へ期待しない。

### Decision
Phase 6 の Postgres schema/query stack は次で固定する。

- schema source of truth は SQL 形式で repo に保持する
- schema diff / migration plan generation には Atlas を使う
- query authoring / typed generated code には `sqlc` を使う
- runtime driver は `pgx/v5` を使う

Atlas は migration plan generation と apply artifact の管理に限定して使い、
governance や production rollout の gate は repo/workflow 側で別途設計する。

### Consequences

**利点:**
- desired schema SQL を正本にしつつ、current DB schema との差分から migration plan を機械生成できる
- query 数が増えても SQL を直接 review しながら generated typing を得られる
- `pgx` の PostgreSQL 向け機能と `sqlc` の generated code をそのまま組み合わせられる
- startup inline DDL を廃止し、runtime と schema apply の責務を分離できる

**トレードオフ:**
- Atlas と `sqlc` の 2 ツールを導入するため、最初の setup と generate 手順は増える
- generated code は review noise になりうるため、SQL source と生成物の配置整理が必要になる
- Atlas Pro 前提の lint/drift detection に頼らないため、運用 gate は別途設計する必要がある

**制約:**
- desired schema SQL を source of truth とし、migration file を唯一の schema 定義として扱わない
- service runtime は schema bootstrap を持たず、既存 schema が適用済みであることを前提にする
- `QueueStore` など既存 service seam は維持し、backend 内部の query 実装だけを段階的に generated code へ寄せる
