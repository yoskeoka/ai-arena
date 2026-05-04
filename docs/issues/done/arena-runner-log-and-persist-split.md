# `arena-runner` の巨大単発 JSON 出力を、ログ出力と永続化データに分離したい

## 概要

現在の `arena-runner` は、試合完了時に match record 全体を 1 つの巨大 JSON として `stdout` に出力している。

`echo-count` fixture のように短時間で完了する試合では大きな問題になりにくいが、長時間試合や途中中断を考えると、以下が不足している。

- 試合進行中の観測しやすい event/log 出力
- 中断時点までの snapshot/exported state の外部可視化
- 再開可能性を高めるための永続化先の分離

## 現在の挙動

- `arena-runner` は試合終了時に final record を単発 JSON で出力する
- event log は final record の内部配列としてのみ見える
- snapshot / exported snapshot も終了時 record に埋め込まれる
- 中断や失敗が起きても、途中経過を逐次ログとして追いにくい

## 期待する整理

まず責務を 2 つに分ける。

- `log`
  - match 開始情報
  - event log
  - 終了時または中断時の state
- `persist data`
  - 現在 final に出している match record JSON

つまり、観測用の逐次出力と、後段で保存・再利用する構造化データを分離したい。

## 方向性

### 1. log は逐次 JSON で出す

- match metadata を試合開始時に 1 レコード出す
- event log は発生のたびに 1 行 1 JSON で出す
- 終了時または中断時に snapshot / exported state をログ出力する
- `match_id` は各ログレコードに含める
- 実装は `slog` ベースに寄せる

### 2. persist data は巨大 JSON のまま維持してよい

- 現在 `stdout` に出している final record JSON は、persist 用 artifact としては有用
- ただし「ログ」と同じ stream に混ぜるのではなく、保存先を意識した別出力にしたい

### 3. local 実行では永続化先を選べるようにしたい

- ローカル実行では file path を指定して final record を保存できるとよい
- これにより CLI から実行しても、ログ確認と artifact 保存を両立しやすい
- 将来 online 環境や DB/object storage に置き換える時も、log と persist の境界を保ちやすい

## 具体的なフォローアップ候補

- `arena-runner` の出力契約を `docs/specs/platform.md` で整理する
- `stdout` / `stderr` / file output の責務分担を決める
- `slog` を使った逐次 JSON log を導入する
- final record の保存先を CLI option で指定できるようにする
- 中断時にもその時点の snapshot / exported state を出力できるようにする
- turn ごとの保存や resume 可能性は、別段階で検討する

## defer 理由

`platform-phase2-02-fixture-e2e` の主目的は、`echo-count` fixture と e2e で platform の happy-path / failure-path を閉じることだった。

このタイミングで runner の出力契約、逐次 logging、永続化先、resume 前提の保存戦略まで広げると、元の plan から外れて実装完了が遅れる。

そのため今回は issue として切り出し、後続の runner / replay / persistence 系タスクで扱う。
