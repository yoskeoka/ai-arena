# Project Plan: AI Arena

## Goal

人間がプレイせず、人間が開発したAIプログラムが代わりにプレイするオンラインゲームプラットフォームを開発する。参加者は優秀なAIの開発を競い合い、観戦者はリアルタイムでゲーム画面を観戦できる。

## Significance

- AI開発の腕を競い合う新しい形の競技プラットフォーム
- LLMに丸投げできない高速判断が求められる設計により、本質的なAIアルゴリズム開発力が試される
- 観戦可能なゲーム画面により、AI同士の対戦をエンターテインメントとして楽しめる
- AIの開発・改善サイクルを通じて、参加者のプログラミングスキル向上にも寄与

## Architecture

### 2層構造

1. **汎用プラットフォーム**: ゲーム非依存の対戦管理基盤。参加 AI の実行環境、対戦管理、認証、記録、観戦配信、およびゲームごとの差分を吸収する連携インターフェースを提供する
2. **ゲーム実装**: 個別ゲームを成立させるための提供物一式。ゲーム仕様、ゲームマスター、全体情報API、必要に応じてビジュアライザやサンプルコードを含む。ゲームマスター本体は platform 内または trusted external game backend のいずれかに配置できる

### ゲーム実行形態

- **標準形態**: ゲームマスター実装は platform 内で完結して動作する。CPU/メモリ要求が platform の運営予算内に収まるゲームはこの形を基本とする
- **拡張形態**: ゲーム内容の動的生成に外部 LLM や強い GPU を使う場合など、platform 単体で収まらない計算資源を要するゲームでは、ゲーム提供者が管理する trusted external game backend 上でゲームマスター本体を動かせるようにする
- **責務分離**: どちらの形態でも、参加 AI の実行、AI との公式入出力契約、対戦開始/終了の管理、認証・記録・観戦の基盤は platform 側の責務とする。external backend 側はゲームマスター本体と、その実行に必要な計算資源・補助サービスのみを担う。platform 上には external backend との通信を仲介するゲーム別アダプタを残し、参加 AI からは常に platform が唯一の公式対向先に見えるようにする

### ゲーム登録要件

プラットフォームに新しいゲームを登録するには以下が必要:

- **必須**: ゲーム仕様書（AIが満たすべき入出力の仕様とゲームのルール）
- **必須**: ゲームマスタープログラム（ゲーム進行を制御するサーバーサイドロジック）
- **必須**: ゲーム全体情報が取得できるAPI群とAPI仕様書
- 任意: ゲーム全体情報APIを利用したビジュアライザプログラム
- 任意: AIプログラムのひな型（サンプルコード）

## Requirements

### ゲーム形態
- **ターン制**: 公平性と実装の明快さを担保。ターンの長さ（応答制限時間）はゲームごとに設定可能
- **高速判断**: LLMに丸投げできないよう、短時間での判断が求められる設計を推奨
- **ターンモデル**: 全プレイヤー同時アクション / 順番制の両方をプラットフォームとして対応可能にする
- **順位決定**: ゲームごとに決定方式を定義（スコア、時間、勝率など複数タイプ）
- タイムアウト時は行動スキップ扱い

### AI通信プロトコル
- **stdin/stdout**: AIプログラムはプロセスとして起動され、stdin/stdoutで通信する
- **JSON-RPC 2.0 準拠**: メッセージフォーマットは JSON-RPC 2.0 に準拠。`id` でリクエスト/レスポンス対応、`method` でメッセージ種別を区別
- **長期プロセス**: AIプログラムはゲーム開始から終了まで起動しっぱなし。過去の入出力から学習・記憶・戦略の更新が可能
- **応答タイムアウト**: ゲームマスターからリクエスト（msg-id付き）が送られ、AIは対応するmsg-idを添えて規定時間内に応答しなければターンスキップ

