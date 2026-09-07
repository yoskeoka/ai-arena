# games-inline-bundle-upload
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0038-unactivated-game-bundle-retention.md`

## Objective

`/operator/games` で operator が game bundle ZIP を選択して upload し、admission 成功時に返る
immutable artifact identity と manifest 由来の game ID、game version、ruleset candidates を同じ画面で
確認して、選択した ruleset を activate できるようにする。operator は digest、game ID、game version を
転記せず、registration ID も入力せずに既存 service が導出する stable competition scope を作成できる。

完了境界は、有効な bundle で `file selection -> upload/admission -> manifest-derived review and
ruleset selection -> activation -> scope row refresh` が browser から成功し、invalid upload と activation
failure が form-local error として観測できることとする。admission 後に activation を完了しない artifact の
削除・TTL・garbage collection は実装しない。これは
`docs/issues/0038-unactivated-game-bundle-retention.md` に既知課題として残す。AI bundle flow、artifact
admission の validation rules、既存 metadata-only compatibility input、server API の atomic
upload-and-register endpoint は対象外とする。

## Current Evidence and Constraints

- `typespec/namespaces/operator/api.tsp:20-25` は operator-only の multipart
  `POST /api/v1/game-bundles` と 201 admission response を既に公開している。
- `typespec/namespaces/shared.tsp:128-155` で registration request は admitted `artifact_id` と
  `ruleset_version` を受け付け、admission response は manifest 由来の game identity、digest、
  supported rulesets を返す。
- `internal/platform/service/http.go:355-393` は upload の 64 MiB cap、admission、response を実装済みで、
  validation failure を保存前に返す。新規 service handler は不要である。
- `operator-ui/src/routes/operator/GamesPage.tsx:18-85` は現在 artifact digest、game ID、game version、
  ruleset、registration ID を手入力し、`createGameRegistration` を直接呼ぶ。これは artifact-backed
  activation の UX contract と一致しない。
- `operator-ui/src/lib/operatorApiClient.ts:100-113` は generated serializer を使う JSON activation seam を
  持つが、既存 multipart endpoint を browser `File` から呼ぶ adapter method はない。
- `docs/specs/platform-service-general-submission.md:27-39` は new game operation が manifest technical
  field や arbitrary artifact ref を form に持たず、admitted artifact を activate することを要求する。

## Black-box Spec Changes

`docs/specs/platform-service-general-submission.md` を、operator game flow が ZIP admission の成功 response
から artifact digest、game identity、選択可能な ruleset を受け、activation request はその admitted artifact
と operator が選んだ manifest-declared ruleset だけを使う、と明確化する。複数 ruleset がある bundle では
operator がその response の候補から一つを選ぶ。client が game ID/version/digest や legacy registration ID を
手入力・改変して activation してはならない。

`docs/specs/platform-service-operator-ui.md` を、Games page が file chooser、admission status/error、
manifest-derived read-only details、ruleset selector、activation action を提供し、successful activation 後に
scope list を refresh する observable behavior として更新する。upload 済みだが未 activation の artifact は
この task では cleanup されないことを scope 外として issue へ参照する。

TypeSpec の endpoint と request/response field inventory は既存 contract で足りるため変更しない。

## Code Change Map

- `docs/specs/platform-service-general-submission.md` (MODIFY)
  - browser game admission-to-activation contract と manifest-derived input boundary を記録する。
- `docs/specs/platform-service-operator-ui.md` (MODIFY)
  - Games page の upload/review/activate observation surface と browser acceptance を記録する。
- `operator-ui/src/lib/operatorApiClient.ts` (MODIFY)
  - generated model の `GameBundleAdmission` を UI-facing type として export し、credentialed multipart
    `FormData` upload を `/api/v1/game-bundles` へ送って 201 response を normalize する narrow adapter を追加する。
- `operator-ui/src/routes/operator/GamesPage.tsx` (MODIFY)
  - manual metadata/digest/registration fields を file chooser と admission result state に置き換える。
    upload 成功時に read-only game ID/version/digest を表示し、response の supported rulesets だけを selector
    に出す。activation は selected ruleset と admitted artifact ID だけを送る。upload / activation の pending
    state を重複送信から守り、失敗時は list を壊さず panel-local error を表示する。
- `operator-ui/tests/operator-ui.ci.spec.js` (MODIFY)
  - service-backed browser lane で実 bundle を file input から upload し、manifest-derived values を確認後、
    手入力 metadata なしで scope が作られることを確認する。fixture / actual service の asset provision が
    必要なら test helper と同じ test-only boundary を更新する。
- `operator-ui/tests/operator-ui.spec.js` (MODIFY, conditional)
  - fixture lane が Games page の new stable selectors を観測できるなら、route-only coverage に加えて
    upload error / ready surface を軽量に固定する。
- `docs/issues/0038-unactivated-game-bundle-retention.md` (NEW)
  - upload 成功後に activation されない immutable artifact cleanup を既知の follow-up として残す。
- `typespec/namespaces/operator/api.tsp` (NO CHANGE)
  - existing multipart upload contract を reuse する。
- `internal/platform/service/http.go` (NO CHANGE)
  - existing admission handler、size limit、validation、response contract を reuse する。

## Execution Steps

1. 先に上記 2 spec を更新し、browser が server-side manifest を再解析せず admission response を唯一の
   metadata source とし、ruleset 選択だけを operator に残す contract を固定する。未 activation artifact の
   retention issue を plan と PR body にリンクする。
2. generated client の ownership を崩さず、`operatorApiClient` に `File` を `FormData` の `bundle` part として
   credentialed upload する adapter を追加する。201 body を generated `GameBundleAdmission` transform で
   application shape に変換し、HTTP / malformed response を existing normalized error contract へ通す。
3. `GamesPage` を two-stage UI に変更する。選択 file を明示的に upload し、成功までは activation を不可に
   する。成功後は admission response の digest、game ID、version を read-only 表示し、ruleset candidates
   を selector にし、default は response の先頭 candidate にする。activation success 後は form state を
   意図的に reset し、competition scopes を reload する。upload をやり直した場合は前の admission selection を
   消して新 response だけを activation target にする。
4. service-backed browser test を real multipart request に更新し、valid game ZIP の upload、auto-populated
   values、ruleset 選択、activation、list row の artifact digest を end-to-end で確認する。server error と
   client error の少なくとも一方を component/fixture lane で確認し、failed admission が activation target を
   作らないことを固定する。
5. TypeSpec-generated client regeneration が必要な変更を含まないことを確認し、operator UI lint/typecheck と
   canonical local browser verification を実行する。実装 branch では completed plan を削除するが、0038 は
   unresolved follow-up として残す。

## Dependencies and Parallelism

- Step 1 は Step 2-4 より前に完了する。API / backend schema migration は dependency ではない。
- Step 2 と Step 3 は adapter shape を合意後に並行可能だが、同じ `operatorApiClient` / `GamesPage` write
  surface を触るため one worktree では直列にする。
- Step 4 は Step 2-3 に依存する。existing actual service bundle fixture を再利用できるかを最初に確認し、
  bundle build を test runtime に埋め込まない。
- Step 5 は全 implementation change に依存する。operator authorization、artifact store、multipart endpoint の
  current deployment configuration は staging manual confirmation の prerequisite だが、この plan の API
  contract dependency ではない。

## Verification

- operator UI unit/type/lint gate が pass し、TypeScript compile に handwritten DTO drift がない。
- service-backed Playwright lane で ZIP を添付し、201 admission response の digest、game ID、game version、
  ruleset candidates が displayed state に反映される。
- test は game ID、game version、artifact digest、registration ID を入力せず、manifest-declared ruleset を
  選んで activation し、作成 scope row に same artifact digest が表示されることを確認する。
- invalid/failed upload は actionable form-local error を表示し、activation action を enabled にせず、既存
  scopes list を維持する。
- supported rulesets が複数なら response にない ruleset を送れず、operator は one candidate を選べる。
- `POST /api/v1/game-bundles` の 64 MiB / validation semantics、existing compatibility JSON registration input、
  backend admission test は変更しないことを regression check で確認する。
- staging では operator 権限を持つ human が released Reversi ZIP を添付し、manual digest transcription
  なしに scope が作られることを確認できる。未 activation upload が残り得る点は 0038 の既知課題として evidence
  に明記する。

## Risks and Mitigations

- upload は admission に成功するが operator が activate しない
  - mitigation: cleanup はこの task に混ぜず `docs/issues/0038-unactivated-game-bundle-retention.md` で lifecycle
    / authorization / referenced digest protection を別途計画する。
- multiple rulesets を一つに暗黙選択すると意図しない scope を作る
  - mitigation: metadata fields は automatic にしつつ、ruleset は manifest response candidates から operator が
    明示選択できる selector にする。
- multipart を generated client seam の外へ散らすと base URL / credential / error normalization が分岐する
  - mitigation: browser-specific `FormData` handling は existing `OperatorApiClient` に閉じ、generated model
    transform と common request normalization を reuse する。
- upload 後に activation を retry すると stale artifact を送る
  - mitigation: new file selection / upload attempt は prior admission state を invalidate し、activation target は
    latest successful admission response だけにする。
