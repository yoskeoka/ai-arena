# ranking-completed-match-discovery
<!-- Internal plan prose is Japanese by repository policy; code identifiers and required literals remain unchanged. -->
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## 目的と完了境界

operator が ranking scope を確認するとき、その集計へ寄与する completed official logical match を全て `match_id` から辿り、対応する run detail を開けるようにする。完了後に Requests view から消える request の発見性を補う。

対象は認証済み operator Rankings surface のみとする。既存 completed-match read model を再利用し、public API、ranking aggregation、spectator の run selector、public selected-run semantics は変更しない。

## References and current behavior

- `typespec/namespaces/operator/api.tsp:99-101` and `operator-ui/src/lib/operatorApiClient.ts:230-233` already provide authenticated `GET /api/v1/matches/completed`.
- `typespec/namespaces/shared.tsp:263-301` preserves both match/run provenance in ranking facts and snapshots.
- `operator-ui/src/routes/operator/RankingsPage.tsx:11-48` already loads completed items to seed scope shortcuts but does not render them as match discovery records.
- `operator-ui/src/routes/operator/RequestsPage.tsx:104-130` only lists accepted requests, so it is not historical discovery after completion.
- `docs/specs/platform-public-spectator.md:24-31` fixes public selection to the `match_id` resource key and platform-selected run; it must not be weakened.

## Change map

- (MODIFY) `docs/specs/platform-service-operator-ui.md`: ranking-scope view が aggregate に寄与する `lifecycleState === "completed" && official` logical match を match ID、selected/official run ID、lifecycle、run-detail navigation とともに表示する behavior を定義する。
- (MODIFY) `operator-ui/src/routes/operator/RankingsPage.tsx`: 読み込み済み records を selected game ID/version/ruleset と `completed && official` で絞り込み、completed-match list と既存 run detail route への link を表示する。
- (MODIFY) `operator-ui/src/routes/operator/*.test.tsx` and browser coverage: assert that a completed match remains discoverable from Rankings after it is absent from Requests, including its match ID and run-detail link.
- (NO API CHANGE) `typespec/namespaces/operator/api.tsp`, generated clients, service, and persistence: the existing authenticated completed-match endpoint has the required provenance and remains the source.

## Work

1. Update the operator behavioral spec before UI code. Define scope matching and the distinction between aggregate ranking values and historical completed-match records.
2. Rankings では `completedItems` から stable な scope-filtered completed official list を導出し、独立した ranking snapshot の empty/loading/error state を壊さない。
3. Render match ID and the applicable run ID as operator-only history, with an accessible link to `/operator/runs/{run_id}`. Keep the ranking snapshot compact and do not infer match IDs from per-bot `last_run_id` fields.
4. Add component and Playwright coverage for two scopes and more than one completed match, proving that only the selected scope appears and run detail navigation retains the displayed IDs.
5. Run TypeSpec generation only if the implementation proves the existing response lacks an exposed field; otherwise leave generated artifacts unchanged.

## Dependencies and parallelism

This plan is independent of Reversi visualizer controls. It is a prerequisite only for an operator-friendly historical handoff, not for the public viewer: the viewer consumes public match discovery directly. The companion Reversi plan uses the public `match_id` resource and never this authenticated surface.

## Verification

- Targeted operator UI unit tests for scope filtering, ID display, empty state, and run-detail href.
- Existing browser operator workflow, extended to create/complete matches, navigate to Rankings, and open a displayed completed run.
- Applicable non-AI quality gates and workflow lint.
- Manual acceptance: after a request is no longer in Accepted Requests, selecting its ranking scope shows its `match_id` and opens its run detail; no public endpoint or ranking aggregate behavior changes.

## Non-goals

- Public/private artifact disclosure, public run-ID choice, changes to official-run promotion, ranking recomputation, pagination/filter API design, and visualizer hosting.
