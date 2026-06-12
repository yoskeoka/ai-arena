# prebuilt-preset-bootstrap
**Execution**: Use `/execute-task` to implement this plan.

## Objective

staging / production の preset queue bootstrap を、`go run` 前提の repo source 実行から
build-time に生成した prebuilt AI artifact 実行へ切り替える。
最初の到達点は、current remote lane の `init failed for p1: invalid-timeout` を
`ARENA_SERVICE_PRESET_CONFIG=./config/platform-service/presets.example.json` 起因の
runtime startup 依存として解消し、Phase 6 の staging verification を再び回せる状態へ戻すことに置く。

## Context

- current staging backend は `ARENA_SERVICE_PRESET_CONFIG=./config/platform-service/presets.example.json`
  を読み、`echo-ai-2turn` preset を queue へ積んでいる
- `echo-ai-2turn.arena.json` は `runtime.command = ["go", "run", "./testdata/ai/echo/echo-ai"]`
  を使っており、remote runtime で source checkout と `go run` 実行を前提にしている
- current staging failure では `Turn 0` の `session_initialized` が
  `FailureReason = "invalid-timeout"` になり、`p1-stderr.log` も空だったため、
  AI logic ではなく bot main 到達前の startup path が疑わしい
- user decision として、いま優先したいのは general AI submission を先に進めることではなく、
  preset bootstrap を remote deploy shape で安定化し、staging / production verification を
  続けられるようにすることにある
- memory / existing docs でも `ARENA_SERVICE_PRESET_CONFIG` は temporary bootstrap 扱いであり、
  長期的な matchmaking / scheduler の正本にしてはならない

## Scope

- remote bootstrap 専用の prebuilt preset catalog を定義する
- Render build/start contract に preset bot artifact build と配置を組み込む
- staging / production で使う `ARENA_SERVICE_PRESET_CONFIG` の指し先を
  source-oriented example から remote-oriented catalog へ切り替える
- local deploy-shaped verification でも同じ prebuilt preset lane を再現できるようにする
- current preset bootstrap が temporary lane であることを docs で明確に保つ

この plan では以下を扱わない。

- general AI submission / registration product flow の実装
- preset bootstrap 自体の恒久化
- scheduler / matchmaking が queue を組み立てる本来の long-term flow
- new game / new AI catalog の大規模拡張

## Options Considered

### Option A: `AI_ARENA_INIT_ACK_TIMEOUT` をさらに延長して `go run` preset を延命する

- 利点: 変更箇所が少ない
- 欠点: startup 成功条件が runtime toolchain / cache / source checkout に依存したまま残る
- 欠点: staging failure の切り分けが弱く、remote verification lane が再び不安定化しやすい

### Option B: general AI submission まで先に進めて preset bootstrap を飛ばす

- 利点: long-term architecture に近い
- 欠点: scope が大きく、staging verification の unblock まで遠い
- 欠点: current release flow の acceptance lane が壊れたままになる

### Recommended: Option C: prebuilt preset catalog を remote bootstrap 用に導入する

- 利点: current operator flow / release flow を大きく変えずに remote lane を安定化できる
- 利点: `go run` / toolchain download / source-path drift を remote runtime から外せる
- 利点: preset bootstrap が temporary lane である前提も維持しやすい

## Spec Changes

### `docs/development/platform-service-online-deploy.md`

- remote bootstrap では source-oriented `presets.example.json` を直接使わず、
  prebuilt AI artifact を参照する remote preset catalog を使うことを明記する
- `make render-build` の責務に backend binary だけでなく required preset artifact preparation を含める
- staging / production の `ARENA_SERVICE_PRESET_CONFIG` がどの catalog を指すべきかを明記する

### `docs/specs/platform-service-operator-api.md`

- first remote lane の preset catalog は server-known config のままでよいが、
  remote deploy shape では source execution ではなく deploy artifact と整合した participant reference を
  指さなければならないことを補足する

### `docs/specs/ai-runtime.md`

- deploy-shaped `local-subprocess` participant は source checkout 上の ad-hoc `go run` に依存せず、
  build/release lane が prepared artifact を供給してよいことを補足する

### `README.md` and/or relevant development docs

- local lightweight example と remote bootstrap example の役割差を明確にする
- `presets.example.json` は contributor-facing example であり、staging / production の canonical preset config ではないと明記する

## Expected Code Changes

- remote bootstrap 用 preset catalog file の追加
- prebuilt preset bot binary を生成して配置する build helper / Make target / script の追加
- `make render-build` から preset artifact preparation を呼ぶ wiring
- remote preset catalog が参照する sidecar manifest or runtime command の追加/更新
- deploy/local verification command が remote preset lane を再現できるようにする最小 wiring

## Sub-tasks

- [ ] current remote preset chain (`presets.example.json` -> `echo-ai-2turn` -> `go run`) を docs と code 上で固定し、差し替え target を明確にする
- [ ] remote bootstrap 専用 preset catalog の file name / location / ownership を決める
- [ ] prebuilt echo preset artifact を build する helper surface を決める
- [ ] [parallel] remote preset catalog と sidecar/runtime command を prebuilt artifact 向けに切り替える
- [ ] [parallel] `make render-build` と local deploy-shaped verification lane を new preset preparation に接続する
- [ ] staging / production env/runbook で `ARENA_SERVICE_PRESET_CONFIG` の canonical 値を更新する
- [ ] remote verification で `preset queue -> completed detail` が再び流れることを確認する

## Parallelism

- [parallel] preset catalog/manifest 側の差し替えと build helper surface の準備は並行できる
- [depends on: build helper surface] `make render-build` への組み込みを進める
- [depends on: remote preset catalog] staging / production env/runbook 更新と remote verification を進める

## Dependencies

- depends on: `0074-platform-online-foundation-03-04-matchmaking-ranking-follow-up-01-phase6-release-flow.md`
- informed by: `0082-platform-service-db-migration-release-flow.md`
- informed by: `arena-runner-e2e-fixture-prebuild-and-init-deadline.md`

## Risks and Mitigations

- prebuilt artifact path を repo/workdir 相対で雑に置くと、Render build output と start-time cwd がずれて再び runtime start failure を踏む
  - mitigation: build output location と preset catalog path resolution を runbook / helper / tests で同時に固定する
- remote bootstrap と local example の preset catalog を同一ファイルに無理に統合すると、contributor UX と deploy shape が再び衝突する
  - mitigation: example catalog と remote bootstrap catalog の役割を分け、どちらがどこで canonical か docs に明記する
- preset bootstrap を stable feature として書いてしまうと、temporary lane が長期契約化する
  - mitigation: docs で matchmaking / scheduler 移行前の temporary bootstrap と明記する

## Design Decisions

- current unblock は general submission ではなく prebuilt preset bootstrap を優先する
- remote lane は `go run` ではなく build-time prepared artifact を参照する
- example/local preset config と remote bootstrap preset config は分離してよい
- `ARENA_SERVICE_PRESET_CONFIG` 自体は temporary bootstrap seam として維持し、long-term architecture とは区別する

## Verification

- local deploy-shaped lane で prebuilt preset catalog を使って `preset queue -> completed detail` を再現できること
- staging backend で `Turn 0` init timeout が解消し、preset enqueue が terminal artifact 生成まで進むこと
- Render build/start contract だけで required preset artifact がそろい、runtime shell access なしでも再現可能であること
- updated docs が `presets.example.json` と remote bootstrap catalog の役割差、および `ARENA_SERVICE_PRESET_CONFIG` の temporary nature を明確に説明していること
