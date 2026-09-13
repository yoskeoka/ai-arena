# Go 1.27 runtime and lint compatibility
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

`go1.27.1` でビルドした Go-WASM AI と game master を、既存の WASI runtime が
filesystem / S3-compatible artifact store のどちらでも完全な match として実行できるようにする。
あわせて、Go 1.27 code を解析できず panic する pinned staticcheck を、Go 1.27 support を
明示する upstream の署名済み stable tag `v0.8.1` へ上げ、`make lint` を再び品質判定として
機能させる。

この plan は `0130-go-1-27-toolchain` の `go` / `toolchain` directive を変更しない。0131 は
`GOTOOLCHAIN=go1.27.1` を明示した compatibility lane で runtime と lint tool を先に適合させ、
0131 の merge 後に保留中の 0130 を再開して module manifest を更新する。

完了時には、Go 1.27.1 で生成した WASI module の init / turn / game-over stream が既存の
JSON-RPC、deny-by-default sandbox、memory limit、stderr capture の契約を保ったまま完走する。
WASI module の early exit は、失われた incoming channel close ではなく、利用者が原因を
識別できる runtime failure evidence として残る。`make test`、`make test-postgres`、
`make test-wasm-go`、`make test-wasm-rust`、`make lint` は Go 1.27.1 selected toolchain で
成功し、tool / ordinary dependency metadata の差分はこの compatibility fix に必要なものだけとなる。
この全証跡から作る `.claude/skills/go-version-upgrade/SKILL.md` は、directive 更新前の
target-version compatibility 調査、block 条件、全 gate の toolchain inheritance を必須手順として
再利用可能に記録する。未検証の skill を先行して追加しない。

次は範囲外とする。

- Go 1.27 language feature を使う product code の導入
- `0130` が所有する `go.mod` の `go 1.27` / `toolchain go1.27.1` directive 更新
- WASI sandbox capability、artifact-store topology、CI workflow trigger の変更
- staticcheck の新規 check による既存 source finding を無関係に一括修正すること
- staticcheck upstream master / pseudo-version / pre-release を pin すること

## 背景と既存根拠

- `docs/exec-plan/todo/0130-go-1-27-toolchain.md:10-18,103-116` は Go 1.27.1 を preferred
  toolchain とし、Go-WASM と lint を同 toolchain で quality-gate とする。しかし実行時には
  Go-WASM admission と dedicated Go-WASM lane が `runtime-stopped` / failed となり、
  `make lint` は `honnef.co/go/tools@v0.7.0` の `unexpected expr: *ast.KeyValueExpr` panic
  で停止した。
- `internal/platform/runtime/wasm_wasi.go:23-98` は module exit 後に stdout reader を drain
  して incoming を close するが、normalization 済み module exit error は `done` にだけ送る。
  `internal/platform/session/session.go:112-151` は closed incoming を原因なしの
  `runtime-stopped` に変換し、`internal/platform/match/match.go:204-225` は init failure の
  stage だけを event に残すため、worker/persisted artifact まで exit evidence が届かない。
- `docs/specs/ai-runtime.md:100-170` は runtime kind を問わない stdio JSON-RPC、stderr capture、
  deny-by-default capability、`runtime-stopped` の分類を所有する。この fix はこの black-box
  runtime behavior を明確化してから実装する。
- `e2e/arena_runner_wasme2e_test.go:16-47` と
  `internal/platform/service/artifact_submission_e2e_test.go:19-45` は Go-WASM fixture を
  mixed runtime path および filesystem/S3-compatible admission path で検証する既存入口である。
- `go.mod:10,23,106` は staticcheck command と `honnef.co/go/tools v0.7.0`、WASI runtime
  `github.com/tetratelabs/wazero v1.11.0` を pin している。staticcheck upstream の stable
  2026.1 release は Go 1.26 までを対象にするが、2026.2.1 (`v0.8.1`) は Go 1.27 support を
  含む 2026.2 series の upstream signed stable tag である。master / pseudo-version / pre-release は
  pin しない。

## 仕様・契約変更

### Runtime failure observation contract

`docs/specs/ai-runtime.md` を次のように更新する。

