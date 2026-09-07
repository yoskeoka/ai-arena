# reversi-submission-zip-upload
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

`/operator/submissions` で authenticated operator が competition scope を選び、AI bundle ZIP を添付して
admission し、成功 response の immutable artifact identity を使って new bot または existing bot の revision を
登録できるようにする。operator は artifact digest、`artifact_ref`、AI submission ID、runtime/AI identity を
転記しない。

完了境界は、有効な scope-compatible AI bundle で `scope and bot choice -> file selection -> upload/admission ->
read-only artifact confirmation -> create/revise -> bot list refresh` が browser から成功することとする。upload
failure は bot/revision を作らず、form-local error として観測できることも含む。legacy AI submission の HTTP
create/list API とその deletion、既存 legacy record を参照する match-request migration、artifact retention/cleanup、
AI admission validation、backend の新規 endpoint、game upload/activation は対象外とする。legacy API の removal
readiness は `docs/issues/0040-legacy-ai-submission-retirement.md` で別途扱う。

## Current Evidence and Constraints

- `typespec/namespaces/operator/api.tsp:51-56` は multipart `POST /api/v1/ai-bundles` を既に公開し、scope
  (`game_registration_id`) と optional display name と bundle を受けて 201 `AISubmission` を返す。
- `internal/platform/service/http.go:322-355` は multipart body を 64 MiB に制限して既存 admission service へ渡す。
  `internal/platform/service/general.go:122-161` は selected scope との game/major/ruleset/runtime compatibility を
  admission 時に検証して immutable artifact を保存する。新規 backend handler は不要である。
- `operator-ui/src/routes/operator/SubmissionsPage.tsx:13-103` の durable bot form は artifact ID を手入力して
  `createOrReviseBot` を呼ぶ。一方 `:50-99` と `:135-153` は legacy `ai-submissions` create/list と
  `artifact_ref` の手入力を表示している。
- `operator-ui/src/lib/operatorApiClient.ts:152-166` は bot create/revise/list の adapter を持つ。generated client
  は `operator-ui/src/generated/operator-api/src/api/operatorClientOperations.ts:345-380` で AI bundle multipart
  operation を持つが、browser route 用 adapter は未公開である。
- `operator-ui/src/routes/operator/GamesPage.tsx:18-132` は同じ two-stage file upload/admission/activation UX の
  reference である。AI flow は admission response の artifact ID を bot revision に渡す点だけが異なる。
- current local/CI browser workflow と artifact-backed service E2E は `POST /api/v1/ai-bundles` / `RegisterAIBundle`
  を使う。legacy endpoint を呼ぶ browser E2E は確認されていないが、`internal/platform/service/request.go:286-326` は
  legacy `AISubmissionID` を `ArtifactRef` に解決する match-request path を残すため removal safety は別途監査が必要である。
- `docs/specs/platform-service-general-submission.md:27-49` は AI form を uploaded artifact に限定し、legacy
  input を新規 operator operation の正本にしないことを要求する。

## Black-box Spec Changes

`docs/specs/platform-service-general-submission.md` を、AI browser flow が selected scope と ZIP を admission に渡し、
成功 response の artifact identity だけで bot create/revise すること、admission failure が bot/revision を作らない
こと、legacy AI submission HTTP surface が existing match-request migration/replay 用の compatibility input であり
new operator UI の入口ではないことを明確化する。

`docs/specs/platform-service-operator-ui.md` を、Submissions page の file chooser、admission/read-only digest、
new/revision action、refresh/error behavior と legacy UI の非表示を observable behavior として更新する。

TypeSpec、generated client、backend HTTP handler は既存 multipart contract で足りるため変更しない。

## Code Change Map

- `docs/specs/platform-service-general-submission.md` (MODIFY)
  - ZIP admission-to-bot-revision contract と legacy compatibility boundary を記録する。
- `docs/specs/platform-service-operator-ui.md` (MODIFY)
  - Submissions page の two-stage observation surface と browser acceptance を記録する。
- `operator-ui/src/lib/operatorApiClient.ts` (MODIFY)
  - `AISubmission` を UI-facing type として export し、credentialed `FormData` の `bundle`,
    `game_registration_id`, `display_name` を `/api/v1/ai-bundles` へ送る narrow adapter を追加する。201 response は
    generated transform と existing error normalization を通す。
- `operator-ui/src/routes/operator/SubmissionsPage.tsx` (MODIFY)
  - manual artifact ID と legacy create/list panel を、scope-bound ZIP upload、admission state、read-only admitted
    artifact confirmation、bot create/revise に置き換える。latest successful admission の artifact ID だけを bot
    request に使い、file replacement/upload failure は prior admission を invalidate する。
- `operator-ui/tests/operator-ui.ci.spec.js` (MODIFY)
  - service-backed browser lane で scope-compatible AI ZIP を添付し、admission digest の確認、new bot/revision、
    bot row refresh、failed upload が bot action を可能にしないことを確認する。実 bundle fixture は existing
    test-only provision seam を再利用する。
- `operator-ui/tests/operator-ui.spec.js` (MODIFY, conditional)
  - fixture lane に stable selectors を追加できる場合だけ、Submissions route の upload-ready/error surface を固定する。
