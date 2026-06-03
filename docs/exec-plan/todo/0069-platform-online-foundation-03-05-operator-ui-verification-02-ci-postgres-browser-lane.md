# platform-online-foundation-03-05-operator-ui-verification-02-ci-postgres-browser-lane
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0068` の local browser verification foundation を再利用し、
Postgres-backed backend と組み合わせた `operator-ui` の minimal browser verification を
GitHub Actions で継続実行できる lane として整える。
最初のゴールは、polling / preset queue / completed detail / artifact access entry の
実画面回帰を durable backend shape で捕捉しつつ、default gate を不必要に重くしないことに置く。

## Context

- `docs/issues/0025-operator-ui-local-and-ci-verification.md` の後半 concern は、
  CI で operator UI の integration/e2e regression を捕まえられないことにある
- `docs/specs/platform-service-skeleton.md` は、default lane と durable Postgres lane を分離して扱う方針をすでに持っている
- operator UI regression は API contract だけでなく polling と画面 state composition の崩れで起きるため、
  browser lane を持たないと検知が遅れる
- 一方で default CI gate に重い browser lane を常設すると Phase 6/7 の速度が落ちるので、dedicated lane に閉じる必要がある

## Scope

- Postgres-backed backend + operator UI build + browser automation を組み合わせた CI lane を定義する
- `0068` の Playwright harness / selector strategy / scenario を再利用する
- schema apply、fixture seed、service/frontend 起動、browser verification、artifact upload を GitHub Actions に載せる
- failure 時の trace / screenshot / backend log を reviewer が読める形で残す
- OpenAI `playwright-interactive` skill や `js_repl` 前提は CI lane に持ち込まない

この plan では以下を扱わない。

- operator UI の broader feature expansion
- ranking / tournament / general submission flow の browser coverage
- heavy exploratory / visual diff testing

## Spec Changes

### `docs/specs/platform-service-skeleton.md`

- durable browser verification lane を dedicated CI lane として扱う契約を追記する
- schema bootstrap、backend/frontend 起動順、verification 実行主体の責務境界を明記する

### `docs/specs/platform-service-operator-ui.md`

- CI で常設する minimal browser coverage 範囲を追記する
- local lane と CI lane が共有する acceptance scenario を明文化する

## Development / CI Doc Changes

- `docs/development/go-quality-gates.md` に operator UI browser lane の位置づけを追記する
- 必要なら `docs/development/platform-service-online-deploy.md` に
  Pages build contract と CI browser verification の関係を補足する

## Expected Code Changes

- `0068` と共有する Playwright tests / fixtures / helper の整理
- Postgres schema apply と seed data を伴う backend bootstrap helper
- CI 専用の start/wait/log collection helper
- GitHub Actions workflow または job の追加
- failure artifact upload と triage 導線追加

## Sub-tasks

- [ ] CI lane の責務を default `go-ci` から切り離し、dedicated lane 名を固定する
- [ ] Postgres-backed backend を deterministic に起動する bootstrap 手順を整える
- [ ] `0068` の browser scenario を CI で再利用できるよう fixture 境界を整理する
- [ ] GitHub Actions で service/frontend/browser verification を実行する
- [ ] screenshot / trace / backend log を failure artifact として回収する
- [ ] repo-local docs に CI lane の起動条件と保守方針を追記する

## Parallelism

- [parallel] CI job 設計と failure artifact 方針整理は並行できる
- [parallel] Postgres bootstrap helper と frontend/browser start helper の整備は並行できる
- browser scenario の CI 組み込みは `0068` の foundation と CI bootstrap helper に depends on する

## Dependencies

- depends on: `0068-platform-online-foundation-03-05-operator-ui-verification-01-local-agent-browser-loop.md`
- depends on: `0065-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- depends on: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`
- depends on: `0046-platform-online-foundation-02-persistence-and-read-model.md`

## Risks and Mitigations

- browser + Postgres + backend + frontend を毎回起動すると CI が重くなり、flake source も増える
  - mitigation: first lane は narrow scenario に絞り、dedicated workflow/job として隔離する
- local lane と CI lane で fixture / selector / startup path が分岐すると二重保守になる
  - mitigation: Playwright scenario と selector strategy は `0068` を正本にして共有し、CI は bootstrap だけ追加する
- failed run の情報が薄いと CI red でも修正コストが高い
  - mitigation: trace / screenshot / backend log / frontend console を artifact として必ず残す

## Design Decisions

- operator UI browser verification は default gate へ直結せず、dedicated durable lane として扱う
- CI lane は local lane を置き換えるのではなく、`0068` の foundation を production-shape backend に延長する
- browser automation stack は local lane と同一の Playwright foundation を再利用する
- CI lane は ordinary Playwright CLI / test runner だけで成立させ、agent-specific skill 依存を持ち込まない