- WASI runtime が request response を返す前に module exit / instantiate failure を起こしたとき、
  platform は `runtime-stopped` に分類しつつ、captured stderr と normalized exit cause を
  run/worker failure evidence に含める。exit code 0 でも response 前なら
  `module exited before response (exit code 0)` のような synthesized cause を残し、channel close
  だけで診断可能な exit cause を失ってはならない。
- 完全な response を stdout から decode 済みなら、その後の clean exit は response delivery を
  上書きしない。response 後の abnormal exit は既に返した response を failure に置換しない一方、
  normalized exit cause を structured audit event に残す。次の request 前の exit は上記の
  `runtime-stopped` contract に従う。
- 既存の `runtime_exited` event payload は replay が `game.ActionStatus` として decode するため、その
  shape を変更しない。exit cause / stderr diagnostic は別 event として append し、旧 record replay を
  読めるままにする。
- この観測性追加は stderr の公開範囲、host capability、JSON-RPC wire format を変えない。

### Development quality-gate contract

`docs/development/go-quality-gates.md` を次のように更新する。

- Go version upgrade の preflight は、target Go toolchain で Go-WASM and Rust-WASM lanes と
  lint suite を実行し、all-green evidence がなければ `go` / `toolchain` directive を上げない。
- staticcheck が target Go version を未 support の場合、aggregate lint を成功として扱わない。
  Go 1.27 では upstream signed stable tag `v0.8.1` を pin して full lint を実行する。master /
  pseudo-version / pre-release / bypass は採らない。

## 変更マップ

- `(MODIFY) docs/specs/ai-runtime.md:100-170`:
  WASI response-before-exit と pre-response exit の観測可能な failure evidence を固定する。
- `(MODIFY) internal/platform/runtime/wasm_wasi.go:23-174`:
  exit cause と captured stderr を失わず、decoded response を優先して delivery する WASI adapter
  lifecycle にする。確定した upstream stable wazero release がこの lifecycle を必要とする場合だけ、
  compatibility fix として pin を更新する。
- `(MODIFY) internal/platform/runtime/runtime.go:20-71` and
  `(MODIFY) internal/platform/runtime/runtime_test.go:120-340`:
  runtime adapter の observable terminal-failure surface と Go 1.27 compiled WASI fixture の
  response/exit ordering regression を固定する。
- `(MODIFY) internal/platform/session/session.go:20-151`,
  `(MODIFY) internal/platform/match/match.go:20-405`, and
  `(MODIFY) internal/platform/service/worker_local.go:220-258` with their focused tests:
  adapter の normalized/synthesized exit cause を `runtime-stopped` 分類を保ったまま session result、
  structured match event、persisted local terminal evidence まで伝播し、player stderr snapshot と同じ
  artifact ownership境界で診断可能にする。response 後の abnormal exit は response delivery を置換せず
  audit event に記録する。
- `(MODIFY) internal/platform/gamemaster/gamemaster.go:40-75,280-320` and
  `(NEW) internal/platform/gamemaster/gamemaster_test.go`:
  WASI game-master session も closed incoming を generic `runtime stopped` に落とさず、adapter の
  normalized/synthesized exit cause と stderr evidence を match failure/event へ渡す。metadata /
  initialize-match response 前の clean/non-zero exit と response 後 abnormal exit の regression を
  固定する。
- `(MODIFY) internal/platform/replay/session_history.go:84-116`,
  `(MODIFY) internal/games/janken/janken.go:192-210`, and
  `(MODIFY) internal/games/echo/echo.go:204-218` with replay regressions:
  `runtime_exited` の既存 `game.ActionStatus` decode shape を維持し、別 diagnostic event が
  history/replay input を壊さず audit evidence として残ることを固定する。
- `(MODIFY) e2e/arena_runner_wasme2e_test.go:16-47` and
  `(MODIFY) internal/platform/service/artifact_submission_e2e_test.go:19-180`:
  Go 1.27.1 compiler を明示する fixture build で mixed runtime と両 artifact store の complete
  match を assertion し、failure 時に stderr / normalized exit cause を reviewable に出す。
- `(MODIFY) go.mod:7-106` and `(MODIFY, only if required) go.sum`:
  Go 1.27 support を含む upstream signed stable tag `honnef.co/go/tools v0.8.1`、および必要な
  stable wazero compatibility release だけに pin を更新する。その他の direct / indirect dependency
  update は revert して別 plan とする。
