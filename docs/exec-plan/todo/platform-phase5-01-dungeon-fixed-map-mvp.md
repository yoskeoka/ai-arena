# platform-phase5-01-dungeon-fixed-map-mvp
**Execution**: Use `/execute-task` to implement this plan.

## Objective

ダンジョンゲームの最初の実装単位として、1 ステージ固定マップ上でゴール到達と宝箱回収を競う MVP を成立させる。Phase 5 全体で必要になる「別 repo へ移せるゲーム境界」「game master の deterministic loop」「Go subprocess による高速 bot 検証」を先に固定し、後続の seed 付き生成と WASM reference AI がこの土台に乗るようにする。

depends on:

- `platform-phase3-03-game-master-runtime-boundary.md`
- `platform-phase4-01-wasm-runtime-contract.md`
- `platform-phase4-02-go-wasm-janken-verification.md`

## Scope

- ダンジョン固有コードを `internal` package へ閉じ込めない配置方針の確定
- 固定 1 ステージ・同時行動・ターン制の MVP ルール確定
- `visible_state` / `full_state` / `exported_snapshot` の最小 payload 形状確定
- ゴール到達順スコアと宝箱スコアの合算による初期採点ルール確定
- local subprocess で動く reference bot の最小成立
- 人手で動作確認しやすい helper の追加

この plan では以下は扱わない。

- 迷路自動生成
- モンスターや戦闘ルール
- 1 ステージを超える進行
- WASM 版 reference AI の成立
- online service / persistence backend の本実装

## Spec Changes

### `docs/specs/dungeon-game.md`

- Phase 5 MVP として扱うルールを実装レベルまで具体化する
  - 固定 1 ステージ
  - 同時行動
  - ゴール到達で早期終了可能
  - 宝箱取得による追加点
- AI 向け `visible_state` の shape を定義する
  - 少なくとも `turn`, `self`, `visible_tiles`, `known_goal`, `known_chests`, `scores`, `remaining_turns`
  - 視界半径は定数または ruleset 設定として調整可能にし、少なくとも 1 マス先 / 2 マス先の切替を想定できる形にする
- game master 向け `full_state` / `exported_snapshot` の最小 shape を定義する
  - マップ識別子
  - `rng_seed`
  - プレイヤー位置
  - 未取得宝箱
  - 現在スコア
- MVP の action schema を定義する
  - `move`
  - `wait`
- 競合解決規則を定義する
  - プレイヤー同士は同一タイルへ同時に移動できる前提とする
  - 宝箱に同時到達した場合は、その turn に到達したプレイヤーで宝箱点を等分する
  - 同一 tile 共有を前提に、後続の PvP / 協力ルールを上に積める状態遷移にする
- 視界制限の最初の形を定義する
  - 全体マップを見せず、半径ベースの局所視界を初期採用候補とする
- ゴール順位ボーナス規則を定義する
  - 同順位は competition ranking とし、同率 1 位が 2 人なら両者に 1 位点を与え、次順位は 2 位点ではなく 3 位点を与える

### `docs/specs/game-master.md`

- game master が match 初期化時に `rng_seed` を受け取り、snapshot / exported snapshot に保持できることを、dungeon game の利用例として補足する

### `docs/specs/platform.md`

- dungeon game の検証では Go subprocess bot を開発用 reference path として使うが、正式提出経路は WASM のまま維持することを補足する

## Expected Code Changes

### non-internal dungeon package tree

- 新規 top-level package tree を追加する
  - 例: `games/dungeon/...`
- この tree 配下に少なくとも以下を置く
  - game metadata / ruleset 定義
  - state / action / snapshot 型
  - fixed-map data
  - turn resolution
  - scoring
  - reference bot の判断ロジックで共有する domain 型
- この tree から `internal/...` を import しない

### game master entrypoint

- `cmd/dungeon-gamemaster/` を追加する
- local subprocess JSON-RPC 契約で動く dungeon game master を成立させる

### local reference bot

- `cmd/dungeon-bot-local/` を追加する
- 固定マップで継続的にゴール到達または宝箱回収ができる最低限の探索 bot を実装する
- 後続の WASM 版と共有する判断ロジックを切り出せる形で実装する

### runner / registry integration

- dungeon game を registry に登録する
- `arena-runner` から fixed-map dungeon match を起動できるようにする

## Verification

- `go test ./...`
- local subprocess の game master と local subprocess bot で dungeon match を完走できる
- 同一入力で deterministic な結果が得られる
- helper コマンドで人間が最短経路の確認を再現できる
- ダンジョン固有 package tree が `internal` import を持たない

## Sub-tasks

- [ ] ダンジョン固有コードの配置方針を決める
- [ ] fixed-map MVP ルールを spec に固定する
- [ ] `visible_state` / `full_state` / `exported_snapshot` の shape を定義する
- [ ] 最初の競合解決と採点ルールを定義する
- [ ] 視界半径の設定方法を定義する
- [ ] local subprocess dungeon game master を追加する
- [ ] [parallel] fixed-map data と turn resolution を実装する
- [ ] [parallel] local reference bot の最小行動ロジックを追加する
- [ ] registry / runner integration を追加する
- [ ] helper と targeted verification を追加する

## Parallelism

- fixed-map data と reference bot の行動ロジックは、payload shape が固まれば並行で進められる
- helper 追加と targeted verification 追加は、match 完走経路ができた後に並行で進められる

## Risks and Mitigations

- 最初からダンジョン要素を入れすぎると、platform 問題と game 問題の切り分けが崩れる
  - mitigation: 最初は固定 1 ステージ、`move` / `wait`、ゴールと宝箱だけに絞る
- `internal/games/...` に寄せると、後続の別 repo 切り出しが難しくなる
  - mitigation: top-level non-internal package tree を採用し、platform 側は adapter と registry だけを持つ
- 開発用 subprocess bot と正式 WASM bot が別物になる
  - mitigation: bot の判断ロジックは shared package に置き、entrypoint だけ分ける
- 視界制限を後回しにすると payload と bot の設計をやり直しやすい
  - mitigation: 最初の MVP から単純な局所視界を採用し、情報制約の軸を先に固定する
- 同一 tile 同居を禁止すると、後続の PvP / 協力追加時に occupancy モデルを作り直しやすい
  - mitigation: 最初から multi-occupancy 前提で state と resolution を設計する

## Design Decisions

- Phase 5 の最初の実装単位は「固定マップ + ゴール到達 + 宝箱回収」の MVP とする
- ダンジョン固有コードは top-level non-internal package tree に置く
- 開発中の最速確認経路は Go subprocess bot とする
- WASM reference AI は後続 plan で成立させるが、判断ロジック共有を前提に最初から package を分ける
- 初期の turn loop は同時行動で統一する
- 初期の視界モデルは、後戻りを避けるため局所視界を採用候補とし、視界半径は調整可能な定数または ruleset 設定で持つ
- プレイヤー同士は同一マスに共存できる
- 宝箱の同時取得は到達者で等分する
- ゴール順位点は competition ranking で配る
