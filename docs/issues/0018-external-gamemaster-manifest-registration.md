# 外部 game master を manifest ベースで一時 registry 登録できず、consumer repo の tagged runner E2E が built-in game に閉じている

## 概要

`arena-runner` は `--game` / `--game-version` を受け取ると、先に
`game_id + game_version major` で registry lookup を行う。

この前提自体は正しいが、現状の lookup 元が built-in descriptor だけだと、
別 repo にある external game master を tagged runner から検証したい consumer repo が
`unsupported game` で止まる。

実際に `reversi-ai-arena` の Phase 1 runner E2E を組もうとすると、
`go install github.com/yoskeoka/ai-arena/cmd/arena-runner@v0.1.0` で入れた
tagged runner は `reversi` を lookup できず、match 開始前に失敗する。

これは Reversi 固有の問題ではなく、
「platform repo の built-in registry に入っていない external game を、
consumer repo 側が versioned runner で自己完結検証できない」という一般問題である。

## 現在の制約

- registry lookup は `game_id + game_version major` を前提にしている
- runner は lookup 成功後の descriptor を基準に game master metadata を確定する
- AI player 側には sidecar manifest があり、runtime と metadata を entry-path 横で解決できる
- 一方で game master 側には、それに相当する consumer-supplied descriptor 読み込み経路がない

そのため、external game repo 側で local subprocess game master binary を用意しても、
runner に「この game はこの command / metadata / ruleset support で起動できる」と
伝える標準手段が存在しない。

## なぜ単純な CLI 追加では足りないか

`--game-master-command ...` のような ad hoc flag 追加だけでも一時的には進められるが、
それだけでは以下が散らばる。

- `game_id`
- `game_version`
- `ruleset_version` または supported rulesets
- build mode (`local-subprocess`)
- resume/debug 時に必要な snapshot/history build 入口

AI player 側で sidecar manifest に寄せている metadata/runtime 責務と比べても、
game master だけ CLI 引数の寄せ集めにすると contract が不均衡になる。

## 望ましい方向

### 1. game master 用 manifest / descriptor file を定義する

最低限、以下を持てる shape が必要。

- metadata
  - `game_id`
  - `game_version`
  - `ruleset_version` または supported rulesets の宣言
- runtime
  - `kind = local-subprocess`
  - command / args
- replay/debug 入口
  - まずは `fresh run` 必須でもよい
  - snapshot resume / history replay は Phase 分割でもよい

### 2. runner がその manifest を読んで一時 descriptor を組めるようにする

- built-in registry を置き換えるのではなく、opt-in の追加解決経路として扱う
- 例:
  - `--game-master-manifest <path>`
  - `--game-descriptor <path>`
- runner はこの入力から temporary descriptor を構築し、
  通常の registry lookup 後と同じ metadata compatibility / build flow へ流す

### 3. 初期スコープは `local-subprocess` に限定する

最初から plugin 全般や remote backend まで広げる必要はない。

まず必要なのは、
consumer repo が「repo 内で build した local executable game master」を
tagged `arena-runner` で起動し、artifact 生成まで完走できること。

## 受け入れ条件のイメージ

- external consumer repo が built-in registry 非登録 game を tagged runner で起動できる
- runner は manifest 由来 metadata と player metadata の compatibility を通常どおり検証する
- local subprocess game master の command 解決が repo-local path で再現できる
- 少なくとも 1 つの external game fixture で E2E が通る
- built-in game (`janken`, `dungeon`, 既存 `echo`) の lookup / build path には影響しない

## フォローアップ候補

- `docs/specs/platform-game-registry.md` に「built-in descriptor だけでなく、
  consumer-supplied temporary descriptor を runner が読める」経路を追加する
- `docs/specs/game-master.md` に external game master manifest の責務境界を追加する
- AI player sidecar manifest との対称性を意識して、metadata/runtime schema の共通化余地を整理する
- 初回は `local-subprocess` のみ、resume/history build 入口は後続 plan に分離する

## 優先度

中〜高。

external game repo を tagged runner で継続検証する方針を取るなら、
この入口がない限り built-in registry に事前登録された game 以外は
consumer repo 単体で green path を持てない。