- `docs/issues/0040-legacy-ai-submission-retirement.md` (NEW)
  - legacy AI submission API/record/path の production data、migration/replay、direct tests/fixtures、generated client
    dependency を棚卸しし、removal preconditions と migration/retention strategy を記録する。
- `typespec/namespaces/operator/api.tsp` (NO CHANGE)
  - existing `POST /api/v1/ai-bundles` multipart contract を reuse する。
- `internal/platform/service/http.go` (NO CHANGE)
  - existing size limit、admission、201 response contract を reuse する。
- `POST /api/v1/ai-submissions` (NO CHANGE)
  - existing match-request migration/replay compatibility API と legacy records を維持するが、new UI からは呼ばない。

## Execution Steps

1. 先に上記 spec を更新し、new UI の唯一の artifact source を admission response とし、legacy AI submission
   create/list は existing migration/replay compatibility だけに残る境界を固定する。API field inventory は TypeSpec
   のままにする。
2. `OperatorApiClient` に browser-specific multipart adapter を追加する。selected `File` と selected scope、bot
   name を form parts として credentialed upload し、201 body を generated `AISubmission` transform で application
   shape に変換する。HTTP/malformed response は common normalized error path を使う。
3. `SubmissionsPage` を two-stage UI に変更する。scope、new/existing bot choice、bot name、AI ZIP を入力し、
   selected file がなければ upload を拒否する。upload success 後に admitted artifact digest と admission metadata を
   read-only 表示し、create/revise action は latest success のみを artifact ID として使う。submit success は form
   and admission state を reset して selected scope の bot list を reload する。
4. legacy `listAiSubmissions`/`createAiSubmission` invocation、legacy fields、legacy list/panel を page から削除する。
   backend/TypeSpec/generated client の legacy endpoint は変更しない。`0040` に removal readiness の evidence と
   follow-up boundary を記録する。既存 legacy match request と current bot composition/ranking identity はこの UI
   change で mutation してはならない。
5. service-backed Playwright lane を actual multipart AI bundle で更新する。selected scope に compatible な bundle
   が admission され、displayed artifact ID が bot create/revise request に使われ、new bot と existing revision が
   stable bot identity を保つことを確認する。invalid/incompatible upload が prior admission を使えず bot mutation
   を発生させないことを固定する。
6. TypeSpec generation/backend implementation が不要であることを確認し、operator UI build と canonical browser
   verification を実行する。implementation branch では completed plan を削除する。

## Dependencies and Parallelism

- Step 1 は Step 2-5 より前に完了する。backend/schema migration と TypeSpec regeneration は dependency ではない。
- Step 2 と Step 3 は adapter shape の合意後に並行可能だが、同じ UI state に関わるため one worktree では直列にする。
- Step 4 は Step 3 と同一 page write surface を触るため直列にする。legacy API removal は依存せず、実行しては
  ならない。
- Step 5 は Step 2-4 に依存する。fixture が selected scope-compatible AI bundle を supply する既存 seam を先に
  確認し、browser test 内で artifact build を新設しない。
- Step 6 はすべての implementation change に依存する。staging での released Reversi AI ZIP の human verification
  は deploy 後の acceptance であり、automated diagnostic preset lane の責務には混ぜない。

## Verification

- operator UI build/typecheck が pass し、handwritten adapter と generated DTO に drift がない。
- service-backed Playwright lane が AI ZIP を file input から selected scope へ upload し、201 admission response の
  artifact digest を read-only state に表示する。
- test は artifact digest/`artifact_ref`/AI submission ID を入力せず、new bot を作成して bot row に active revision
  が反映されること、次の compatible ZIP で same `bot_id` を revision して ranking identity を変えないことを確認する。
- missing, invalid, or scope-incompatible ZIP は actionable form-local error を表示し、create/revise action を有効に
  せず、existing bot list と prior admitted artifact target を維持しない。
- legacy section、`Artifact Ref` input、`Create AI submission` action は `/operator/submissions` に表示されず、
  existing `POST /api/v1/ai-submissions` compatibility API と historical records は変更しない。
- staging では operator が released Reversi AI ZIP を attach し、existing Reversi scope に new bot/revision を
  artifact transcription なしで登録できる。game upload/admission/registration の 0109 evidence と diagnostic
  `online-release-staging-verify` をこの acceptance の代替にしない。

## Risks and Mitigations

- admission succeeded but create/revise is not completed, leaving an unreferenced immutable AI artifact
  - mitigation: artifact retention/cleanup は game bundle と同様に別 lifecycle task とし、本 change に delete/TTL を
    混ぜない。
- selected file/scope changes after admission could submit a stale or incompatible artifact
  - mitigation: file または scope の変更と every upload attempt は prior admission state を clear し、latest successful
    response だけを create/revise target にする。
- removing the legacy panel could be mistaken for removing migration compatibility
  - mitigation: page invocation/display だけを削除し、TypeSpec、HTTP handler、legacy records と existing match-request
    migration behavior は unchanged boundary として test/PR description に明記する。API removal の可否は `0040` の
    data/dependency audit 完了後にだけ判断する。
- multipart handling could diverge from base URL, credentials, or error normalization behavior
  - mitigation: `OperatorApiClient` の narrow adapter に閉じ、Games page と同じ credentialed form/error policy を reuse
    する。
