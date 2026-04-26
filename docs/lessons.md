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
