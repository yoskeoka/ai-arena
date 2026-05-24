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

## [2026-04-26] 内部ドキュメント言語: `docs/*` は当面日本語

### Context
`ai-arena` のリポジトリ内ドキュメントは、日本語と英語が混在し始めている。一方で既存 ADR では
公開プラットフォームの UI 言語とロケールを英語 + UTC に決めている。

ここで決めたいのは公開 UI の言語ではなく、リポジトリ内の内部設計・計画・仕様ドキュメントを
どの言語で書くか、という運用ルールである。

### Decision
**`ai-arena/docs/*` 配下の内部ドキュメントのうち、plan / issues / specs / ADR などは、当面日本語で統一する。**

### Consequences

**利点:**
- 現時点の主要開発者が最短で思考・設計を書き下せる
- project-plan、exec-plan、spec、issue、ADR の読み書きが同じ言語に揃う
- 公開 UI/公開 API の英語方針と、内部設計ドキュメント運用を分離して管理できる

**明確化:**
- これは `docs/*` 配下の内部ドキュメント方針である
- プロダクト UI、外部公開 API、対外ドキュメントの言語方針は別途判断する
- プロジェクトが生成する成果物の UI やメッセージ言語は、この方針の対象外とする
- 実装コード中のコメントは英語のままとする
- コミットメッセージは英語のままとする
- PR title / PR description は英語のままとする
- 既存の英語 docs は必要に応じて順次日本語へ寄せる

**トレードオフ:**
- 将来、英語話者の共同編集者が増えた場合は再判断が必要になる

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

## [2026-04-26] 不完全情報の中心設計: 視界制限を採用

### Context
AI Arena の価値は、単純なルール実装ではなく AI が継続的に状態推定・記憶・探索戦略を
持つ必要があるゲームを提供することにある。完全情報ゲームだけでは、AI の内部状態や
探索アルゴリズムよりも固定戦略の最適化に寄りやすい。

### Decision
**本命ゲームでは視界制限による不完全情報を中心設計として採用する。**

### Consequences

**利点:**
- AI が過去の観測結果を内部記憶として保持する意味が生まれる
- LLM への単発問い合わせより、継続的な状態推定と探索戦略が重要になる
- 観戦者にとっても「見えていない相手との遭遇」がドラマになる

**却下した代替案:**
- 常に全体マップを公開する完全情報設計: 実装は簡単だが、AI 競技としての面白さが浅い

**トレードオフ:**
- 仕様・観戦・デバッグが複雑になるため、全体状態とプレイヤー可視状態の両方を定義する必要がある

---

## [2026-04-26] Phase 2 実証ゲーム: じゃんけんを採用

### Context
Phase 2 ではプラットフォームの基礎実装を早く検証したい。一方で、後続のダンジョンゲームに
必要な性質の一部は先に確認しておく必要がある。

### Decision
**最初の実証ゲームとして複数ラウンドのじゃんけんを採用する。**

### Consequences

**利点:**
- ルールが極小で、ゲームマスター実装と AI プロトコル確認に集中できる
- 同時行動、隠し手、タイムアウト、順位決定などプラットフォームの基本機能を検証できる
- 観戦表示もラウンド reveal 型で簡単に作れる

**却下した代替案:**
- いきなりダンジョンゲームを実装する: 要素が多すぎて、プラットフォーム問題とゲーム問題が分離しにくい
- コイントスのような単純すぎるゲーム: 同時行動やプレイヤー選択の表現力が不足する

**トレードオフ:**
- 戦略の深さは浅いため、長期的な主力ゲームには向かない
- じゃんけんだけでは順番制ゲームや複雑な action schema の検証はできない

---

## [2026-04-26] Phase 2 実行戦略: 先にローカルプロセス、後で WASM 統合

### Context
project-plan の最終形は WASM 提出だが、Phase 2 の主目的はプラットフォームの進行制御と
AI プロトコルを早く実証することにある。WASM ランタイム統合まで同時に着手すると、
切り分けが難しくなる。

