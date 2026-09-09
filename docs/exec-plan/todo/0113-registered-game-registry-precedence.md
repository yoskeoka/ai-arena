# registered-game-registry-precedence
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

staging の operator match 作成で、upload・activation 済み Reversi game と互換な bot を選択しても
`unsupported game "reversi"` となる不整合を解消する。online service の match admission、queue、worker は、
operator が activation 時に固定した game artifact digest を同じ process-local registry で exact に解決し、
外部 admission 済み release を built-in より優先する。

完了時、artifact-backed competition scope から作成した match は、manifest と一致する exact game
artifact で admission・queueing・実行される。digest が未 admission、game ID / exact game version が
scope metadata と異なる、または ruleset が descriptor にない場合は queue record を作らず拒否する。
artifact identity を持たない legacy / local built-in match は、external release が選べない場合にのみ
built-in を使い、既存の built-in test fixture と standalone runner の挙動は維持する。

## 背景と確定方針

`cmd/arena-service/main.go` の `newCLIApp` は `NewDefaultAdmissionValidator(nil, dryRun)` を
artifact runtime / `NewWASIOverlay` の構築より前に実行している（現行 252-285 行）。nil の validator
registry は `registry.Default()` に置換されるため、built-in の echo / janken しか解決できない。
その後に作る `admissionRegistry` は game bundle admission、activation、worker には渡されるが、
`CommandService.Submit` の admission には渡らない。

実行計画では以下を確定方針とする。

1. artifact digest を含む `MatchSubmission` は、その digest を artifact registry で exact lookup する。
   その結果の descriptor は submission の game ID、exact game version、ruleset と一致しなければ
   admission を拒否し、version lookup や built-in への fallback をしない。
2. digest を含まない version lookup では、同一 `game_id + major` に外部 admission 済み release が
   一つでもあれば、その集合内で最大 semver を選ぶ。外部 release が一つもないときだけ built-in
   release 集合を通常どおり解決する。これは upload artifact を先に選ぶための policy であり、
   active scope が digest を snapshot することを置き換えない。
3. `arena-service` は artifact runtime、overlay registry、validator の順に構築し、admission、
   game upload / activation、worker に同一 overlay instance を注入する。

この優先順位は依頼で明示されたプロダクト方針であり、新しい ADR を必要とする未決定の設計選択は残さない。
`registry.Default()` を使う standalone `arena-runner` / replay convenience API、fixture、nil を許す
下位 constructor は、external artifact store / activation context を持たない built-in-only fallback として
変更しない。online service の production wiring では nil を渡さない。

## Black-box 仕様変更

- `docs/specs/platform-game-registry.md` を更新し、online service registry の lookup precedence を定義する。
  external admission 済み release は built-in より先に選ばれ、artifact digest がある match は exact
  identity のみを使う。exact lookup 失敗・metadata 不整合時に version / built-in fallback はしない。
- `docs/specs/platform-service-general-submission.md` を更新し、scope が snapshot した selected game
  artifact と match admission の整合性を明記する。game ID、exact version、ruleset のいずれかが
  descriptor と不一致なら queue 保存前に拒否する。HTTP / TypeSpec の field 変更はない。

## 既存実装の参照

- `cmd/arena-service/main.go`
  - `newCLIApp`（現行 249-355 行）: validator が default registry を捕捉してから WASI overlay を作る
    順序、および service component への registry 注入点。
  - `cliApp.newWorker`（現行 476-484 行）: worker には `cliApp.registry` を渡している既存の正しい経路。
- `internal/platform/service/admission.go`
  - `NewDefaultAdmissionValidator` / `Validate`（現行 25-59 行）: nil fallback と version-only admission
    lookup。artifact identity を優先する共通 admission 境界。
- `internal/platform/service/request.go`
  - `MatchRequestService.Create`（現行 115-178 行）: activated game の `GameArtifactID` を submission に
    snapshot して `CommandService.Submit` に渡す経路。
- `internal/platform/service/worker_local.go`
  - `NewLocalRunnerInvoker` / `Run`（現行 37-87 行）: production では injected overlay を受け、
    `GameArtifactID` がある実行を exact lookup する既存の正しい経路。
- `internal/platform/registry/registry.go`
  - `Registry.Lookup`、`LookupVersion`、`LookupArtifact`（現行 106-160 行）: store lookup と
    descriptor resolution の共通入口。
- `internal/platform/registry/store.go`
  - `InMemoryStore.Lookup` と latest-release selection（現行 46-60 行および末尾 helper）: 同じ key の
    release 群を semver だけで混在選択している現在の precedence 欠落点。
- `internal/platform/registry/wasm_resolver.go`
  - `NewWASIOverlay` / `modeResolver`（現行 79-120 行）: built-in clone と writable WASI record を
    合成する service registry。external-first の tier をここで構成する。
- `internal/platform/service/general.go`
  - `registerArtifactBackedGame`（現行 204-277 行）: activation が既に `LookupArtifact` と manifest /
    descriptor identity を exact に照合する先行契約。
- `internal/platform/service/artifact_submission_e2e_test.go`
  - `runArtifactSubmissionProof`（現行 57-184 行）: filesystem / S3-compatible bundle store を通す
    game upload -> activation -> AI upload -> request -> worker の回帰 proof。
