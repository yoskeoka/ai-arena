# TypeSpec generated client の空 context 後処理

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

TypeSpec の HTTP client emitter が operation を持たない client に出力する未使用 `#context` により、再生成のたびに operator UI の strict TypeScript build が壊れる状態を解消する。`typespec` の正式な build entrypoint が emitter 出力を完了後に決定的に正規化し、再生成・CI・staging release が同じ strict-build-safe artifact を扱うようにする。

完了は次をすべて満たす時点とする。

- `pnpm --dir typespec run build` を clean checkout で実行しても、`AiArenaClient` と `SharedClient` の unused private context による `TS6133` を再導入しない。
- 後処理は既知の empty-client 出力だけに限定し、対象 shape が変わった場合や対象外の generated code を変更しようとした場合は成功を装わず failure になる。
- TypeSpec source から再生成した committed OpenAPI/client artifact に drift がなく、operator UI の strict build が通る。
- TypeSpec/generated-client に関わる変更は push と `workflow_dispatch` の別を問わず staging deploy の前に上記再生成・drift・strict build を検証し、最新 merged SHA の staging release が Pages build と既存の version/worker-readiness 確認まで成功する。

対象外:

- operator/public HTTP wire contract、client の公開 operation、runtime behavior、`noUnusedLocals`、TypeSpec package version の変更。
- upstream TypeSpec emitter の修正そのもの、または production release。

Addresses: `docs/issues/0124-typespec-http-client-empty-context-output.md`、https://github.com/yoskeoka/ai-arena/actions/runs/35036665520 （staging release incident）。

## 現状と原因

- `typespec/package.json:5-14` の `build` は `tsp compile` の直後に Prettier を実行するだけであり、emitter が出した unused context を正規化する段階がない。
- `typespec/tspconfig.yaml:9-18` は `@typespec/http-client-js` の output を `operator-ui/src/generated/operator-api` に固定し、upstream issue #11978 の空 client 出力を既知の blocker として記録している。
- `operator-ui/src/generated/operator-api/src/aiArenaClient.ts:93-101,244-249` は operation を持たない `AiArenaClient` / `SharedClient` に private context と初期化を出力する。`operator-ui/tsconfig.app.json:14-16` の strict unused-local check はこれを `TS6133` として拒否する。
- commit `019a48f` は generated file を直接直して staging build を復旧したが、commit `c389d99` の再生成でその修正が戻った。run `35036665520` の `deploy-staging / Build operator UI` は同じ failure で deploy 前に停止した。
- `docs/issues/0124-typespec-http-client-empty-context-output.md:29-35` は恒久手編集を採らない旧判断を記録している。今回の決定的 postprocess は source build に組み込み、手作業と区別して同記録を更新する必要がある。

## 仕様・運用契約

HTTP field inventory と browser product behavior は変更しないため、`docs/specs/` の black-box contract 変更はない。生成・release mechanics を behavioral UI spec に混在させない。

代わりに development document で次を運用契約として定める。

- TypeSpec build は emitted artifact を唯一の defined postprocess で正規化してから format する。生成済み client の手編集は許可しない。
- 正規化器は `aiArenaClient.ts` の、operation を持たない既知 client の unused context/import/initializer だけを処理する。emitter の shape が変わる場合は no-op 成功ではなく、upstream fix を評価するための failure を返す。
- generator、正規化器、emitted artifact、または TypeSpec dependency の変更は、再生成後の artifact drift check と operator UI strict build を staging prerequisite として通す。
- upstream が strict-compile-safe な output を提供した時点で、postprocess の不要性を clean regeneration と strict build で検証し、別 plan で撤去を判断する。

## 変更マップ

- `(NEW) docs/development/typespec-generated-client.md` — generated client の ownership、決定的 postprocess、upstream-removal 条件、local/CI verification の運用契約を記録する。
- `(MODIFY) docs/development/README.md` — 新しい generated-client 運用文書を development docs の索引に追加する。
- `(MODIFY) docs/issues/0124-typespec-http-client-empty-context-output.md` — direct hand edit を置き換える採用判断、postprocess の限定性、#11978 を monitor して撤去を別判断する条件に更新する。
- `(NEW) typespec/tools/normalize-empty-client-context.mjs` — emitted `aiArenaClient.ts` を構造的に検査し、空 `AiArenaClient` / `SharedClient` のみから unused context と専用 import を除去する fail-closed postprocessor を置く。
- `(MODIFY) typespec/package.json` — `build` を compile → normalize → format に固定し、後処理を contributor と CI の共通 entrypoint にする。
- `(MODIFY) operator-ui/src/generated/operator-api/src/aiArenaClient.ts` — 上記 entrypoint の出力だけを commit し、unused field/import/initializer を残さない。
- `(NEW) tools/dev/verify-typespec-generated-client.sh` — frozen TypeSpec install、TypeSpec build、generated artifact drift check、frozen operator UI install、strict build を順に実行する fail-fast verifier を置く。
- `(MODIFY) Makefile` — verifier の repo-local target を追加して local/CI entrypoint を共有する。
- `(NEW) .github/workflows/typespec-generated-client.yml` — TypeSpec、postprocessor、generated client、TypeSpec/runtime lock、verifier、workflow の変更で verifier を実行する dedicated CI gate を置く。
- `(MODIFY) .github/workflows/online-release-staging.yml` — 上記と同じ path set では `typespec-generated-client` の latest successful push run を deploy prerequisite に加え、failure 時は Pages build より前に release を停止する。