### Decision
**Phase 2 の最初の検証ではローカルプロセス実行を使い、プロトコル確認後に WASM 統合へ進む。**

### Consequences

**利点:**
- ターン進行、ゲームマスター、ログ収集などの基礎問題を先に分離して検証できる
- AI サンプルを素早く作って通信仕様を叩ける
- 問題発生時に transport/進行制御の不具合か、WASM 実行統合の不具合かを切り分けやすい

**整合性メモ:**
- これは最終提出形式の決定を覆すものではない
- 外部向け仕様は WASM 提出を前提に維持する
- ローカルプロセス実行は Phase 2 の開発・検証手段としてのみ使う

**トレードオフ:**
- 実装中は最終運用形と一時的に差分が出るため、WASM 統合を後回しにしたまま固定化しない運用が必要

---

## [2026-05-12] Dungeon domain は ruleset/layout/state/payload の subsystem 境界で再編する

### Context
Phase 5 完了時点の `games/dungeon` では、`Ruleset`、`FullState`、`Match` に static rule、
generated layout、mutable state、payload assembly が集まっていた。この構成でも fixed-map / seeded-maze
までは実装できるが、敵、罠、消費アイテム、複数 floor などを追加すると、どの data が seed 由来で、
どの data が進行中 state で、どの data が外部 contract payload なのかが曖昧になり、変更影響を追いにくい。

### Decision
Dungeon domain は少なくとも以下の subsystem 境界で再編する。

- `ruleset definition`: static rule
- `generated layout`: seed から決まる初期配置
- `match state`: turn ごとに変化する mutable state
- `contract payload`: AI / platform / spectator 向け view

`Match` は domain façade として残してよいが、state assembly、layout 生成、payload 投影までを
再び 1 箇所へ集約してはならない。fresh run / resume では static rule 選択、layout 再構築、mutable state 復元を
分離して扱う。

### Consequences

**利点:**
- 将来 actor / item / effect / combat を追加するときに、layout 由来か mutable state 由来かを先に固定できる
- replay / resume で `rng_seed` から再生成した layout と保存済み payload の一致確認を独立責務として保てる

---

## [2026-05-24] official external game master admission は built-in / sandboxed / external adapter を分離する

### Context

`0052` により、consumer repo が `--game-master-manifest` で `local-subprocess` game master を
fresh run 検証できる dev-only overlay path は成立した。一方で、official registered game として
何を admission するかは別問題である。

ここでは次を同時に整理する必要がある。

- 運営自身の game master と、信頼できるパートナーが source 提供して運営が review する game master を
  どの tier で扱うか
- source を取り込まずに platform 管理下で運用する official submission に、どの runtime kind を許可するか
- GPU / LLM など platform 単体では抱えにくい資源を必要とする game master を、どの route で受けるか
- `docker` / OCI container を今すぐ official support に含めるか

### Decision

- official game master admission は少なくとも次の 3 tier に分ける
  - `official built-in`
  - `official sandboxed submission`
  - `official external adapter`
- `official built-in` には、運営自身が実装した game master に加え、信頼できるパートナーから source 提供を受けて
  運営が review / CI / release 管理を引き受ける game master も含める
- `official sandboxed submission` の第一候補 runtime kind は `wasm-wasi` とする
- raw `local-subprocess` executable は host 隔離が弱いため、official registration では許可しない
- `trusted external adapter` は GPU / LLM / 専用 service 依存の game 向け official route として残す
- `docker` / OCI container は将来候補として残すが、現時点では未サポートとする

### Consequences

**利点:**
- dev-only overlay と official admission policy を混同せずに進められる
- `official built-in` は通常の repo review / CI / release フローへ載せられる
- self-contained な外部投稿 game は `wasm-wasi` へ寄せることで、platform 運用と host security を単純化できる
- GPU / LLM 系の例外は `official external adapter` に隔離できる

