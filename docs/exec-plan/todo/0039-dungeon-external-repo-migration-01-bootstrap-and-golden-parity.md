# dungeon-external-repo-migration-01-bootstrap-and-golden-parity
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`dungeon-game-ai-arena` を dungeon game の開発場所として立ち上げ、
現在 ai-arena で保持している dungeon game master / domain / golden verification を
仕様変更なしで別 repo へ移すための最初の成立点を作る。

この plan の完了条件は、独立 repo 側で現在と同じ golden がそのまま通り、
local 起動から完了までと CI e2e が問題なく成立することに置く。
この段階では ai-arena 側の dungeon 実装は削除しない。

親 plan:

- `0038-dungeon-sidecar-boundary.md`

## Scope

- `dungeon-game-ai-arena` repo の最小 bootstrap
- `cp` 主体での portable dungeon file 群の移設
- 同一 golden を使った local verification 導線の移設
- 同一 golden を使った CI e2e 導線の移設
- ai-arena 側を source-of-truth とした parity 確認

この plan では以下は扱わない。

- ai-arena 側からの dungeon 実装削除
- ai-arena module への version tag 付与
- `game_version` / `ruleset_version` / golden expectation の変更
- trusted external game backend 向けの新 transport
- `cp` では説明できない大規模再設計

## Spec Changes

### `docs/specs/dungeon-game.md`

- dungeon game の開発場所を external repo へ移しても、portable sidecar boundary と
  payload / golden contract は不変であることを明記する
- 移行完了の判定を「現行 golden が同じまま external repo の local と CI e2e で通ること」として明記する
- `cp` 主体での物理移設を前提にし、移行中は content rewrite より file move / import path 置換を優先する方針を書く

### `docs/specs/game-master.md`

- `github.com/yoskeoka/ai-arena/gamemaster` を使う external repo sidecar でも
  JSON-RPC 2.0 + NDJSON stdio transport 契約は変わらないことを補足する

## Expected Code Changes

### `dungeon-game-ai-arena` repo

- `go.mod` と最小 build/test entrypoint
- `cmd/dungeon-gamemaster`
- `games/dungeon/...`
- golden / e2e / fixture / testdata / CI wiring
- local run から completion までを実行する command / Make target / script の最小配線

### `ai-arena` repo

- parity 比較に必要な import path / artifact path / docs の最小調整
- 移設対象 file 群を `cp` で扱えるようにする source list または手順整理

execution では、移設作業の主経路を以下の順に固定する。

1. copy source set を列挙する
2. `cp` で新 repo へ複製する
3. module path と import path の必要最小限の差分だけ直す
4. local golden を通す
5. CI e2e を通す

## Verification

この plan の execution PR は、少なくとも以下を満たしたとき完了とする。

- `dungeon-game-ai-arena` で現行 dungeon golden が仕様変更なしで通る
- `dungeon-game-ai-arena` で local 起動から completion までの確認が通る
- `dungeon-game-ai-arena` の CI で同じ golden を使う e2e が通る
- ai-arena 側の dungeon verification も引き続き通り、比較元として残っている
- parity 差分が出た場合、その差分が ruleset / payload 変更ではなく移設不備として扱われる

## Sub-tasks

- [ ] external repo へ `cp` で持ち出す portable file 群を列挙する
- [ ] `dungeon-game-ai-arena` の最小 module / CI / local command skeleton を作る
- [ ] [parallel] dungeon game master / domain / fixture 群を `cp` 主体で移設する
- [ ] [parallel] same-golden local verification 導線を移設する
- [ ] [depends on: dungeon game master / domain / fixture 群を `cp` 主体で移設する, same-golden local verification 導線を移設する] same-golden CI e2e を移設する
- [ ] [depends on: same-golden CI e2e を移設する] ai-arena 側との parity を確認し、削除前 gate を記録する

## Parallelism

- portable file copy と local verification wiring は、copy source set が固まれば並行で進められる
- CI wiring は local verification が一度通った後に独立して詰められる
- ai-arena 側の削除作業はこの plan と並行しない

## Risks and Mitigations

- copy 対象の見落としで external repo 側だけ hidden dependency が残る
  - mitigation: `cmd/dungeon-gamemaster` から辿る portable package 群を先に列挙し、`cp` source set を明文化する
- 移行中に golden を更新してしまい parity 判定が曖昧になる
  - mitigation: この plan では golden 更新を禁止し、差分はまず移設不備として扱う
- file 内容を広く読み替えながら移すと boundary が再び崩れる
  - mitigation: content rewrite は module/import/CI に必要な最小限に限定し、主経路は `cp` と path 置換にする