- `internal/platform/registry/registry_test.go`
  - `TestInMemoryStoreLookupSelectsLatestReleaseWithinMajor`（現行 51-69 行）: tier 内の最大 semver
    選択を維持する既存の基準。

## 変更マップ

- `docs/specs/platform-game-registry.md` (MODIFY): online service の external-admitted / built-in
  lookup precedence と digest exact-resolution の observable contract を追加する。
- `docs/specs/platform-service-general-submission.md` (MODIFY): activated scope が固定した game artifact
  identity と admission rejection boundary を追加する。
- `internal/platform/registry/store.go` (MODIFY): external admitted release tier と built-in fallback tier を
  区別して lookup し、tier 内だけで最大 semver を選ぶ store / overlay composition を提供する。
- `internal/platform/registry/wasm_resolver.go` (MODIFY): `NewWASIOverlay` を external-first tier と
  built-in fallback resolver の合成へ変更し、artifact exact lookup を writable external tier に限定する。
- `internal/platform/service/admission.go` (MODIFY): `GameArtifactID` がある submission を exact lookup し、
  resolved descriptor と submission metadata の一致・ruleset を queue 前に検証する。
- `cmd/arena-service/main.go` (MODIFY): runtime と shared overlay を validator より先に構築し、
  validator、artifact admission、general service、worker に同一 instance を渡す。
- `internal/platform/registry/registry_test.go` (MODIFY): external-first、tier 内 semver、artifact exact
  lookup、built-in fallback の registry contract を固定する。
- `internal/platform/service/artifact_submission_e2e_test.go` (MODIFY): uploaded artifact-backed game の
  match request が admission を通り、queue に digest が固定され、worker が同じ digest を実行する
  regression proof を強化する。
- `internal/platform/service/admission_test.go` (NEW): exact digest lookup、metadata/ruleset mismatch と
  missing artifact の queue-before rejection を unit level で固定する。
- `cmd/arena-service/main_test.go` (MODIFY): production `newCLIApp` wiring が validator に shared WASI
  overlay を渡すことを、artifact-backed submission acceptance で回帰テストする。

## 実行サブタスク

1. 仕様を先に更新する。online service における external-first tier、artifact identity がある場合の
   fallback 禁止、legacy/built-in の適用条件を上記 black-box contract と一致させる。wire contract は不変である。
2. registry overlay を two-tier にする。external admission record は writable primary store に保持し、
   primary の該当 key があるときはその records の最大 semver を返す。primary に該当 key がない場合だけ
   cloned / read-only built-in records を使う。`LookupArtifact` は primary にある digest だけを exact 解決する。
   同じ game ID / major を持つ built-in を external release が shadow しても、external descriptor が
   必ず選ばれることを確認する。
3. admission validator を artifact-aware にする。`GameArtifactID != ""` なら `LookupArtifact` で
   descriptor を取得し、artifact ID、game ID、exact game version、ruleset を照合する。不一致または
   missing digest は dry-run / queue save より前に error とし、artifact ID が空の legacy submission だけが
   external-first version lookup を使う。
4. `newCLIApp` の construction order を変更する。artifact runtime と shared overlay の成功後に
   `dryRun.WithBundleStore` と validator を構築し、その overlay を command service に注入する。worker の
   existing injection、activation の exact manifest validation、API / TypeSpec shape は変更しない。
5. registry unit test、validator unit test、service artifact E2E、`newCLIApp` wiring regression を追加 /
   更新する。Reversi 固有の hard-code や game-name whitelist を追加せず、janken fixture 等の generic
   artifactで同じ contract を証明する。

## 依存関係と並行性

1 は 2 と 3 の前提である。2 と 3 は spec の確定後に並行可能であり、4 は 2 と 3 の public constructor /
lookup contract が固まってから行う。5 は 2-4 と並行で test scaffold を作れるが、最終 assertion は全変更後に
行う。DB migration、sqlc、TypeSpec regenerate、operator UI migration は不要であり、既存 scope / bot / queue
schema は変更しない。

## 検証

- registry focused tests で、同じ `game_id + major` に external admitted release と built-in release が
  共存しても external を選び、external tier 内は最大 semver、external がない key は built-in を選ぶことを確認する。
- validator focused tests で、artifact-backed submission は exact digest を用い、unknown digest、game ID /
  exact version / ruleset mismatch では dry-run / queue save を行わないことを確認する。
- filesystem と S3-compatible bundle store の両方で、game ZIP admission -> artifact-backed activation ->
  AI bundle / bot selection -> match request -> queued digest -> worker execution が成功することを確認する。
- `newCLIApp` regression で production composition が default-only validator を再導入しないことを確認する。
- `make test`、`make lint`、`git diff --check`、および plan PR に必要な workflow lint を実行する。manual staging
  acceptance は Reversi ZIP と同一 AI の2席で upload -> admission -> registration -> match creation を行い、
  queued / completed record の game artifact digest が admission response と一致することを観察する。これは
  `echo-reference` automated lane には追加しない。

## 変更しない境界

- Reversi 固有の built-in registry entry、game ID whitelist、manifest の手入力、manual artifact digest 指定は追加しない。
- AI bundle、bot ownership、scope schema、queue / retry / rerun の snapshot shape、operator API / TypeSpec、
  ranking、artifact retention / GC は変更しない。
- standalone runner と injected-registry replay API は external context を持つ caller が registry を注入できる
  現行境界を維持し、package-level default API を online-service registry に変換しない。
