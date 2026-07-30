# arena-runner-official-game-bundle-e2e
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0038-arena-runner-official-game-bundle-e2e.md`

## 目的と完了境界

外部 game repository が、release に載せる exact `arena-bundle/v1` ZIP を `arena-runner` へ直接渡し、
WASM/WASI game master と複数 AI の match を deterministic に完走できるようにする。同時に既存の built-in game、
`--game-master-manifest` による dev-only `local-subprocess` overlay、`--player` entry-path を破壊しない。

完了時には game master の入力 source を built-in / manifest overlay / game ZIP のいずれか一つから選べ、各 player は
`--player`（既存 entry）または `--player-bundle`（AI ZIP）を player ごとに混在して指定できる。ZIP は archive validator を
通した bytes から private temporary directory へ materialize して WASI で起動し、metadata 不整合・kind 不整合・重複
player ID・不正 ZIP は match loop 開始前に失敗する。fresh run と snapshot resume を実証し、WASI history reconstruction が
未実装である現状は明示的な failure として保つ（本 plan で replay engine を実装しない）。

## 契約変更

`docs/specs/platform.md` を更新し、次の observable runner contract を固定する。

- `--game-master-bundle <zip-path>` は game ZIP の local-only overlay input、`--player-bundle player_id=zip-path` は
  repeatable な AI ZIP input とする。
- game source は built-in flags、`--game-master-manifest`、`--game-master-bundle` の三択で相互排他的とし、ZIP manifest が
  metadata source of truth になる。AI source は player ごとに legacy entry と ZIP を混在できる。
- ZIP は `artifactbundle.Read` と同じ validation を通過した exact bytes だけを materialize し、game/AI とも filesystem /
  network capability を与えない WASI runtime で開始する。AI ZIP は `artifact_kind=ai`、game ZIP は `artifact_kind=game` を要求する。
- game bundle と sidecar は `game_id` 完全一致、game-version major 一致、ruleset 完全一致を match 開始前に満たす。AI bundle は
  v1 manifest が持つ `game_id` と game-version major を照合し、ruleset target を未宣言の artifact metadata から推測しない。
  selected ruleset は game bundle の宣言で検証する。全 player source を横断して player ID を一意にする。ZIP AI の record identity は immutable digest、legacy AI は既存の resolved
  `ai_id` を維持する。
- ZIP-backed game master の fresh run / snapshot resume は descriptor の既存 API を通す。history/record replay reconstruction
  はその descriptor が未対応なら fail-fast とし、結果 artifact の source-of-truth/replay 意味を偽装しない。

## 現状の根拠

- `cmd/arena-runner/main.go:31-40,80-150,204-342,424-488` は game source の built-in/manifest 分岐と、entry-path のみを
  受け取る `--player` session loading を持つ。
- `cmd/arena-runner/gamemaster_manifest.go:20-122` は manifest から dev-only `local-subprocess` descriptor を作るが、
  WASI/ZIP input を拒否する。
- `artifactbundle/bundle.go:72-138` は ZIP/manifest/WASM import を検証し SHA-256 digest を返す。
- `internal/platform/service/artifact_bundle_store.go:61-90` と `internal/platform/registry/wasm_resolver.go:29-76` は
  exact digest の materialization、private directory cleanup、WASI game session の既存参照実装である。
- `internal/platform/service/worker_local.go:96-157` は player 単位で `ArtifactID` と `ArtifactRef` を分岐しており、
  artifact/既存方式の AI 混在を runtime session contract が許容することを示す。
- `internal/platform/registry/wasm_resolver.go:54-56` は WASI descriptor の history reconstruction を未実装として明示している。
- `docs/issues/0038-arena-runner-official-game-bundle-e2e.md:3-47` が consumer-facing E2E surface の要求を記録している。

## 変更マップ

- `docs/specs/platform.md` (MODIFY)
  - runner の game/AI bundle input、source precedence/exclusivity、mixed-player compatibility、WASI replay limitation を定義する。
- `cmd/arena-runner/main.go` (MODIFY)
  - `--game-master-bundle` と repeatable `--player-bundle` を parse し、source selection、global player-ID validation、
    bundle/legacy session assembly を既存 match loop へ接続する。
- `cmd/arena-runner/gamemaster_bundle.go` (NEW)
  - validated game ZIP から temporary private directory と `wasm-wasi` `GameDescriptor` を構築し、shutdown 時 cleanup を所有する。
- `cmd/arena-runner/player_bundle.go` (NEW)
  - validated AI ZIP の metadata compatibility、digest identity、materialization、WASI session と cleanup を player ごとに扱う。
- `cmd/arena-runner/main_test.go` (MODIFY)
  - game ZIP × AI ZIP、game ZIP × legacy AI、manifest game × AI ZIP、同一 match 内の legacy/ZIP AI 混在、
    snapshot resume、fail-fast cases を black-box `run` tests で追加する。
- `cmd/arena-runner/testdata/` または test-only bundle builder (NEW/MODIFY)
  - production release archive schema に従う最小 WASI game/AI ZIP fixture を用意し、source tree/module path を直接渡さない。
- `docs/issues/0038-arena-runner-official-game-bundle-e2e.md` (DELETE)
  - implementation PR で acceptance 完了後に resolved local issue と matching plan を lifecycle policy に従って削除する。

## 実施項目

1. CLI source model を分離する。
   - game source を built-in registry、manifest overlay、bundle overlay の discriminated input として parse し、複数指定を
     flag validation で拒否する。bundle manifest の metadata を CLI flags で上書きしない。
   - player source を legacy entry と bundle entry の共通 player spec に正規化し、両 flag をまたぐ重複 ID を拒否する。

2. game ZIP overlay を既存抽象へ載せる。
   - bundle bytes を読み、validator、artifact kind、ruleset/player-count constraint、metadata compatibility を match 開始前に確認する。
   - module を `0700` private temporary directory へ materialize し、runtime args/memory budget と match-lifetime context を渡して
     WASI game master session を構築する。session shutdown/error path の両方で directory を cleanup する。
   - fresh / snapshot-resume descriptor build を実装し、history reconstruction は unsupported error を返す。

3. AI ZIP input を player session assembly へ載せる。
   - 各 bundle を validator 済み exact bytes から private directory へ materialize し、manifest runtime args/memory budget で
     WASI adapter を開始する。AI identity は bundle digest とする。
   - legacy `--player` は既存 `catalog.LoadEntry`、sidecar、local-subprocess/WASI behavior をそのまま使う。すべての session は
     共通 close path で shutdown と materialization cleanup を行う。

4. consumer-facing E2E を固定する。
   - ZIP game × ZIP AI（複数 AI）、ZIP game × legacy AI、manifest overlay game × ZIP AI、同一 match 内の ZIP/legacy AI
     mix をそれぞれ完走させ、標準 artifact layout の `result-summary.json`、`snapshot.json`、`exported-snapshot.json`、
     `record.json` を検査する。
   - ZIP metadata/kind mismatch、duplicate ID、game source conflict、invalid archive、incompatible AI game/version metadata を match loop 前に
     reject すること、temporary directories を確実に消すこと、WASI history replay が明確に unsupported であることを検証する。

## 依存関係・並行性

- depends on: merged artifact admission/runtime wiring (`#297`) と AI bundle worker dispatch (`#299`)。両者が導入した
  validator/materialization/session contract を runner-local consumer surface で再利用する。
