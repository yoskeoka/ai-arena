# echo-shuffle-ruleset
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`echo-count` fixture に seed-aware な shuffle ruleset を追加し、`arena-runner` の
`rng_seed` 注入・snapshot/record からの seed 復元・`--rng-seed` conflict rejection を、
dungeon 実装に依存せず ai-arena platform 単体で継続検証できる状態にする。

depends on:

- `platform-phase2-02-fixture-e2e.md`
- `arena-runner-artifact-io-contract.md`

## Scope

- `echo-count` に shuffle 結果を game master だけが保持する seed-aware ruleset を追加する
- `rng_seed` を含む `echo-count` snapshot / record contract を fixture appendix に反映する
- same-seed rerun と snapshot/record seed source-of-truth を `arena-runner` e2e で固定する
- dungeon 依存の `rng_seed` conflict coverage を `echo-count` ベースへ置き換える

この plan では以下は扱わない。

- `echo-count` を独立ゲームとして拡張すること
- AI player の action schema や挙動の変更
- `janken` や他 game への seed-aware contract 横展開
- replay/debug CLI surface 自体の再設計

## Spec Changes

### `docs/specs/platform.md`

- `echo-count` fixture appendix に shuffle ruleset を追加する
- `rng_seed` を受け取るのは game master であり、AI player へ shuffle 結果や seed を露出しないことを明記する
- seed-aware ruleset の `expected` 列は `rng_seed` による deterministic shuffle で決まり、
  同一 `player` 順・同一 `rng_seed` なら同じ turn progression になることを明記する
- `snapshot.game_state.rng_seed` を source of truth として replay/debug entrypoint から再投入できることを、
  fixture appendix の具体例として補足する

## Expected Code Changes

### `internal/games/echo/`

- shuffle ruleset 名、turn count、seed-aware expected sequence 生成を追加する
- `Config` と fresh-run / snapshot-resume 経路で `rng_seed` を受け取り、game master 内部 state に保持する
- `snapshot.game_state` に `rng_seed` と shuffled expected sequence、またはそれを再構成できる最小 state を保持する
- `NewFromSnapshot` と `SnapshotFromHistory` が seed-aware ruleset を復元できるようにする

### runner / registry / replay wiring

- `internal/platform/registry/defaults.go` で `echo-count` build spec から `rng_seed` を `echo` game master へ渡せるようにする
- 必要なら `cmd/arena-runner` 周辺 test helper の `rng_seed` assertion を `echo-count` fixture に合わせて更新する

### `e2e/` and fixture data

- dungeon 依存の `rng_seed` conflict tests を `echo-count` shuffle ruleset ベースへ置き換える
- 小さい handcrafted `snapshot.json` / `record.json` fixture を追加し、`game_state.rng_seed` を含む path を固定する
- same-seed rerun で normalized result shape が一致する e2e を追加する
- 必要なら different-seed で expected sequence が変わることを確認する narrow e2e を追加する

## Verification

完了は unit test と `arena-runner` e2e で判定する。最低限、以下を機械的に確認できること。

- `arena-runner --game echo-count --ruleset <shuffle-ruleset> --rng-seed <seed>` で match を完走できる
- 同一 `rng_seed`・同一 player 順で 2 回実行したとき、normalized result shape が一致する
- handcrafted `snapshot.json` / `record.json` が `game_state.rng_seed` を含み、`--rng-seed` override を reject する
- `--snapshot-input` / `--record-input` 指定時に seed が source of truth として復元される
- 既存の non-seeded `echo-count` ruleset coverage が壊れていない

想定 verification コマンド:

- `go test ./internal/games/echo ./internal/platform/registry ./cmd/arena-runner ./e2e/...`

## Sub-tasks

- [ ] `docs/specs/platform.md` の fixture appendix に shuffle ruleset と seed contract を追加する
- [ ] `internal/games/echo/` に seed-aware shuffle ruleset と snapshot/history 復元を実装する
- [ ] [depends on: `internal/games/echo/` に seed-aware shuffle ruleset と snapshot/history 復元を実装する] registry/build path から `rng_seed` を `echo` game master へ渡す
- [ ] [parallel] handcrafted `snapshot.json` / `record.json` fixture を追加する
- [ ] [depends on: registry/build path から `rng_seed` を `echo` game master へ渡す, handcrafted `snapshot.json` / `record.json` fixture を追加する] `arena-runner` e2e を `echo-count` shuffle ruleset ベースへ更新する
- [ ] [depends on: `arena-runner` e2e を `echo-count` shuffle ruleset ベースへ更新する] targeted unit/e2e を実行し、same-seed rerun と conflict rejection を確認する

## Parallelism

- spec update と handcrafted fixture 作成は並行できる
- `echo` game 実装と fixture file 準備は独立に進められる
- e2e の置換は seed-aware `echo` 実装と fixture file の両方が揃ってから行う

## Risks and Mitigations

- `echo-count` に random 要素を足しすぎると `janken` の責務境界を侵食する
  - mitigation: random は turn ごとの `expected` 列 shuffle のみに限定し、AI action schema と scoring は既存のまま据え置く
- `snapshot.game_state` に shuffled 列を持ちすぎると fixture が冗長になる
  - mitigation: replay/debug に必要な最小 state だけを保持し、golden ではなく compact assertion で検証する
- different-seed assertion を広く取りすぎると PRNG 実装差分に弱くなる
  - mitigation: primary gate は same-seed determinism と source-of-truth conflict rejection に置き、different-seed は narrow fixture に限定する

## Design Decisions

- shuffle 結果は game master だけが保持し、AI player へ事前公開しない
- `rng_seed` は `echo-count` shuffle ruleset の game master 初期条件として受け取り、snapshot/record から復元できる source of truth とする
- seed-aware coverage は新 game を増やさず `echo-count` fixture appendix の ruleset 追加で閉じる
