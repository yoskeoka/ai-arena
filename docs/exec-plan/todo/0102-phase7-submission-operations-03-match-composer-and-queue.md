# phase7-submission-operations-03-match-composer-and-queue
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

operator UI で competition scope を選ぶと、ready/active bot 群から game manifest の
`player_count` 件を重複なしにランダム選択して表示し、`Shuffle` で再選択でき、確定した composition を
submit すると existing single logical FIFO queue へ initial run が積まれる flow を成立させる。

完了境界は、UI の convenience validation に依存せず server が scope、bot state、participant count、
uniqueness を再検証し、accepted match が exact game release/digest と exact bot submission revisions を
snapshot することとする。

## Black-box Contract

- UI は game scope を select し、scope-filtered eligible bots を取得する。
- initial selection と Shuffle は unbiased Fisher-Yates 等で active/ready bot から exact player count を選び、
  seat order も composition の一部として表示する。
- eligible bot が不足すると submit を disable し、理由を表示する。
- online form から `output_dir`、`player_id`、artifact/submission ID の自由入力を除く。
- final request は scope ID と ordered bot IDs を送る。server が `p1..pN` 等の match-local player IDs と
  output artifact prefix を生成する。
- server は count、distinct bot、same scope、active/ready、current revision existence を transaction 境界で再確認する。
- accept 時点で active game release/digest と各 bot の active submission revision/artifact digest を固定する。
- bot revision 更新後も accepted run、retry、rerun は original pinned revisions を使う。
- preset/manual source は同じ queue authority/FIFO policy に正規化し、new matchmaking daemon は作らない。

## Existing Implementation References

- `docs/specs/platform-service-match-request-scheduling.md`
  - current request/scheduling/validation/preset normalization, lines 37-151
- `internal/platform/service/request.go`
  - participant entity and current minimum validation, lines 1-22, 223-258
- `internal/platform/service/types.go`
  - queue submission currently stores only player ID + artifact ref, lines 15-31
- `internal/platform/service/store_postgres.go`
  - durable queue write, lines 70-83
- `operator-ui/src/routes/operator/RequestsPage.tsx`
  - current ID/output-dir/participant manual form, lines 66-112
- `operator-ui/src/lib/operatorApiClient.ts`
  - generated-client adapter boundary, lines 41-67

## Code Change Map

- `docs/specs/platform-service-match-request-scheduling.md` (MODIFY)
  - scope composition, exact player count, pinning, server-generated player/output behavior
- `docs/specs/platform-service-persistence.md` (MODIFY)
  - queue snapshot includes game release, bot, submission revision, artifact digest
- `typespec/namespaces/shared.tsp` (MODIFY)
- `typespec/namespaces/operator/api.tsp` (MODIFY)
- generated OpenAPI/operator client (MODIFY)
  - scope-filtered eligible bot list and simplified match request
- `internal/platform/service/request.go` (MODIFY)
  - bot resolution, eligibility/count validation, immutable snapshot construction
- `internal/platform/service/types.go` (MODIFY)
  - submitted player provenance beyond opaque artifact ref
- `internal/platform/service/postgres/schema/` and migration/query/sqlc (MODIFY)
  - durable request visibility and pinned run provenance
- `internal/platform/service/worker_local.go` (MODIFY)
  - materialize pinned artifact digests rather than resolve mutable local paths
- `operator-ui/src/routes/operator/RequestsPage.tsx` (MODIFY)
  - scope selector, selected bot cards, Shuffle, submit/insufficient state
- operator UI fixtures/Playwright tests (MODIFY)
  - deterministic injection for tests while production selection remains random

## Sub-tasks

- [ ] request/pinning/server-generated field contract を spec-first で固定する。
- [ ] eligible bot query と simplified match request を TypeSpec/handlers に追加する。
- [ ] durable request/run snapshot に bot/revision/game artifact provenance を残す。
- [ ] server-side exact count/distinct/scope/state validation を実装する。
- [ ] Requests page を selector + random composition + Shuffle へ置き換える。
- [ ] accepted run が FIFO queue へ入り、worker が pinned bytes を使う E2E を追加する。

## Dependencies and Parallelism

- depends on: `0101-phase7-submission-operations-02-registration-bot-ownership.md`
- [parallel] TypeSpec/backend request work と frontend composer は wire contract 固定後に並行できる。
- blocks: `0103-phase7-submission-operations-04-ranking-rerun-cycle.md`

## Verification

- player count -1/+1、duplicate、wrong scope、retired/not-ready bot rejection
- active revision race at acceptance does not create mixed/mutable provenance
- UI initial selection and Shuffle produce exact distinct count; insufficient state is explicit
- accepted match/request/run survives service restart and enters existing FIFO queue
- bot revision update after acceptance does not change retry/rerun artifact digests
- local/Postgres/S3-compatible browser lane, Go tests, TypeSpec/client generation, UI build, workflow lint

## Risks and Mitigations

- client-only validation で forged request が不正 composition を積む
  - mitigation: server owns final eligibility/count/pinning validation
- bot update と request acceptance が競合し revision が混ざる
  - mitigation: active revision read と request snapshot を同じ transaction/consistent read にする
- random selection を backend matchmaking に拡大して scope が膨らむ
  - mitigation: Phase 7 は operator composer の random suggestion + immediate FIFO enqueue に限定する

## Design Decisions

- random selection/Shuffle は operator composition UX であり、pending matchmaking service は導入しない。
- match-local player ID と output locator は server が生成する。
- accepted match/run は mutable bot/game state ではなく exact revision/digest を保持する。

