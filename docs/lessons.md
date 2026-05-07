# Lessons

## [2026-04-26] ドキュメント言語方針の適用範囲を先に固定する

- **Mistake**: 「docs を日本語にする」を広く解釈し、コミットメッセージや PR メタデータまで同じ方針で寄せる前提で進めかけた
- **Pattern**: リポジトリ内ドキュメント言語と、コード・VCS・GitHub メタデータの言語を分離せずに扱ってしまう
- **Rule**: 言語方針を扱うときは、少なくとも `docs/*`、`AGENTS.md`、生成物 UI/メッセージ、コードコメント、コミットメッセージ、PR title/description の6領域に分けて適用範囲を明示する
- **Applied**: `AGENTS.md` の運用ルール、`docs/design-decisions/adr.md` の内部ドキュメント言語 ADR、今後の `ai-arena` ドキュメント/PR 運用

## [2026-04-26] timeout / invalid action の仕様は逃げ得を先に潰す

- **Mistake**: `no_action` を通常の引き分け解決に巻き込むと、意図的に応答しないことへ戦術上の余地を残してしまう
- **Pattern**: timeout や invalid action を「単なる欠損入力」として扱い、インセンティブ設計まで詰めずに仕様化してしまう
- **Rule**: 対戦ゲームで timeout / invalid action を定義するときは、まず「意図的に出さないことにメリットがないか」を確認し、必要なら個別敗北扱いを明記する
- **Applied**: `docs/specs/janken-game.md` の `no_action` 解決規則、今後の AI Arena ゲーム仕様

## [2026-04-27] 起動確認とゲーム初期化確認を分ける

- **Mistake**: `init` の成功をそのまま runtime health 全体の確認とみなすと、platform が確認できる範囲とゲーム固有初期化の範囲が混ざる
- **Pattern**: 実行基盤の起動成功と、ゲームプロトコル上の準備完了を同じ `ready` 概念で扱ってしまう
- **Rule**: platform spec では「load / instantiate / stream 接続」と「game init request に応答できること」を分けて定義し、ゲーム固有 readiness は `init` 応答の意味として各ゲームが定義する
- **Applied**: `docs/specs/platform.md` の起動確認記述、`docs/specs/janken-game.md` の `ready: true` の意味づけ

## [2026-04-29] match end は one-way notification で定義する

- **Mistake**: 試合終了後の最終結果通知と AI の後処理を、通常の request/response と同じ形で扱うと、不要な応答義務と runtime 終了責務が曖昧になる
- **Pattern**: 「結果を渡すこと」と「終了猶予を与えること」を別メカニズムにせず、末尾プロトコルを詰め切らない
- **Rule**: AI が最終振り返りやレポート出力を行う必要があるゲームでは、試合終了メッセージは response 不要の `game_over` notification とし、必要なら `shutdown_after_ms` を明示する
- **Applied**: `docs/specs/platform.md` の `game_over` / shutdown 記述、`docs/specs/janken-game.md` の最終通知例

## [2026-04-29] platform 共通仕様とゲーム固有運用の境界を残す

- **Mistake**: `stderr` の公開タイミングや AI 差し替え条件まで platform 共通仕様で固定すると、ゲームごとの面白さや運営ルール差分を潰してしまう
- **Pattern**: 保存責務と公開タイミング、再起動フックと発火条件を同じ層で定義してしまう
- **Rule**: platform spec では「保存する」「差し替え前処理フックを持つ」までを定義し、公開タイミングや差し替え条件はゲーム仕様へ委ねる
- **Applied**: `docs/specs/platform.md` の `stderr` 取得タイミングと `turn` 前差し替え前処理の記述

## [2026-05-07] fixture 検証都合を product 向け切替機能へ一般化しない

- **Mistake**: `echo-count` の subprocess 検証 needs を、そのまま `arena-runner` の `--game-master-mode` と registry の複数 mode 切替へ一般化した
- **Pattern**: e2e fixture の都合で必要な分岐を、通常経路の user-facing 設定や汎用 registry contract に昇格させてしまう
- **Rule**: ある分岐が fixture/e2e の等価性確認にしか必要ないなら、まず別 fixture game か test-only registration で閉じる。通常利用者が選ばない切替を product path に足さない
- **Applied**: `echo-count` / `echo-count-subprocess` の分離、`arena-runner` の game-master mode 削除、Phase 3 runtime boundary 設計

## [2026-05-07] execution 系依頼は PR 初回 follow-up まで止めない

- **Mistake**: 実装完了時点で区切ってしまい、user が期待している `commit -> push -> PR 作成 -> CI/初回 follow-up` まで進めずに止まった
- **Pattern**: `execute-task` の完了条件をローカル実装とテスト成功に寄せすぎて、repo workflow 上の landing steps を会話上の「次の依頼待ち」と誤認する
- **Rule**: user が `commit` や `PR作成まで` を含む execution 完了を求めたら、完了報告は少なくとも `commit -> push -> PR 作成 -> 30秒待機後の CI/check 確認` を終えてから行う。途中経過は commentary で出し、final は workflow の停止条件を満たした後に限る
- **Applied**: `execute-task` 後の `review-task` 運用、`ai-arena` / `ww` の実装ブランチ handoff、今後の PR 作成依頼全般

## [2026-05-08] spec では実装シンボル名より責務境界を書く

- **Mistake**: spec の説明を補強するつもりで、`registry.Registry.Lookup*` のような実装コード名に寄せた表現を許容しかけた
- **Pattern**: 現在のコード構造をそのまま spec に写してしまい、責務は同じでも小さなリファクタリングで spec 修正が必要な書き方になる
- **Rule**: spec では concrete な関数名・メソッド名・型名をむやみに持ち込まず、まず責務・入出力・境界を書く。実装で使っている型や interface を書く場合も、それが安定した抽象概念として spec の主語になっているときだけに限る
- **Applied**: `docs/specs/platform-game-registry.md` の lookup 流れ、今後の platform / registry / runner 系 spec 全般

## [2026-05-08] runtime 差分を長く残すなら game id も分ける

- **Mistake**: Phase 4 の Go-WASM 検証を既存 `janken` game id にそのまま重ね、subprocess 系の `janken` と runtime 差分を spec / fixture / registry 上で分離しない前提で進めた
- **Pattern**: 同じルールセットでも、長期的に別 verification line として残す runtime 差分を「player artifact だけの違い」とみなして game identity へ反映しない
- **Rule**: 同一ルールを共有しても subprocess 系と WASM 系のように別 verification line を併存させるなら、`echo-count` と同様に game id を分けて registry / fixture / spec で並立させる
- **Applied**: `janken` と `janken-wasm` の分離、今後の runtime-specific fixture game 設計
