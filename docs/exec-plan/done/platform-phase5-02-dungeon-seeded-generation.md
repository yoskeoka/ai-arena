# platform-phase5-02-dungeon-seeded-generation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

fixed-map MVP の次の段階として、seed 付きのダンジョン生成、スタート/ゴール/宝箱配置、宝箱スコア割当を deterministic に行えるようにする。match ごとの完全再現性を保ちつつ、ランダム性でゲーム性が崩れすぎない scoring contract を先に固定する。

depends on:

- `platform-phase5-01-dungeon-fixed-map-mvp.md`

## Scope

- 迷路生成アルゴリズムの導入
- start / goal / chest placement の deterministic 化
- `rng_seed` の注入・保持・export
- 宝箱スコアの固定組合せからの割当ルール導入
- replay / exported snapshot からの完全再現性確認

この plan では以下は扱わない。

- モンスターや戦闘
- 複数ステージ
- online persistence backend
- WASM reference AI

## Spec Changes

### `docs/specs/dungeon-game.md`

- `rng_seed` を match 初期条件の必須項目として定義する
- `full_state` / `exported_snapshot` に `rng_seed` と生成結果を保持する契約を追加する
- 迷路生成規則を定義する
  - NxM グリッド
  - 通路幅 1
  - 全通路連結保証
- start / goal / chest placement の規則を定義する
- 宝箱スコアの割当を「固定スコア集合を shuffle して配置する」方式として定義する
- 再現性検証時に seed だけで同じ初期状態を復元できる条件を明記する

### `docs/specs/platform.md`

- replay / debug の観点で exported snapshot に seed を含める理由を補足する

## Expected Code Changes

### dungeon generation

- `games/dungeon/generation` または同等 package を追加する
- perfect maze を生成できるアルゴリズムを実装する
- start / goal / chest placement を deterministic に行う

### snapshot / export contract

- game master が `rng_seed` を保持し、snapshot / exported snapshot / result から参照できるようにする
- replay 用の初期化入口で seed を優先入力として扱えるようにする

### scoring expansion

- 宝箱スコア集合と到達順位ボーナスの合算ロジックを実装する
- score table 自体は game 固有 package に閉じ込める

## Verification

- `go test ./...`
- 同一 `rng_seed` で複数回実行して同じ map / placement / score allocation になる
- 異なる `rng_seed` で初期状態差分が生まれる
- exported snapshot に含まれる `rng_seed` から初期状態を再構成できる
- 宝箱スコアが固定集合からの割当になっており、極端な偏りが game design 上抑えられている

## Sub-tasks

- [ ] 迷路生成アルゴリズムを選定し spec に固定する
- [ ] `rng_seed` 注入・保持・export 契約を追加する
- [ ] start / goal / chest placement を deterministic にする
- [ ] 宝箱スコアの固定集合と割当方式を定義する
- [ ] generation 実装と replay verification を追加する
- [ ] [parallel] exported snapshot / debug helper を更新する

## Parallelism

- scoring table の設計と exported snapshot 契約の更新は並行で進められる
- generation 実装と replay helper 更新は seed contract が固まれば並行で進められる

## Risks and Mitigations

- 乱数で直接スコア値まで広く揺らすと、ゲーム性が run ごとにぶれやすい
  - mitigation: スコア値自体は固定集合にし、配置だけを seed で変える
- 迷路生成アルゴリズムを複雑にしすぎると、再現性検証や bot 開発が遅くなる
  - mitigation: 最初は perfect maze を 1 つ生成する単純な手法を採用し、地形多様化は後続に送る
- seed が exported snapshot に残らないと、観測できた試合の完全再現ができない
  - mitigation: snapshot / exported snapshot / result の全経路で seed を保持する

## Design Decisions

- 最初の迷路生成は perfect maze を対象とする
- 初回採用候補のアルゴリズムは recursive backtracker とする
- `rng_seed` は game master 起動時に注入し、初期状態の正本として保持する
- 宝箱スコアは固定集合からの割当方式を採用し、スコアレンジ自体は乱数で広げない
