# dungeon-external-repo-migration-03-ai-arena-removal
**Execution**: Use `/execute-task` to implement this plan.

## Objective

external repo 側で same-golden local / CI e2e、fixture bot / AI player verification、
tagged SDK import、dungeon 用 CI coverage の移行が成立した後に限り、
ai-arena から dungeon game 実装を削除し、platform repo は SDK と runner contract の提供側へ役割を絞る。

親 plan:

- `0038-dungeon-sidecar-boundary.md`

depends on:

- `0039-dungeon-external-repo-migration-01-bootstrap-and-golden-parity.md`
- `0040-dungeon-external-repo-migration-02-sdk-tag-and-import-contract.md`
- `0042-dungeon-external-repo-removal-gate-verification.md`

## Scope

- ai-arena repo から dungeon-specific code / fixture / registration / docs の削除
- ai-arena runner から external repo の game development を起動する置換経路確認
- ai-arena 側 spec を external repo ownership に合わせて更新
- 削除後も ai-arena runner と public SDK が external game development を支える形へ整理

この plan では以下は扱わない。

- external repo 側の新機能追加
- `v0.1.0` より先の SDK 拡張
- dungeon ruleset / payload schema の変更
- parity 成立前の先行削除

## Spec Changes

### `docs/specs/dungeon-game.md`

- file を完全に削除する
- dungeon 固有仕様の canonical source は `dungeon-game-ai-arena` 側へ移す

### `docs/specs/platform.md`

- ai-arena platform repo が external game development を支える責務と、
  dungeon 固有実装を保持しない状態を反映する

### `docs/specs/game-master.md`

- game master 開発者向け導線を external repo 前提へ更新する

## Expected Code Changes

- `cmd/dungeon-gamemaster`
- `cmd/dungeon-bot-local`
- `cmd/dungeon-map-helper`
- `games/dungeon/...`
- `testdata/ai/dungeon/...`
- dungeon fixture / golden / e2e assets
- `internal/platform/registry` などの dungeon 既定登録
- dungeon を前提にした docs / examples / test references

削除は dependency order を守って進める。

1. external repo parity / fixture / WASM / CI migration と tagged import の証跡を固定する
2. ai-arena runner から external repo の dungeon development を起動できる置換経路を確認する
3. ai-arena から dungeon registration と verification entrypoint を落とす
4. ぶら下がる dungeon package 群を削除する
5. spec / docs / examples を cleanup する

## Verification

この plan の execution PR は、少なくとも以下を満たしたとき完了とする。

- ai-arena repo 内に dungeon game 実装が残っていない
- ai-arena 側の build/test が dungeon code なしで通る
- `docs/specs/dungeon-game.md` が ai-arena repo から削除されている
- ai-arena docs/specs が external repo ownership と一致している
- external repo 側で same-golden local / CI e2e、fixture bot verification、Go-WASM verification、
  Rust AI player verification、tagged import、dungeon 用 CI coverage の証跡が参照できる

`0042` 完了後に参照する gate 証跡:

- ownership / removal gate: `dungeon-game-ai-arena/docs/specs/dungeon-external-sdk-consumption.md`
- dungeon Go-WASM CI: `dungeon-game-ai-arena/.github/workflows/wasm-verification.yml`
- host runtime Rust-WASM coverage: `ai-arena/.github/workflows/wasm-verification.yml`

## Sub-tasks

- [ ] 0039、0040、0042 の完了証跡を確認し、削除 gate を閉じる
- [ ] ai-arena runner から external repo の dungeon development を起動する置換経路を確認する
- [ ] [depends on: ai-arena runner から external repo の dungeon development を起動する置換経路を確認する] dungeon registration / runner wiring を ai-arena から外す
- [ ] [parallel] dungeon package / command / fixture / golden assets を削除する
- [ ] [parallel] spec / docs / examples を external repo ownership に更新し、`docs/specs/dungeon-game.md` を削除する
- [ ] [depends on: dungeon package / command / fixture / golden assets を削除する, spec / docs / examples を external repo ownership に更新する] ai-arena build/test を実行し、削除後整合を確認する

## Parallelism

- code asset 削除と docs/spec cleanup は、削除 gate が閉じた後に並行できる
- build/test verification は両方の cleanup が揃ってから行う

## Risks and Mitigations

- parity 成立前に ai-arena 側を消して rollback が難しくなる
  - mitigation: 0039、0040、0042 を hard gate にし、same-golden 以外の verification/CI migration 証跡確認前は削除しない
- docs だけ external repo ownership へ寄って code 側 cleanup が残る
  - mitigation: registry / command / fixture / package deletion と `docs/specs/dungeon-game.md` 削除を同じ plan で束ねる
- ai-arena runner の external game development 導線まで誤って削る
  - mitigation: 削除前に external repo 起動の置換経路を確認し、削除対象を dungeon 固有実装に限定して `gamemaster` public surface と runner contract は残す