### AIプログラム提出
- **言語自由**: WASMにコンパイル可能な言語であれば何でも（Rust, C/C++, Go, Zig, AssemblyScript等）
- **提出形式**: WASM バイナリ（WASI準拠）で提出。プラットフォーム側でのコンパイル・ビルドは行わない
  - WASM/WASI はデフォルトでサンドボックス化されており、外部通信禁止が設計上保証される
  - wazero（pure Go WASMランタイム）でプラットフォームにネイティブ統合
  - 起動がミリ秒単位で高速
- **提出可否と公式サポート範囲は分ける**: WASM/WASI の実装仕様に沿う module であれば提出対象にできる一方、公式の guide・sample・verification assets を整備して動作保証する言語は段階的に広げる
- **リソース制限**:
  - メモリ: WASM仕様のlinear memory max pagesで上限設定可能
  - CPU/実行時間: Goのcontext.WithTimeoutで制御。ターン制ゲームの応答タイムアウト機構と自然に統合
- **プラットフォーム非依存**: WASMはクロスプラットフォーム。提出者がOS/アーキテクチャを意識する必要がない
- **ホットデプロイ**: AIプログラムの改善バージョンはいつでも提出可能。反映タイミング（次のターン、次のラウンド、次のゲーム等）はゲームルールで定義する

### AIログ・デバッグ
- WASMサンドボックスにより、AIプログラムからの外部通信は一切不可（送受信とも）
- **stdout**: JSON-RPC通信専用（プラットフォームとの公式チャネル）
- **stderr**: AI開発者が自由に書き出すログチャネル（思考過程、評価値、戦略判断の記録、デバッグ情報等）
- プラットフォームがstderrを全て捕捉・保存し、AI開発者はAPI経由で取得可能
- AIの改善サイクル（対戦→ログ分析→改善→再登録）を支援するための基盤

### AI参加ルール
- **外部通信は不可能**（WASMサンドボックスにより設計上保証）
- プレイ中のメモリ内学習（戦略更新等）は許可

### 観戦機能
- ゲーム状態をリアルタイム表示できる仕組み
- ゲームの状況が視覚的に理解しやすいUI
- 観戦プロトコルの詳細は後続フェーズで設計

### プラットフォーム
- **実装言語**: Go
- オンラインで複数のAIが参加・対戦できるインフラ
- マッチメイキングやリーグ/トーナメント形式の対戦管理
- ゲームマスター実装の配置先として、platform 内実行と trusted external game backend 実行の両方を将来的に扱える設計
- trusted external game backend を使う場合でも、AI 実行環境・match lifecycle・監査・記録の主導権は platform 側に残し、`event_log` / `snapshot` / `exported_snapshot` は platform 側で保持する正本とする
- trusted external game backend との接続には、少なくとも認証済みの専用チャネルを要求し、恒久運用では相互認証を前提に検討する。あわせて external backend は各 turn の結果、監査に必要なイベント列、および観戦・公開に使える状態スナップショットを platform に返送する責務を負う

## Milestones

以下の phase は厳密な直列ではなく、先に固定すべき interface 契約を gate としつつ、依存関係を守りながら並行で進める。

- [x] Phase 1: プラットフォーム設計・ゲームコンセプト策定（ドキュメントのみ）
- [x] Phase 2: プラットフォームコア実装 + `janken` によるローカル実行での実証
- [x] Phase 3: AI player / platform / game master の共通 interface 契約を固定し、複数ゲーム対応と trusted external game backend 対応の土台を整える
- [x] Phase 4: WASM/WASI 実行を正式な AI 実行経路として成立させ、Go 製 AI を基準に `janken` で検証する
- [x] Phase 5: ダンジョンゲーム MVP を、プラットフォーム改善と並行で進める
  - `dungeon-game-ai-arena` への切り出しと golden parity 確認を完了し、ai-arena 側には platform / SDK / artifact / registry の責務を残す
  - dungeon 固有コードが `internal/platform/*` に依存しない境界を整え、platform 側の public contract 候補を狭く固定する
