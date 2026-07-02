# typespec-api-contracts-02-operator-ui-client-adoption
**Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

`0098-typespec-api-contracts-01-foundation.md` で導入した TypeSpec-generated client を
`operator-ui` から実際に利用するよう refactor し、
current handwritten `operator-ui/src/api.ts` を
generated contract + thin browser adapter へ置き換える。

この plan のゴールは、
route page ごとに DTO や fetch path を手書きで複製する構成をやめ、
frontend が backend operator API のうち
frontend 向け group として emit された contract を直接使うことにある。

完了条件:

- `operator-ui` route code が handwritten `OperatorApiClient` に依存していない
- generated client/types を route page と shared hook が利用している
- browser-specific base URL、credentials、error normalization は thin local adapter に閉じ込められている
- current operator UI acceptance surface が既存 verification lane で維持されている

## Context

- current `operator-ui/src/api.ts` は DTO 定義、fetch path、error decode、URL assembly を 1 file に抱えている
- `operator-ui/src/App.tsx`、`LoginPage.tsx`、`useOperatorPageState.ts`、
  `GamesPage.tsx`、`InvitesPage.tsx`、`SubmissionsPage.tsx`、
  `RequestsPage.tsx`、`RankingsPage.tsx`、`RunDetailPage.tsx`
  がこの handwritten client に依存している
- `docs/specs/platform-service-operator-ui.md` は
  page-local fetch/polling と route-first module boundary を正本にしているため、
  generated client adoption 後も page-local state/polling 自体は壊してはならない
- user は frontend から backend API のうち frontend 向け contract を利用できるなら、
  client lib 生成などを使って frontend から利用するよう refactor したいと明示している

## Option Snapshot

### Option A: generated client を各 page から直接呼び出し、base URL/credentials/error 処理も page 側へ展開する

- 利点:
  wrapping layer が最小になる
- 欠点:
  browser-specific fetch policy が route file ごとに散りやすい

### Option B: generated client を thin local adapter で包み、
### base URL、credentials、error normalization、login URL helper だけを local ownership に残す

- 利点:
  TypeSpec-generated contract を使いながら browser 実装都合を 1 箇所へ閉じ込められる
- 欠点:
  adapter layer を 1 段維持する必要がある

## Recommendation

Option B を採る。

- TypeSpec-generated shapes と operation surface を route code に見せる
- ただし `credentials: "include"`、base URL normalize、
  login URL assembly、UI-facing error text 抽出は local adapter に残す
- `route-first` と `page-local fetch/polling` は維持し、
  global client store や query library 導入へは広げない

## Scope

- handwritten `operator-ui/src/api.ts` の責務を分解する
- generated client/types の import path を route code へ繋ぐ
- browser-specific adapter を整える
- route pages と shared hooks の imports / call sites / DTO references を置き換える
- `platform-service-operator-ui.md` の API call reference を generated-contract-aware な表現へ更新する

この plan では以下を扱わない。

- new router library の導入
- global cache/query library の導入
- operator UI screen/interaction scope の拡張
- public/external API 向け frontend consumer の追加

## Existing Implementation References

- `operator-ui/src/api.ts`
  - handwritten DTOs, `OperatorApiClient`, error decode, lines 1-456
- `operator-ui/src/App.tsx`
  - `ProtectedOperatorRoute`, `logoutAndReturnToLogin`, lines 17-174
- `operator-ui/src/routes/login/LoginPage.tsx`
  - session check and GitHub login URL helper usage, lines 10-136
- `operator-ui/src/routes/operator/useOperatorPageState.ts`
  - overview polling and preset enqueue flow, lines 10-171
- `operator-ui/src/routes/operator/GamesPage.tsx`
  - game list/create usage, lines 11-122
- `operator-ui/src/routes/operator/InvitesPage.tsx`
  - invite issue flow, lines 17-131
- `operator-ui/src/routes/operator/SubmissionsPage.tsx`
  - AI submission list/create usage, lines 11-156
- `operator-ui/src/routes/operator/RequestsPage.tsx`
  - request list/create usage, lines 14-169
- `operator-ui/src/routes/operator/RankingsPage.tsx`
  - completed list -> ranking scope -> snapshot usage, lines 11-222
- `operator-ui/src/routes/operator/RunDetailPage.tsx`
  - detail read and run follow-up actions, lines 12-153
- `docs/specs/platform-service-operator-ui.md`
  - route/page interaction and polling contract, lines 42-72, 180-297
