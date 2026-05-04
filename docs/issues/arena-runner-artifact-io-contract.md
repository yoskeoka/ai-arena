# `arena-runner` の artifact input/output 契約が揃っておらず、record と history の役割が CLI から分かりにくい

## 概要

`platform-phase2-03-replay-debug` で `arena-runner` に replay/debug entrypoint を追加した結果、以下の CLI surface が成立した。

- output
  - `--log-output`
  - `--persist-record`
  - `--exported-snapshot-output`
- input
  - `--record-input`
  - `--snapshot-input`
  - `--history-input`
  - `--target-turn`

しかし、現状は source-of-truth artifact と derived debug artifact の階層が flag 名から読み取りづらい。

特に以下が混線している。

- `--persist-record` と `--record-input` は実質的に入出力の対だが、片方だけ `persist` という動詞で naming されている
- `--history-input` はあるが、history 専用 output flag は存在しない
- `--exported-snapshot-output` はあるが、internal snapshot の dedicated output flag は存在しない
- `--target-turn` は replay/resume 境界 turn の指定だが、名前だけでは `history-input` 専用か `record-input` でも使えるのかが分かりにくい

## 現在の挙動

現状実装の contract は以下である。

- source of truth は persisted final match-record artifact 1 個
  - `--persist-record <path|stdout>` で出力
  - `--record-input <path>` で再入力
- history は source of truth ではなく、record の `event_log` から抽出した derived input
  - `--history-input <path>` は補助 entrypoint
  - dedicated output flag はまだない
- snapshot は source of truth ではなく、debug 用に hand-crafted 編集も許す derived input
  - `--snapshot-input <path>` は補助 entrypoint
- `--target-turn <n>` は `--history-input` だけでなく `--record-input` とも組み合わせて使う replay/resume 境界 turn

この設計自体は妥当だが、CLI naming と help text だけ見ると artifact hierarchy が十分に伝わらない。

## 望ましい方向

### 1. `record` を replay/debug の第一入口として固定する

- 通常利用では `--record-input` を優先する
- `history-input` は「record から `event_log` を抜き出した raw history file を直接与えたい特殊ケース」に限定する
- `snapshot-input` は「debug 用に hand-crafted で微調整したいケース」に限定する

### 2. input/output naming を対称にする

候補:

- `--persist-record` を `--record-output` に寄せる
- `--target-turn` を `--replay-target-turn` または `--resume-target-turn` に寄せる

この rename は help text の改善だけではなく、spec / examples / compatibility policy と一緒に決める必要がある。

### 3. artifact 出力戦略を再設計する

個別 output flag を増やし続けるより、`--output-dir` を導入して runner が標準レイアウトで artifact を配置する方が UX がよい可能性がある。

例:

```text
arena-runner-match-output/<match-id>/
  record.json
  structured-log.ndjson
  snapshot.json
  exported-snapshot.json
  event-log.json
```

この場合の整理候補:

- `record.json` を source-of-truth artifact に固定
- `event-log.json` / `snapshot.json` / `exported-snapshot.json` は derived artifact
- 個別 flag は override として残すか、互換期間の alias にする

## 具体的なフォローアップ候補

- `docs/specs/platform.md` で artifact hierarchy を先に固定する
- `--persist-record` / `--record-input` / `--history-input` / `--target-turn` の naming を再検討する
- `--output-dir` 導入可否を決める
- `history` / `snapshot` / `exported snapshot` の dedicated output policy を決める
- `record` から `event_log` / `snapshot` を機械的に抜き出す helper CLI を用意する

## 優先度

中。

今の CLI は機能的には動いているが、artifact model を誤読させやすい。
特に replay/debug を継続運用する前に、`record` が source of truth で `history` / `snapshot` が derived entrypoint であることを CLI 上でも自然に読めるようにしておきたい。