**Docker を今すぐ採用しない理由:**
- network isolation、filesystem / capability 制限、resource limit、image / dependency scan まで含めると
  admission / 運用負荷が高い
- 現時点では、まず `wasm-wasi` で受けられない具体例が出るまで lane を開けない方が簡潔

**トレードオフ:**
- `wasm-wasi` では使える言語・ライブラリ・OS 機能に制約がある
- その制約で受けられない game master は、built-in 化するか、将来の Docker lane、または
  `official external adapter` を使う判断が必要になる
- `Match.Apply` のような façade API を残しつつ、内部の state assembly と payload projection を差し替えやすい

**制約:**
- platform から見える external payload shape は互換維持を優先する
- dungeon package tree は引き続き top-level portable package とし、新しい `internal` 依存を持ち込まない

---

## [2026-05-12] Dungeon turn progression は fixed-order phase engine で整理する

### Context
Phase 5-06-01 で dungeon domain を ruleset / layout / state / payload に分離した後も、
turn progression 自体は `Match.Apply` の中で movement、chest resolution、goal finish、score refresh、
discovery refresh を直列に処理していた。現状 mechanic では読めるが、このまま actor interaction、tile effect、
inventory、combat を足すと `Apply` が条件分岐の集積点へ戻りやすい。

### Decision
Dungeon turn progression は fixed-order phase engine として扱う。

- `Match.Apply` は public façade と orchestration に寄せる
- action normalization
- movement resolution
- interaction resolution
- terminal / score update
- visibility refresh

現在ある move / wait / chest split / goal bonus / discovery refresh はこの phase 順へ載せ替える。
将来の actor interaction、tile effect、actor effect tick は既存 phase 順の空き slot へ差し込む。

### Consequences

**利点:**
- turn mechanic の追加時に「どの phase が state を読むか/書くか」を先に固定できる
- targeted scenario catalog を pipeline parity gate として使い続けられる
- `Match` façade を保ったまま内部 phase を個別 unit test しやすい

**制約:**
- 現行 ruleset の public behavior は phase 化だけでは変えない
- phase 抽象化を広げすぎず、まずは現行 mechanic の移送に留める

---

## [2026-05-14] Dungeon feature expansion は subsystem seam と deterministic contract を先に固定する

### Context
Phase 5-06-01 と 02 で、dungeon domain は ruleset / layout / state / payload に分かれ、turn progression も
fixed-order phase engine へ寄せた。ここから actor、item、inventory、combat、effect、visibility/FOV を追加できるが、
phase 追加や helper 追加をその都度 ad-hoc に行うと、hidden information 境界、replay source-of-truth、stable iteration
order が再び曖昧になる。

### Decision
Dungeon feature expansion は subsystem seam を先に固定してから進める。

- subsystem は少なくとも `actors`、`items`、`inventory`、`combat`、`effects`、`visibility/fov` を前提にする
- 各 subsystem は `generated layout` 由来か `match state` 由来かを先に決めてから payload へ投影する
- subsystem hook は既存 5 phase の内側 slot へだけ差し込み、新しい top-level phase を安易に増やさない
- subsystem を跨いでも deterministic contract は single RNG ownership、fixed phase order、stable iteration order、replay / resume source-of-truth を維持する
- scenario catalog、deterministic result regression、replay / resume verification は別役割の gate として併存させる

### Consequences

**利点:**
- feature expansion 前に「どの subsystem がどの phase で state を読むか/書くか」を固定できる
- hidden information を bypass する近道を runtime helper や reference AI 側へ入れにくくなる
- same-condition regression と replay verification の責務が混ざらず、failure の意味が読みやすい

**制約:**
- seam は narrow hook と deterministic helper の明示に留め、未実装 subsystem の抽象化を過度に広げない
- platform は引き続き payload 境界だけを前提とし、subsystem internal state を特別扱いしない

---
