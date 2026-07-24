# phase7-submission-operations-05-deploy-topology-and-reversi-staging
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

Phase 7 の service / worker / storage minimum topology を運用契約として固定し、
reversi-ai-arena の GitHub Release assets を staging operator flow から実際に upload/register/submit して、
Shuffle -> queue -> completed -> ranking -> rerun/promote まで通す。

完了境界は preset echo/janken ではなく Reversi bundle の exact release URL/digest を evidence として残し、
service restart 後も registration/bot/request/ranking が見え、in-flight run が永久に leased/running に残らず
recovery policy に従うこととする。この plan の完了時に依存 plan の acceptance を再確認し、
`docs/project-plan.md` の Phase 7 child items/status を実績に合わせて更新する。

## Topology Decision

Phase 7 minimum は current Render `arena-service serve` の HTTP service + colocated single worker を維持する。
shared Neon Postgres と R2 を使い、Pages には R2 credential を渡さない。distributed fairness や複数 worker は
後続とし、1 process/1 worker の operational guard、lease expiry/startup recovery、worker heartbeat/queue lag、
graceful shutdown を追加する。

service/worker 分離は将来 seam として command/config boundary を明示するが、この plan では deploy unit を
増やすことより、現 topology の recovery と observable operation を優先する。

## Existing Implementation References

- `docs/specs/platform-service-persistence.md`
  - Pages + Render + Neon + R2 split, metadata/artifact responsibilities, lines 74-111
- `docs/specs/platform-service-single-worker-assumptions.md`
  - current non-atomic single-worker boundaries
- `cmd/arena-service/main.go`
  - colocated HTTP server/worker loop and R2 wiring, lines 467-528, 639-690
- `internal/platform/service/worker_loop.go`
  - current poll loop, lines 1-38
- `internal/platform/service/store_postgres.go`
  - durable queue/lease backend
- `.github/workflows/online-release-staging.yml`
  - current staging release topology
- `.github/workflows/online-release-staging-verify.yml`
  - current echo preset remote verification, lines 20-24, 92-136
- `docs/development/platform-service-online-deploy.md`
  - provider bootstrap/deploy inventory

## Code Change Map

- `docs/specs/platform-service-single-worker-assumptions.md` (MODIFY)
  - minimum topology, lease expiry/recovery, heartbeat, shutdown and scaling guard
- `docs/specs/platform-service-persistence.md` (MODIFY)
  - submitted bundle storage/materialization and operational recovery
- `docs/development/platform-service-online-deploy.md` (MODIFY)
  - Pages/Render/Neon/R2 inventory, secret ownership, migration/recovery/runbook
- `docs/project-plan.md` (MODIFY)
  - update Phase 7 only after all child acceptance is evidenced
- `cmd/arena-service/main.go` (MODIFY)
  - topology modes/guard, graceful drain, recovery wiring
- `internal/platform/service/worker_loop.go` (MODIFY)
- `internal/platform/service/store_postgres.go` and schema/migration/query (MODIFY)
  - lease deadline, heartbeat/recovery and queue observability
- `.github/workflows/online-release-staging.yml` (MODIFY)
- `.github/workflows/online-release-staging-verify.yml` (MODIFY)
- `.github/workflows/online-release-production.yml` (MODIFY as required)
  - schema apply ordering, release deploy, Reversi asset inputs/digests, remote browser verification
- `operator-ui` remote Playwright scenario/helper (MODIFY)
  - actual release bundle upload through complete Phase 7 flow
- runbook/evidence output under workflow artifacts (NEW/MODIFY)
  - release asset URLs/digests, created scope/bot/match/run IDs, ranking verification result

## Sub-tasks

- [ ] minimum topology、single-worker guard、lease/recovery/heartbeat contract を spec-first で固定する。
- [ ] stale leased/running/persisting run の startup/expiry recovery と graceful shutdown を実装する。
- [ ] migration -> service deploy -> remote verify の ordering と secrets inventory を固める。
- [ ] Reversi release ZIPs を取得し、staging browser/API flow から upload/register/bot create する。
- [ ] Shuffle、queue、completion、ranking、rerun/promote/recompute verify を remote scenario で通す。
- [ ] preset echo を diagnostic fallback に下げ、Reversi evidence を release gate にする。
- [ ] all acceptance 後に project plan Phase 7 status を更新する。

## Dependencies and Parallelism

- depends on: `0103-phase7-submission-operations-04-ranking-rerun-cycle.md`
- depends on: `reversi-ai-arena/docs/exec-plan/todo/0007-ai-arena-release-artifacts.md`
- [parallel] recovery implementation と remote scenario scaffolding は artifact/API contract 固定後に並行できる。

## Verification

- stale lease/restart/recovery integration tests and single-worker guard failure test
- local deploy-shaped Postgres + S3-compatible full Reversi cycle
- staging exact GitHub Release assets/digests accepted; no repo-local path/preset dependency
- service restart retains scopes, bots, requests, runs and ranking
- remote browser evidence covers upload -> registration -> 3 named bots -> Shuffle -> match -> ranking -> rerun/promote
- ranking recompute/verify succeeds after promote
- staging/production workflow syntax/action pin checks, applicable Go/UI tests, workflow lint

## Risks and Mitigations

- remote E2E が cross-repo latest asset に依存し再現不能になる
  - mitigation: release tag、asset URL、SHA-256 を explicit workflow input/evidence に固定する
- colocated process crash が service と worker を同時停止する
  - mitigation: durable state、lease recovery、health/heartbeat、bounded single-worker model を先に閉じる
- Phase 7 status を部分実装だけで完了扱いする
  - mitigation: this plan が全 dependency と real Reversi evidence を確認して最後に status を更新する

## Design Decisions

- Phase 7 minimum deploy は colocated Render service + single worker、Neon、R2、Pages とする。
- preset は diagnostic fallback、Reversi release artifact flow を Phase 7 acceptance lane とする。
- multi-worker/distributed fairness は recovery と real operating cycle 完了後の follow-up とする。