- 先に spec と CLI input normalization を固定する。その後 game ZIP descriptor と AI ZIP session loader は並行実装できる。
  E2E matrix は両方が揃った後に追加する。
- `0107-phase7-artifact-submission-e2e-staging` の operated-service release proof は別 boundary であり、本 plan は external repo が
  tagged public runner を使う local/CI proof に限定する。

## AI 方式を混在する際の確認事項

混在そのものは session contract 上問題ない。match loop は player ごとに同じ `PlayerSession` と JSON-RPC stream を扱い、
service worker も既に `ArtifactID` と `ArtifactRef` を player 単位で分岐している。

注意点は方式差を隠さないことである。ZIP AI は immutable digest、既存 AI は sidecar が解決した `ai_id` になるため record 上の
identity 表現は異なる。また AI bundle v1 は ruleset target を必須 metadata として持たないため、ruleset は game bundle 側の
宣言と game master の init/turn contract で守る。ZIP は WASI capability 制限を強制できる一方、`local-subprocess` は開発用の運用前提に留まる。
したがって本 plan は、同じ match への混在を許可しつつ、各 player source の identity/runtime kind を input provenance として
検証可能にし、互換性・重複 ID・cleanup を共通 gate にする。公平性上 ZIP-only を必須にする official service policyは変更しない。

## 検証

- `go test ./cmd/arena-runner/...` の black-box matrix:
  - game ZIP × 2 AI ZIP
  - game ZIP × existing AI
  - existing manifest game × AI ZIP
  - one match 内の existing/ZIP AI mix
  - snapshot resume、invalid/mismatch/source-conflict/duplicate-ID fail-fast、temporary-directory cleanup
- `go test ./artifactbundle/... ./internal/platform/...` で validator、WASI runtime、registry/service materialization の既存契約を回帰確認する。
- `make lint` と workflow linter を実行する。
- 外部 consumer と同じ形の `go run ./cmd/arena-runner --game-master-bundle ... --player-bundle ...` invocation で、
  compact artifacts を `result-summary.json` → `exported-snapshot.json` / `snapshot.json` → `structured-log.ndjson` の順に確認する。

## リスクと緩和

- bundle overlay が service admission を迂回して official registration に見える
  - runner-local に限定し、registry/store を書かず、validator と runtime contract だけを reuse する。
- ZIP/legacy player mix が identity や sandbox の違いを曖昧にする
  - ZIP は digest identity、legacy は既存 `ai_id` を維持し、runtime kind/source ごとの検証を test matrix に含める。
- WASI game の replay 未対応を fresh run 成功で覆い隠す
  - snapshot resume と history reconstruction を区別し、後者は explicit unsupported としてテストする。