- `docs/specs/platform-frontend-architecture.md`
  - page-local data boundary and route-first ownership, lines 47-131, 152-190
- `operator-ui/package.json`
  - existing verification scripts, lines 1-32

## Code Change Map

- `operator-ui/src/api.ts` (MODIFY or DELETE)
  - replace monolithic handwritten DTO/client file with generated-contract entrypoint or thin compatibility layer
- `operator-ui/src/generated/operator-api/*` (MODIFY)
  - consume the emitted client/types produced by the TypeSpec project
- `operator-ui/src/lib/operatorApiClient.ts` (NEW)
  - thin browser adapter for base URL normalization, credentials policy, login URL helper, and error normalization
- `operator-ui/src/App.tsx` (MODIFY)
  - swap `OperatorApiClient` construction and auth/session/logout wiring to the new adapter
- `operator-ui/src/routes/login/LoginPage.tsx` (MODIFY)
  - use generated-contract-backed session/login helper
- `operator-ui/src/routes/operator/useOperatorPageState.ts` (MODIFY)
  - use generated-contract-backed active/completed/detail/preset operations
- `operator-ui/src/routes/operator/GamesPage.tsx` (MODIFY)
  - use generated game registration request/response shapes
- `operator-ui/src/routes/operator/InvitesPage.tsx` (MODIFY)
  - use generated signup invite request/response shapes
- `operator-ui/src/routes/operator/SubmissionsPage.tsx` (MODIFY)
  - use generated AI submission request/response shapes
- `operator-ui/src/routes/operator/RequestsPage.tsx` (MODIFY)
  - use generated match request request/response shapes
- `operator-ui/src/routes/operator/RankingsPage.tsx` (MODIFY)
  - use generated ranking query/response shapes
- `operator-ui/src/routes/operator/RunDetailPage.tsx` (MODIFY)
  - use generated detail/action operations
- `docs/specs/platform-service-operator-ui.md` (MODIFY)
  - update API call ownership wording so route pages depend on TypeSpec-generated contract rather than handwritten DTO duplication

## Spec Changes

- `docs/specs/platform-service-operator-ui.md`
  - page-local fetch/polling contract は維持しつつ、
    frontend API call surface は TypeSpec-generated operator contract を使うこと、
    browser-specific adapter は local ownership であることを明記する

## Sub-tasks

- [ ] generated client の import surface を route code から扱いやすい形に整える
- [ ] base URL / credentials / login URL / error normalization を thin local adapter へ移す
- [ ] auth/session route 依存を `App.tsx` と `LoginPage.tsx` で置き換える
- [ ] overview polling / preset enqueue を generated-contract-backed flow へ置き換える
- [ ] games / invites / submissions / requests / rankings / run detail page を generated types/operations へ置き換える
- [ ] handwritten DTO duplication が不要になった部分を削除する
- [ ] verification lane で route/page acceptance surface が維持されることを確認する

## Parallelism

- [parallel] auth/session adapter 置換と
  CRUD page (`GamesPage`, `InvitesPage`, `SubmissionsPage`, `RequestsPage`) の置換は並行できる
- [parallel] `RankingsPage` と `RunDetailPage` の read/action 置換は並行できる
- `useOperatorPageState.ts` と `App.tsx` は adapter surface の形に depends on する

## Dependencies

- depends on: `0098-typespec-api-contracts-01-foundation.md`

## Verification

- `operator-ui` build が generated client import を含めて通る
- existing browser verification lanes が current acceptance surface を維持する
- handwritten `operator-ui/src/api.ts` が source-of-truth DTO file ではなくなっている
- route-local state/polling が `platform-frontend-architecture.md` の page-local boundary を逸脱していない

## Risks and Mitigations

- generated client を直接 page へ流し込みすぎると browser-specific fetch policy が散る
  - mitigation:
    thin local adapter を挟み、credentials/base URL/error handling を 1 箇所に残す
- generated code の命名や call shape が current UI ergonomics と噛み合わない
  - mitigation:
    route code は generated client を直接全面露出せず、必要なら local adapter で薄く整形する
- refactor 途中で page-local polling/state が global/shared へ漏れる
  - mitigation:
    `platform-frontend-architecture.md` の route-first/page-local boundary を守る

## Design Decisions

- frontend は TypeSpec-generated contract を使う
- browser-specific fetch concerns は local adapter ownership に留める
- route-first / page-local fetch-polling は current contract のまま維持する