- `(MODIFY) docs/development/go-quality-gates.md:1-104`:
  Go version upgrade preflight と unsupported staticcheck の block policy を追加する。
- `(NEW) .claude/skills/go-version-upgrade/SKILL.md`:
  全 Go 1.27 compatibility gate が成功した後にだけ、今回実証した toolchain upgrade の
  preflight を再利用可能な手順として記録する。official target release、Go-WASM AI / game-master、
  pinned Go tools、WASI runtime の target-version support を directive 更新前に調査し、未対応なら
  bypass せず compatibility plan を先行させること、`GOTOOLCHAIN` を child fixture build を含む全 gate に
  継承させること、green evidence 後に skill を作ることを固定する。
- `(NO CHANGE, verify) .github/workflows/go-ci.yml:31-123` and
  `(NO CHANGE, verify) .github/workflows/wasm-verification.yml:34-80`:
  `go-version-file: go.mod` と既存 job topology は維持し、0130 resume 後の current-head CI で
  compatibility lane が成功することを確認する。

## 実施手順

1. Go 1.27.1 selected toolchain で focused Go-WASM E2E と artifact submission E2E を実行し、
   module exit error、stderr snapshot、decoded response の有無を test failure に出す最小 regression を
   先に追加する。Go 1.26 toolchain と比較して原因を確定し、fixture / runtime code のどちらが
   early exit を引き起こすかを切り分ける。
2. `docs/specs/ai-runtime.md` を先に更新し、response delivery を保持する境界と pre-response
   WASI exit の evidence contract を固定する。
3. Step 1 で確定した lifecycle defect を `wasmWASIAdapter`、player session、WASI game-master
   session、match/service propagation に最小実装する。正常 response を EOF/normal exit より先に届け、
   response 前の clean exit には synthesized cause を付け、異常 exit は `runtime-stopped` の分類を
   保ちながら run result / persisted evidence / test diagnostics で原因を追えるようにする。response 後の
   abnormal exit は delivered response を置換せず audit event に残す。filesystem と S3-compatible store
   のいずれにも固有の fallback を加えない。
4. module generated by Go 1.27.1 が pinned wazero の supported behavior 外であることを確認した
   場合だけ、upstream release notes と stable tag を根拠に wazero を最小 stable release へ更新する。
   stable release がない場合は upstream issue URL と focused reproduction を記録し、この plan を
   blocked とする。pre-release / master pin、interpreter fallback、Go compiler downgrade は採らない。
5. staticcheck upstream signed stable tag `v0.8.1` が Go 1.27 support を含む 2026.2 series であることを
   確認してから、`honnef.co/go/tools` tool pin をその tag に更新し `go mod tidy` を実行する。新しい lint
   finding は本 plan の tool update が直接露出したものだけ修正し、既存 source cleanup は separate issue
   にする。
6. `docs/development/go-quality-gates.md` に preflight / block policy を記録し、各 command とその
   fixture の child `go build` が同じ compiler を継承するよう
   `GOTOOLCHAIN=go1.27.1 CACHE_ROOT=/tmp/ai-arena-go-quality-gates make test`、
   `GOTOOLCHAIN=go1.27.1 CACHE_ROOT=/tmp/ai-arena-go-quality-gates make test-postgres`、
   `GOTOOLCHAIN=go1.27.1 CACHE_ROOT=/tmp/ai-arena-go-quality-gates make lint`、
   `GOTOOLCHAIN=go1.27.1 CACHE_ROOT=/tmp/ai-arena-go-quality-gates make test-wasm-go`、
   `GOTOOLCHAIN=go1.27.1 CACHE_ROOT=/tmp/ai-arena-go-quality-gates make test-wasm-rust` を実行する。
   Postgres lane は `make postgres-up` と既存 DSN contract を使う。restricted sandbox では writable
   `/tmp` cache root を使う。
