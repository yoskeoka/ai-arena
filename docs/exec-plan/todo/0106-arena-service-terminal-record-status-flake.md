# arena-service-terminal-record-status-flake
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0038-arena-service-terminal-record-status-flake.md`

## 目的と完了境界

`arena-service run-once --match-timeout` の match context が期限切れになった実行は、player subprocess の終了や session transport の結果との競合があっても、terminal record を `canceled` として永続化・出力する。terminal artifact が得られた場合の service lifecycle は従来どおり `completed` とし、runner terminal status と混同しない。

完了は、`TestRunOnceFailureStillPrintsTerminalRecord` が要求する artifact/output を保ったまま、deadline expiry を反復しても `match_status=canceled` で安定することとする。match context が生きている状態での runtime stop、protocol failure、artifact persist failure は従来どおり `failed` であり、queue、TypeSpec、HTTP wire contract、WASM/WASI transport の意味は変更しない。

## 原因仮説と現行参照

- `docs/issues/0038-arena-service-terminal-record-status-flake.md` lines 1-34
  - Postgres CI で `TestRunOnceFailureStillPrintsTerminalRecord` が `failed` を出力した観測と、terminal record を維持した cancellation を期待する問題定義。
- `cmd/arena-service/main_test.go:283-349` `TestRunOnceFailureStillPrintsTerminalRecord`
  - 10ms の match timeout で real local players を実行し、`completed` lifecycle、terminal artifacts、`canceled` match status を assertion している。
- `internal/platform/service/worker.go:39-121` `(*Worker).ProcessNext`
  - terminal record が得られれば persister 実行後に lifecycle を `completed` とし、runner record の status を terminal artifact summary へそのまま写す。
- `internal/platform/service/worker_local.go:45-94` `(*LocalRunnerInvoker).Run`
  - `--match-timeout` から child/session/game-master に共有する run context を作り、runner record と error を service worker へ返す。
- `internal/platform/match/match.go:164-186` `(*Runner).Run`、`190-225` `initializeSessions`、`242-288` `runDecisionLoop`、`461-487` `executeTurn`
  - runner は returned error が context cancellation/deadline なら `canceled`、それ以外なら `failed` とする。現在は session result が timeout である場合にのみ `ctx.Err()` を優先するため、deadline と `runtime-stopped` 等の観測順が競合すると non-context error が `failed` になる余地がある。
- `internal/platform/match/match_test.go:151-214` cancellation/failure regressions
  - context cancellation と ordinary init runtime-stop の区別を既に unit level で確認している。
- `docs/specs/platform.md:534-549` match lifecycle phase
  - `completed` / `failed` / `canceled` は terminal phase で record status と整合しなければならない。
- `docs/specs/platform-service-read-model.md:58-72` lifecycle と terminal status の分離
  - terminal artifact persist 成功を表す service lifecycle `completed` は、runner terminal status が `canceled` でも成立する契約。

## Black-box Specification Changes

`docs/specs/platform.md` の match lifecycle contract に、match を統括する context の cancellation/deadline を runner が観測した時点で terminal status を `canceled` とする優先規則を追加する。

- session/process/game-master が同時に返した runtime-stop、timeout、shutdown error は、期限切れ match context に由来する実行の terminal classification を `failed` へ上書きしない。
- context が生きている実行の runtime-stop、protocol error、game-master error は `failed` のままとする。
- terminal record を保存できた service run は、runner status にかかわらず lifecycle `completed` とする既存契約を明示的に維持する。

TypeSpec field、artifact format、queue state enum、runner invocation API は変更しない。

## Code Change Map

- `docs/specs/platform.md` (MODIFY)
  - match-scoped cancellation/deadline と concurrent session/process failure の terminal-status precedence、および service lifecycle との分離を observable contract として追記する。
- `internal/platform/match/match.go` (MODIFY)
  - init/turn execution path で run context の cancellation/deadline を一貫して terminal cancellation として返す小さな classification seam を設ける。
  - context が active な ordinary session/runtime failure は既存の event recording と `failed` classification を保つ。
- `internal/platform/match/match_test.go` (MODIFY)
  - deadline 済み context と `runtime-stopped`/non-timeout session result が競合しても `canceled` record と `match_canceled` event になる deterministic regression を追加する。
  - active context の runtime-stop が `failed` に残る既存 regression を維持・必要なら明確化する。
- `cmd/arena-service/main_test.go` (MODIFY)
  - real local invocation の flaky regression を timing 偶然性に依存しない failure mode へ補強し、terminal artifact、lifecycle `completed`、terminal `canceled`、error return の組合せを反復で検証できるようにする。
- `docs/issues/0038-arena-service-terminal-record-status-flake.md` (DELETE)
  - implementation verification と PR preparation の完了後、resolved local issue として execution PR で削除する。plan PR では残す。
- `docs/exec-plan/todo/0106-arena-service-terminal-record-status-flake.md` (DELETE)
  - implementation verification と PR preparation の完了後、matching execution plan として execution PR で削除する。plan PR では残す。

## 実行サブタスク

- [ ] spec-first で、match context の deadline/cancel を観測した terminal classification の優先規則と service lifecycle との独立性を更新する。
- [ ] runner の init と turn path を調べ、match context が done のとき session result の観測順に依存せず同じ context error を上位へ返すようにする。active context の ordinary failure path は変更しない。
- [ ] fake session/master を用いた deterministic match regression を追加し、deadline + runtime-stop の競合が `canceled` となり `failed` event を出さないことを固定する。
- [ ] `arena-service` の real local regression を、terminal record と artifact persistence を end-to-end で確認する coverage として維持し、repetition 時の failure output に status/event context を残す。
- [ ] targeted repeat、Postgres-enabled service test、full quality gates を実行する。implementation PR では resolved issue と plan を削除し、原因・contract・repeat evidence を記録する。

## 依存関係と並行性

- depends on: none
- [sequential] terminal-status precedence を spec で固定してから runner classification を変更する。
- [parallel after contract] deterministic match regression と CLI/service end-to-end regression の fixture/assertion は並行できる。
- [sequential] targeted repeat と Postgres/full quality gate は implementation と両 test lane の完了後に実行する。

## Verification

- `GOPATH=/tmp/ai-arena-terminal-status-flake/go GOMODCACHE=/tmp/ai-arena-terminal-status-flake/go/pkg/mod GOCACHE=/tmp/ai-arena-terminal-status-flake/go-build go test ./internal/platform/match -run 'TestRunner.*(Cancellation|Failure|Deadline)' -count=100`
- `GOPATH=/tmp/ai-arena-terminal-status-flake/go GOMODCACHE=/tmp/ai-arena-terminal-status-flake/go/pkg/mod GOCACHE=/tmp/ai-arena-terminal-status-flake/go-build go test ./cmd/arena-service -run TestRunOnceFailureStillPrintsTerminalRecord -count=100`
- focused match and `cmd/arena-service` packages with `-race`; command timeout requires a reduced count の場合は実行 PR に actual count を記録する。
- `make test-postgres`
- `make test`
- `make lint`
- record/event/artifact output を確認し、deadline expiry は lifecycle `completed` + terminal `canceled`、active-context runtime failure は terminal `failed`、terminal artifact を生成できない failure は lifecycle `failed` であることを確認する。

## リスクと緩和

- context が deadline に達する直前の独立した runtime failure を誤って `canceled` と見なす可能性がある。
  - 緩和: classification の境界を runner が run context の done を観測した時点に限定し、active-context runtime-stop の `failed` regression を残す。
- CLI の短い wall-clock timeout だけに依存すると、CI の負荷差で再び flaky になる。
  - 緩和: primary regression は fake session による deterministic deadline/runtime-stop race とし、CLI test は artifact/persistence integration coverage に限定する。
- runner status の修正で service lifecycle の意味を変えてしまう可能性がある。
  - 緩和: `Worker.ProcessNext` の terminal-persist-success path を変更せず、read-model contract と end-to-end assertion で `completed` / `canceled` の二軸を固定する。
