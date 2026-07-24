# phase7-submission-operations-04-ranking-rerun-cycle
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

named bot と pinned revisions を ranking / retry / rerun / promote lifecycle へ通し、artifact 更新後も
同じ bot の順位が分裂せず、historical rerun が original bytes を使い、official run の切替が
ranking snapshot へ正しく反映される最小運営 cycle を完成させる。

既存 lifecycle mechanics を作り直さず、competitor identity と実 Reversi flow の不足を閉じる。

## Black-box Contract

- ranking competitor key は stable `bot_id` とし、entry は bot name と owner-visible label を表示する。
- queue/run は `bot_id`、exact submission revision、artifact digest を保持する。
- new match は current active revision を使うが、retry/rerun は parent match が pin した revision/digest を使う。
- bot revision 更新や retire は historical official result/ranking を書き換えない。
- failed run の retry、completed run の rerun、non-official completed run の promote、queued cancel は
  existing distinct lifecycle semantics を維持する。
- rerun completion は自動 official/ranked にせず、promote 後に同じ match の旧 official input を置換する。
- ranking recompute/verify は bot identity と official run set で deterministic に一致する。

## Existing Implementation References

- `docs/specs/platform-service-ranking-lifecycle.md`
  - official-run aggregation and current artifact-ref competitor key, lines 68-127
- `internal/platform/service/ranking.go`
  - ranking update/apply path, lines 370-454
- `internal/platform/service/run_commands.go`
  - current retry/rerun/promote orchestration, lines 1-19 and command methods
- `internal/platform/service/http.go`
  - current follow-up routes and operator protection, lines 147-159
- `operator-ui/src/routes/operator/RankingsPage.tsx`
  - current competitor-ref presentation, lines 150-162
- `operator-ui/src/routes/operator/RunDetailPage.tsx`
  - current action execution/availability, lines 48-74, 104-120

## Code Change Map

- `docs/specs/platform-service-ranking-lifecycle.md` (MODIFY)
  - bot identity, revision-spanning aggregate, retire/update behavior
- `docs/specs/platform-service-match-request-scheduling.md` (MODIFY)
  - retry/rerun pinned provenance cross-reference
- TypeSpec shared/operator ranking/run models and generated clients (MODIFY)
- `internal/platform/service/types.go` (MODIFY)
- `internal/platform/service/ranking.go` (MODIFY)
- `internal/platform/service/run_commands.go` (MODIFY)
- Postgres queue/request schema/query and ranking snapshot codec (MODIFY)
  - preserve bot/revision provenance across all attempts
- `operator-ui/src/routes/operator/RankingsPage.tsx` (MODIFY)
- `operator-ui/src/routes/operator/RunDetailPage.tsx` (MODIFY)
- local/CI/auth-enabled Playwright scenarios (MODIFY)
  - named bot ranking, revision update, rerun candidate, promote/correction

## Sub-tasks

- [ ] competitor identity migration と old snapshot compatibility/migration rule を spec-first で固定する。
- [ ] run/ranking update に bot/revision provenance を通す。
- [ ] retry/rerun が parent の pinned revisions を保持することを保証する。
- [ ] promote が official input と ranking snapshot を deterministic に置換する。
- [ ] Rankings/Run Detail pages を named bot/revision semantics に更新する。
- [ ] Reversi 2-bot completed -> ranking -> revision update -> rerun -> promote scenario を追加する。

## Dependencies and Parallelism

- depends on: `0102-phase7-submission-operations-03-match-composer-and-queue.md`
- [parallel] ranking codec/migration と frontend presentation は identity contract 固定後に並行できる。
- blocks: `0104-phase7-submission-operations-05-deploy-topology-and-reversi-staging.md`

## Verification

- same bot with multiple revisions remains one ranking entry
- retry/rerun retains original game/AI digests after active revision changes
- rerun is non-official until promote; promote swaps ranking input exactly once
- recompute/verify equals stored snapshot after correction
- retired bot history/ranking remains visible but cannot enter a new match
- Go/Postgres tests, actual browser flow, TypeSpec/client generation, UI build, workflow lint

## Risks and Mitigations

- competitor key migration が existing ranking snapshot を silent に混在させる
  - mitigation: schema/version marker と explicit rebuild/migration gate を設ける
- rerun が current revision を拾い historical reproduction を壊す
  - mitigation: follow-up run constructor は parent snapshot だけを source にする
- correction と ranking update の不整合窓が広がる
  - mitigation: current single-worker boundaryを維持し、recompute/verify を acceptance に含める

## Design Decisions

- bot identity は ranking の継続単位、submission revision は実行再現単位とする。
- retry/rerun は historical pinned revisions を使い、new match だけが current active revision を採用する。
- ranking/rerun は identity と official adoption が密結合するため同じ child plan で閉じる。