## 実施手順

1. **運用契約と blocker record を先に更新する**
   - development document に source-of-truth、許容される generated-output transformation、fail-closed 条件、upstream removal criteria、verification sequence を記す。
   - issue 0124 の旧判断を、手作業禁止を維持したまま build-owned deterministic postprocess を採用する判断に更新する。#11978 の追跡を続け、postprocess を emitter fix と偽らない。

2. **限定的かつ fail-closed な正規化器を実装して build に接続する**
   - `aiArenaClient.ts` を読み、expected generated imports、empty client class bodies、context assignment がすべて揃う current emitter output だけを受理する。
   - `AiArenaClient` と `SharedClient` について、unused private field、context creation import、context type import、constructor initializer を対応するまとまりとして除去する。constructor parameter が unused になる場合は TypeScript strict compiler が許容する unused-parameter shape に変更する。
   - `OperatorClient` / `PublicClient`、operation method、model import、endpoint/options が runtime behavior に必要な class は変更しない。既知 token の不足、複数一致、意外な class body、または postprocess 後にも target token が残る場合は nonzero exit で止める。
   - package script を `tsp compile`、normalizer、Prettier の順にして、直接の `tsp compile` だけを日常の regenerated artifact source と見なさない。再生成 artifact を更新し、手修正 diff を残さない。

3. **reproduction と drift を共通 verifier にする**
   - verifier はまず `pnpm --dir typespec install --frozen-lockfile` で独立した TypeSpec workspace を clean checkout に materialize してから TypeSpec build を実行する。続いて `typespec/generated/openapi/operator/` と `operator-ui/src/generated/operator-api/` の tracked/untracked drift を検査し、operator UI の frozen install と `pnpm run build` を行う。
   - normalizer の current-shape fixture/shell test を追加し、expected output は正規化し、unexpected or already-fixed emitter output は明確に failure となることを確認する。postprocessor が黙って広い出力を壊せないことを regression で固定する。
   - Make target は verifier を呼ぶだけにし、developer と CI の command sequence を分岐させない。

4. **CI と staging release gate を同期する**
   - dedicated workflow の path filter と staging `requiredWorkflows` の pattern を同一の変更集合にする。少なくとも `typespec/**`、generated client、operator UI package/lock の generated runtime、verifier/Make target、workflow file を含める。
   - staging prepare は push と `workflow_dispatch` の両方で target SHA の relevant change を判定し、該当時の `typespec-generated-client` failure、missing、pending を deploy 前に拒否する。手動dispatch は prerequisite の bypass ではない。changed-file comparison または required-workflow lookup を検証できない場合も default deploy へ fall back せず、fail-closed で停止する。既存 Go/browser prerequisite と deploy/version/health contract は維持する。
   - workflow lint の test seam がある場合は、TypeSpec-only change と manual dispatch target の双方で gate が選ばれること、workflow lookup failure と gate failure のどちらも staging deploy job を作らないことを追加して確認する。

5. **verification、PR、staging acceptance を閉じる**
   - local verifier と normalizer regression、applicable workflow lint を通し、generated diff が normalizer の出力だけであることを review する。
   - implementation PR ではこの plan と issue 0124 を削除し、plan PR/implementation PR/Git history から判断を追跡可能にする。
   - latest head の required CI と review を `review-task` で確認し、merge 後に `online-release-staging` の target SHA で operator UI build、Pages deploy、Render deploy、exact `/version`、ready `/healthz` がすべて success になるまで確認する。

## 依存関係・並行性

1. Step 1 は Step 2 より先に行う。product spec 更新は不要だが、build ownership の開発文書と blocker record は code より先に確定する。
2. normalizer implementation（Step 2）と verifier/workflow test design（Step 3-4）は並行可能である。ただし generated artifact の最終更新と CI gate 接続は、normalizer regression が完了してから行う。
3. dedicated gate の actual workflow success は staging prepare acceptance の前提である。
4. upstream issue の解決は本計画に依存しない。upstream fix 採用/撤去は別 plan とする。

## 検証

- normalizer unit/fixture regression: current empty-client fixture が expected artifact になり、missing/duplicate/unexpected/already-fixed shape が nonzero exit になること。
- generation: `pnpm --dir typespec run build` の直後に `operator-ui/src/generated/operator-api/src/aiArenaClient.ts` に unused `#context`、関連する unused context imports、or their initializers が残らないこと。
- strict consumer: `pnpm --dir typespec install --frozen-lockfile`、`pnpm --dir operator-ui install --frozen-lockfile`、`pnpm --dir operator-ui run build` が `TS6133` なしで通ること。
- drift: common Make/verifier target が clean checkout で TypeSpec/OpenAPI/generated-client drift なしを確認し、意図的な artifact mutation を failure にすること。
- workflow selection: TypeSpec-only、normalizer-only、generated-client-only、operator runtime dependency-only の各 diff と、それらを target にした manual dispatch で dedicated workflow が起動し、staging prepare が同 workflow の completed success を要求すること。changed-file comparison または workflow lookup を検証できない場合は deploy を開始しないこと。
- release: latest merged SHA の `online-release-staging` が DB migration、operator UI build、Pages deploy、Render trigger、target full SHA の `/version`、API/worker `OK` の `/healthz` まで success であること。
- quality/PR: applicable document/workflow lint、required CI、`review-task` の latest-head follow-up を完了すること。
