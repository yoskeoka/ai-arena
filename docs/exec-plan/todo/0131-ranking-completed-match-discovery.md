# ranking-completed-match-discovery
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective and completion boundary

Allow an operator inspecting a ranking scope to find every completed logical match that contributes to that scope and open its run detail. This closes the discovery gap after an accepted request leaves the Requests view: a completed match remains findable by `match_id` beside the run selected for its ranking result.

Completion is limited to the authenticated operator Rankings surface. It reuses the existing completed-match read model; it does not add a public API, alter ranking aggregation, expose a run selector to spectators, or change public selected-run semantics.

## References and current behavior

- `typespec/namespaces/operator/api.tsp:99-101` and `operator-ui/src/lib/operatorApiClient.ts:230-233` already provide authenticated `GET /api/v1/matches/completed`.
- `typespec/namespaces/shared.tsp:263-301` preserves both match/run provenance in ranking facts and snapshots.
- `operator-ui/src/routes/operator/RankingsPage.tsx:11-48` already loads completed items to seed scope shortcuts but does not render them as match discovery records.
- `operator-ui/src/routes/operator/RequestsPage.tsx:104-130` only lists accepted requests, so it is not historical discovery after completion.
- `docs/specs/platform-public-spectator.md:24-31` fixes public selection to the `match_id` resource key and platform-selected run; it must not be weakened.

## Change map

- (MODIFY) `docs/specs/platform-service-operator-ui.md`: specify that a ranking-scope view exposes its completed logical matches with match ID, selected/official run ID, lifecycle, and run-detail navigation.
- (MODIFY) `operator-ui/src/routes/operator/RankingsPage.tsx`: filter the already-loaded completed records to the selected game ID, version, and ruleset; render a readable completed-match list and link each item to the existing run detail route.
- (MODIFY) `operator-ui/src/routes/operator/*.test.tsx` and browser coverage: assert that a completed match remains discoverable from Rankings after it is absent from Requests, including its match ID and run-detail link.
- (NO API CHANGE) `typespec/namespaces/operator/api.tsp`, generated clients, service, and persistence: the existing authenticated completed-match endpoint has the required provenance and remains the source.

## Work

1. Update the operator behavioral spec before UI code. Define scope matching and the distinction between aggregate ranking values and historical completed-match records.
2. In Rankings, derive a stable, scope-filtered list from `completedItems`; show empty/loading/error states without hiding the independently loaded ranking snapshot.
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