7. Step 1-6 の full green evidence が揃った後にのみ `.claude/skills/go-version-upgrade/SKILL.md` を
   作る。skill は、(a) Go official release と `go.mod` の pinned Go tools / WASI runtime の
   target-version support を directive 更新前に確認する、(b) Go-WASM AI と game master を含む
   filesystem / S3-compatible regression と lint suite を `GOTOOLCHAIN=go<target>` 継承下で実行する、
   (c) tool/runtime incompatibility は bypass / old compiler fixture で隠さず separate compatibility plan
   を先行させる、(d) all-green local/CI evidence を得てから upgrade PR を作る、という今回実証済みの
   判断だけを記録する。未成功の仮説、特定 version 固有でない dependency update、CI topology 変更を
   一般手順として含めない。
8. 0131 を merge した後にのみ、0130 branch を current main へ rebase して Go directives を適用し、
   same quality gates と `go-ci` / `wasm-verification` latest-head evidence を取得する。0131 の PR は
   0130 directive を先取りして含めない。

## 依存関係と並行性

| 順序 | 作業 | 依存 | 並行可否 |
| --- | --- | --- | --- |
| 1 | Go 1.27 focused reproduction / exit evidence | なし | staticcheck release research と並行可 |
| 2 | black-box runtime contract | 1 | 不可 |
| 3 | WASI lifecycle fix / stable wazero decision | 2 | lint tool pin と並行可 |
| 4 | staticcheck v0.8.1 pin / findings | upstream signed stable tag | WASI fix と並行可 |
| 5 | complete local gates / documentation | 3, 4 | 不可 |
| 6 | evidence-based Go upgrade skill | 5 | 不可 |
| 7 | 0131 PR CI, then 0130 resume | 6, merge | 順次 |

## 検証

- Go 1.27.1 で生成した Go-WASM fixture が `TestArenaRunnerJankenGoWASMMixedRuntimePath` を
  completed status、expected winner、stderr capture、history artifact 付きで通過する。
- 同じ compiler output を使う
  `TestArtifactSubmissionUploadToWASIStartAcrossBundleStores` が filesystem と S3-compatible store
  の双方で completed record と digest identity を返す。
- response 前に WASI module を exit code 0 / non-zero で exit させる focused test は
  `runtime-stopped` と normalized/synthesized exit cause / stderr evidence を観測する。response 後の
  clean exit test は response delivery を失わず、response 後の abnormal exit test は delivered response
  を置換せず structured audit event に exit cause を残す。これらは player AI と WASI game master の
  両方の session path で確認する。
- existing record replay は `runtime_exited` payload を従来どおり `game.ActionStatus` として decode
  でき、new diagnostic event を含む record も同じ snapshot/history result を再構成できる。
- `go.mod` / `go.sum` は approved staticcheck v0.8.1 / 必要なら stable wazero pin だけを変更し、
  `go mod tidy` 後に余計な dependency update がない。
- `GOTOOLCHAIN=go1.27.1` を export した selected toolchain で `make test`、`make test-postgres`、
  `make lint`、`make test-wasm-go`、`make test-wasm-rust` が成功する。staticcheck panic を pass と扱う
  bypass は存在しない。
- `.claude/skills/go-version-upgrade/SKILL.md` は、official target release、pinned Go tools と WASI
  runtime の target-version support、Go-WASM AI / game-master の両 artifact-store lane、全 quality gate の
  `GOTOOLCHAIN` inheritance、unsupported dependency の block / compatibility-plan policy を含む。一方、
  all-green evidence 前に skill を作る手順、old compiler fixture、lint bypass、unrelated dependency update を
  推奨しない。
- 0130 resume PR では existing `go-ci` と `wasm-verification` の current-head checks が
  `go-version-file: go.mod` 経由で Go 1.27.x を選び成功する。operator browser workflow はその PR の
  path filter に従い、未起動を failure evidence にしない。

## 実行時の判断

- Go-WASM fixture を Go 1.26 で固定する案は採らない。0130 の single version-source contract と
  Go 1.27 support を偽装するためである。
- staticcheck master / pseudo-version / pre-release を pin する案は採らない。Go 1.27 support を含む
  upstream signed stable tag `v0.8.1` を採用する。
- `runtime-stopped` を timeout / malformed に再分類する案は採らない。既存 runtime spec の
  transport-failure distinction を壊し、operator が WASI process exit を診断できなくなるためである。