- [ ] Phase 6: match state・artifact・公開用 game state を扱える永続化基盤と service skeleton を整える
  - child item: match submission から result persist までを 1 本通す service skeleton を整える
  - child item: `record` / `event_log` / `snapshot` / `exported_snapshot` の保存モデルと read model を定義する
  - child item: resume / replay / audit に再利用できる source-of-truth artifact の置き場を固める
- [ ] Phase 7: AI 提出、game 提出、matchmaking、ranking、早期 deploy pipeline を含むオンライン運営基盤を整える
  - child item: AI submission / game registration / validation / queueing の operator flow を成立させる
  - child item: matchmaking / ranking / rerun を含む最小運営サイクルを整える
  - child item: service / worker / storage の最小 deploy 形を定め、継続運用できる実行トポロジを固める
- [ ] Phase 8: public な external game state を読み取る観戦用ビジュアライザを整える
  - child item: public game state を取得・配信する API / artifact 契約を固める
  - child item: spectator 向けの snapshot / event stream 読み取り導線を整える
  - child item: game repo 側 visualizer と platform 側 state delivery を接続できる最小 viewer を成立させる
- [ ] Phase 9: Go 製 WASM AI 開発フローの外部向け導線を整え、Rust を複数言語評価の先行レーンとして取り込みつつ、多言語サポート拡張の準備を進める
  - child item: Go WASM を公式サポート言語として扱う guide / sample / verification assets を整える
  - child item: Rust を先行 evaluation lane として CI / sample / compatibility matrix に載せる
  - child item: 他言語へ広げるための support policy と acceptance bar を定義する

### Phase の意図

- Phase 3 は、ダンジョンゲーム開発で無駄な手戻りを避けるために、先に共通 interface 契約を固める phase である
- Phase 4 は、project-plan の最終要件である WASM 提出を開発用の暫定経路ではなく正式な実行経路へ寄せる phase である
- Phase 5 のダンジョンゲーム開発は、Phase 3 の契約固定後に着手し、以後の platform 改善と並行で進める
  - 最初の到達点は、固定 1 ステージでゴール到達と宝箱回収を競う MVP とする
  - 実装は別 repo へ移動できる境界を保つため、ダンジョン固有コードは `internal` package に依存させない
  - 開発中のフィードバックサイクルは Go subprocess bot で高速化しつつ、同じ判断ロジックを WASM 版 reference AI へ流用できる形を目指す
- Phase 5 完了後の ai-arena は、特定ゲームの拡張よりも online platform としての service skeleton・永続化・運営導線を優先して進める
- Phase 6 の最初の到達点は、submission -> validation -> queue -> match run -> result persist を 1 本の運営フローとして通すことである
- Phase 7 は、その最小 online flow を運営可能な形へ厚くする phase であり、operator workflow・matchmaking・ranking・deploy を順に閉じる
- Phase 6 で公開向け state の永続化と供給基盤を整えた後は、Phase 8 のビジュアライザを段階的に前倒しで進めてよい
- Phase 9 では、WASM/WASI の実装仕様に沿う module は提出可能としたまま、公式の guide・sample・verification assets を整備して動作保証する AI 開発言語はいったん Go に限定する。Rust は platform の複数言語評価の先行候補として扱い、TypeScript や Python など開発者層を広げやすい言語は Go 向け資産の横展開先として将来サポート候補に残す

### 将来構想（優先度未定）
- ダンジョン探索ゲームの段階的拡張:
  - ソロ探索 → 資源競合 → 直接対戦 → 協力/裏切り選択
  - 視界制限による不完全情報、PvP/協力要素
- ゲームプラグイン機構（ゲーム登録のセルフサービス化）
- 複数の順位決定方式（スコア、時間、勝率等）のフレームワーク化
