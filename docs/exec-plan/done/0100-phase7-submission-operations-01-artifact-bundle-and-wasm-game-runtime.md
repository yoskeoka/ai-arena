# phase7-submission-operations-01-artifact-bundle-and-wasm-game-runtime
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

Phase 7 の artifact submission の最初の実装単位として、game bundle の
`arena-bundle/v1` contract、content-addressed storage、latest-release registry、
WASM/WASI game-master runtime を platform foundation として成立させる。

完了境界は、game ZIP を validate/pack でき、digest keyed local/S3 store と registry から
WASM game-master session を解決できることとする。AI bundle admission、full archive hardening、
service runtime wiring と worker dispatch は後続 plan に分離する。

## Artifact Decision

- AI と game は lifecycle / ownership / release cadence が異なるため、別 bundle とする。
- 各 bundle は root の `manifest.json` と、manifest が宣言する 1 個の WASM module だけを持つ
  versioned ZIP とする。
- manifest は execution metadata の正本とし、web form に game/protocol/runtime field を再入力させない。
- form/API は単一 `bundle` file を `multipart/form-data` で受け取る。
- client が local path、R2 locator、GitHub URL を `artifact_ref` として指定する contract は廃止し、
  server が SHA-256 を計算して opaque `artifact_id` を返す。
- storage key は digest から導出し、同一 bytes は deduplicate してよい。

別 upload の manifest + module は atomicity と release asset の同一性が弱く、form metadata + module は
artifact を自己記述不能にするため採用しない。ZIP は execute bit を必要としない WASM-only contract に
限定し、archive validation cost を固定 layout で抑える。

## Black-box Contract

共通 manifest は schema version、artifact kind、module path/size/SHA-256、runtime kind を持つ。

- AI bundle は `ai_id`、transport、`game_id`、exact `game_version`、`ruleset_version`、
  requested WASM memory limit を持つ。
- game bundle は `game_id`、exact semantic `game_version`、game-master protocol version と、
  ruleset ごとの `ruleset_version`、exact `player_count`、`max_active_bots_per_owner`、
  AI module/memory/turn deadline budget を持つ。
- official upload lane は AI/game とも `wasm-wasi` だけを受け付け、native command と symlink を拒否する。
- manifest request は platform/ruleset 上限を緩められず、上限超過は reject する。
- bundle validation は path traversal、absolute/backslash/NUL path、link/device、duplicate/case collision、
  undeclared entry、file count、compressed/uncompressed size、compression ratio、hash mismatch を拒否する。
- WASM magic/version、wazero compile、許可 import、network/filesystem capability 不付与を確認する。
- queue が受け取るのは mutable URL ではなく、validation 済み artifact identity/digest である。

field-level schema の正本は repo-owned JSON Schema と typed Go model に置き、HTTP upload/response は
TypeSpec を正本とする。`docs/specs/` は layout の意味、validation、storage/runtime behavior を説明する。

## Existing Implementation References

- `docs/specs/platform-game-registry.md`
  - dev overlay と official admission、WASM first policy, lines 55-190
- `docs/specs/platform-service-general-submission.md`
  - current path-based AI validation と binary upload deferred, lines 37-109
- `internal/platform/catalog/catalog.go`
  - current sidecar/runtime manifest と WASM resolver, lines 24-48, 104-126
- `internal/platform/service/admission.go`
  - local-path-only artifact admission, lines 85-128
- `internal/platform/runtime/wasm_wasi.go`
  - current WASI execution boundary, lines 34-89
- `internal/platform/service/artifact_s3.go`
  - S3/R2 object store and stable locator support, lines 1-90
- `internal/platform/registry/registry.go`
  - current descriptor/build constraints, lines 16-62
- `internal/platform/gamemaster/gamemaster.go`
  - in-process/local-subprocess session adapters, lines 43-64, 132-154
- `typespec/namespaces/shared.tsp`
  - current caller-supplied `artifact_ref` submission shape, lines 139-156

## Code Change Map

- `schemas/arena-bundle-v1.schema.json` (NEW)
  - language-neutral bundle manifest source of truth
- `artifactbundle/` (NEW)
  - public manifest models, ZIP reader, validation, digest, and safe materialization contract
- `cmd/arena-artifact/` (NEW)
  - external game repositories and CI can run the exact platform pack/validate contract
- `tools/dev/package-builtin-game-bundles.sh` (NEW)
  - echo / janken を official submission と同じ WASM bundle として生成し、upload / registry / worker の結合検証 fixture にする
- `docs/specs/platform-artifact-bundle.md` (NEW)
  - observable packaging, validation, storage, and execution behavior
- `docs/specs/index.md` (MODIFY)
  - route bundle schema lookup to JSON Schema/typed source
- `docs/specs/platform-game-registry.md` (MODIFY)
  - official WASM artifact-backed game descriptor and ruleset constraint behavior
- `docs/specs/platform-service-general-submission.md` (MODIFY)
  - remove caller path/URL submission as online contract and reference artifact identity
- `typespec/namespaces/shared.tsp` (MODIFY)
- `typespec/namespaces/operator/api.tsp` (MODIFY)
- `typespec/generated/openapi/operator/openapi.json` (MODIFY)
- `operator-ui/src/generated/operator-api/` (MODIFY)
  - multipart upload and artifact validation response contract
- `internal/platform/service/artifact_*.go` (MODIFY)
  - content-addressed put/read and filesystem/R2 parity
- `internal/platform/service/` artifact ingestion/materializer components (NEW)
  - bounded upload, validation, immutable persistence, worker cache/materialization
- `internal/platform/registry/` (MODIFY)
  - artifact-backed descriptor record/resolution、同一 `game_id + major` 内で semver 最大 release を通常 lookup する規則、richer ruleset constraints
- `internal/platform/gamemaster/` (MODIFY)
  - WASM/WASI game-master session using the existing logical JSON-RPC API
- `cmd/arena-service/` (MODIFY)
  - upload/store/materializer/runtime wiring and limits
- unit/integration/negative fixtures (NEW/MODIFY)
  - malicious archive, bad hash/schema/import, oversize, local/R2 parity, WASM game session

## Sub-tasks

- [x] JSON Schema、typed model、fixed ZIP layout、digest contract を foundation として固定する。
- [x] external repo が同じ contract を利用できる `arena-artifact validate` CLI を追加する。
- [x] echo / janken の WASM game bundle を生成する開発用 script を追加する。
- [x] filesystem と S3/R2 の digest keyed bundle store を追加する。
- [x] TypeSpec upload API、generated artifacts、Go handler を追加する。
- [x] WASM game-master session と artifact-backed registry descriptor を追加する。
- [ ] 0105: bounded reader、archive security negative matrix、wazero policy、service runtime wiring。
- [ ] 0106: AI bundle admission、worker dispatch、match digest pinning。
- [ ] 0107: local S3-compatible E2E と built-in/Reversi staging verification。

## Dependencies and Parallelism

- blocks: `0105-phase7-artifact-admission-hardening-and-runtime-wiring.md`
- blocks: `0106-phase7-artifact-backed-ai-submission-and-worker-dispatch.md`
- informs: `0107-phase7-artifact-submission-e2e-staging.md`
- informs: `reversi-ai-arena/docs/exec-plan/todo/0007-ai-arena-release-artifacts.md`
- blocks: `0101-phase7-submission-operations-02-registration-bot-ownership.md`

## Verification

- schema/Go model parity test
- archive security negative tests and fuzz/property coverage where practical
- WASM compile/import/resource-limit tests for both artifact kinds
- filesystem and local S3-compatible content-addressed round trip
- exact bundle bytes produce the same artifact ID across repeated uploads
- WASM game master and WASM AI can be materialized and started without host path input
- `go test` for affected packages, TypeSpec generate/verify, operator-ui build, workflow lint

## Risks and Mitigations

- archive ingestion が decompression bomb/path traversal surface になる
  - mitigation: streaming/bounded read、fixed entry allowlist、pre/post extraction limits、links rejection
- game master WASM adapter が AI runtime policy と混ざる
  - mitigation: runtime engine は再利用し、game-master logical session adapter と resource profile は分離する
- mutable locator を queue に残すと再現性が失われる
  - mitigation: accept 時点で artifact ID/digest を snapshot し、worker は digest だけを materialize する

## Design Decisions

- official Phase 7 artifact は AI/game とも WASM-only の separate versioned ZIP とする。
- manifest は bundle 内の technical source of truth、form は control-plane metadata のみを所有する。
- artifact bytes は immutable/content-addressed とし、client-supplied path/URL を online contract にしない。
- game registry は `game_id + game_version major` を安定 lookup key とし、同一 key に複数 release がある場合は semver 最大の admitted release を通常 lookup で返す。旧 release は削除せず、監査と既存 match の artifact digest 再現のため保持する。
