# TypeSpec client upgrade、staging release 復旧、upgrade skill

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

`online-release-staging` が TypeSpec-generated operator client の TypeScript 型検査で停止せず、対象 SHA の Pages/Render deploy と既存の version/worker-readiness 確認まで到達できるようにする。

今回の対象は、run `35011122951` で確認された current generator の未使用 `#context` 出力をまず安全に解消し、互換な TypeSpec 一式へ更新して同じ障害を generator 起因で再発させないことにある。さらに、今後の TypeScript/TypeSpec 更新で実際に成功した discovery、変更、検証だけを repo-local skill に記録する。

完了は次をすべて満たす時点とする。

- generated client と operator UI が strict TypeScript build を通る。
- TypeSpec source からの再生成後に committed artifact の drift がない。
- `typespec/**` または generated client/runtime dependency の変更は、push と manual dispatch のいずれの staging release 前にも専用 CI gate を通過しなければならない。
- 更新済み skill は実測した package compatibility、生成物差分の確認、ローカル/CI verification、失敗時の切り分けを含む。
- 最新 SHA の staging release が target version と worker readiness を確認して成功する。

対象外:

- operator/public HTTP wire contract の変更、`noUnusedLocals` の緩和、TypeSpec 以外の依存関係一斉更新、production release。
- TypeSpec emitter の upstream 修正を ai-arena から公開・採用決定すること。更新後にも障害が残る場合は、最小再現と version evidence を残して別途判断する。

Addresses: https://github.com/yoskeoka/ai-arena/actions/runs/35011122951 （staging release incident。外部 GitHub issue は作成しない。）

## 現状と原因

- `typespec/package.json:5-14` は `tsp compile` で OpenAPI と JS client を `operator-ui/src/generated/operator-api/` へ出力し、現行 pin は compiler/http/openapi3 `1.13.0`、rest `0.83.0`、http-client-js `0.15.0` である。
- `operator-ui/tsconfig.app.json:14-16` は `noUnusedLocals` を有効にしている。run `35011122951` の `Build operator UI` は generated `AiArenaClient.#context` と `SharedClient.#context` を未使用として `TS6133` で失敗した。
- `operator-ui/src/generated/operator-api/src/aiArenaClient.ts:92-102,243-248` は operation を持たない client に context を保存しており、現在の failure を再現する出力である。`5b9a6bb` で一度生成物だけを直したが、`ff85f5e` の再生成で戻った。
- `.github/workflows/operator-ui-browser.yml:3-27` と `.github/workflows/online-release-staging.yml:57-84` は `typespec/**` を required browser/typecheck input として扱わない。このため TypeSpec-only change が release 前の browser lane を起動せず、staging build で初めて検出された。
- `.github/workflows/online-release-staging.yml:114-123` の manual dispatch は resolved target SHA を即時 deploy 許可しており、現状では required workflow を検証しない。`.github/workflows/online-release-staging.yml:197-209` の push 側と同じ prerequisite 判断へ統合する必要がある。

## 仕様・運用契約

`docs/specs/platform-service-operator-ui.md` を先に更新し、operator UI が消費する generated client は TypeSpec source と committed artifact の組で扱い、strict TypeScript compile と generated-drift check を release 前の観測可能な delivery gate とすることを明記する。HTTP field inventory や public/operator API の振る舞いは変えないため、wire contract の TypeSpec source 自体はこの障害対策で変更しない。

## 変更マップ

- `(MODIFY) docs/specs/platform-service-operator-ui.md` — generated client の source-of-truth、strict compile、generation-drift/release gate の責務を browser delivery contract として追加する。
- `(MODIFY) typespec/package.json`, `typespec/pnpm-lock.yaml`, `typespec/pnpm-workspace.yaml` — TypeSpec compiler/libraries/JS emitter を互換な組で更新し、lock と minimum-release exception を同期する。
- `(MODIFY) operator-ui/package.json`, `operator-ui/pnpm-lock.yaml` — emitter が要求する generated runtime dependency を root UI consumer と lock に同期する。
- `(MODIFY) operator-ui/src/generated/operator-api/**`, `typespec/generated/openapi/operator/openapi.json` — 更新済み emitter による再生成結果だけを commit する。恒久的な手編集を残さない。
- `(NEW) tools/dev/verify-typespec-generated-client.sh` — clean checkout で frozen TypeSpec install/compile、tracked/untracked の generated-artifact drift check、operator UI frozen install/strict build を順に実行する失敗終了の単一入口を置く。
- `(MODIFY) Makefile` — 上記 script を呼ぶローカル/CI 共通 target を追加する。
- `(NEW) .github/workflows/typespec-client.yml` — TypeSpec、generated client、runtime dependency、verification script/workflow の変更で共通 target を実行する軽量 CI gate を追加する。
- `(MODIFY) .github/workflows/online-release-staging.yml` — 同じ入力群で `typespec-client` workflow を release prerequisite に加える。既存の Go/browser prerequisite と deploy/version/readiness contract は変更しない。
- `(NEW) .claude/skills/typespec-client-upgrade/SKILL.md` — 実施後の evidence を基に、TypeSpec/TypeScript dependency update の discovery、更新、generated artifact review、verification、復旧判断を再利用可能な手順として残す。

## 実施手順

1. **契約と暫定復旧を先に閉じる**
   - spec を更新してから、現行 pin の generator output で未使用 context を除く最小の generated-client correction を行う。
   - `pnpm --dir operator-ui install --frozen-lockfile` と `pnpm --dir operator-ui run build` を実行し、`TS6133` が解消することを確認する。
   - この correction は release unblocking のための明示的な中間 commit とする。`noUnusedLocals` を下げず、public/operator client API の形・runtime behavior を変えない。

2. **TypeSpec 一式を互換な組で更新して再生成する**
   - npm registry と emitter peer dependency を読んで、compiler/http/rest/openapi3/http-client-js を同一 compatibility line へ上げる。JS emitter だけを更新しない。
   - 更新後 emitter が出力する generated package dependency を確認し、operator UI root の `@typespec/ts-http-runtime` を必要時に同じ compatibility line へ同期する。
   - `pnpm --dir typespec install` で lock を更新後、`pnpm --dir typespec run build` で OpenAPI と JS client を再生成する。temporary correction と同等以上の compile-safe output になることを確認し、手編集で generator の欠陥を覆い隠さない。
   - public/operator の exported client method、wire path、model serialization が意図せず変わらないことを、generated diff と既存 operator UI imports (`operator-ui/src/lib/operatorApiClient.ts`) から確認する。互換性を満たす update が得られない場合は、暫定 correction だけを release fix として分離し、取得した versions/peer dependencies/minimal reproduction を plan-linked local issue に残して upgrade を推測で完了させない。

3. **再発を CI gate にする**
   - verification script/Make target を、TypeSpec frozen install → TypeSpec compile → generated output の tracked diff と untracked file 検査 → operator UI frozen install → `pnpm run build` の順に固定する。artifact directory 内の `git diff --exit-code` だけでなく、同 directory に untracked file がないことも失敗条件にする。
   - dedicated workflow の path filter と staging release gate の input pattern を一致させる。少なくとも `typespec/**`、generated client、operator UI の runtime package/lock、verification script、workflow 自体を対象にする。
   - `typespec-client` の failure は staging deploy の前に prepare job を失敗させる。manual dispatch でも resolved target SHA の parent diff から同じ required-workflow set を導出し、diff を安全に取得できない場合は gate を迂回して deploy しない。browser E2E をこの軽量 compile/drift gate の代替にせず、既存 browser lanes は維持する。

4. **成功した upgrade 知識を skill にする**
   - skill は Step 2-3 の成功後に作成する。未検証の version や一般論は書かず、実際に採用した package set、peer dependency の確認方法、lock update、generator output の確認点、共通 verification command、CI/release gate、失敗時の分岐だけを記録する。
   - generated file の恒久手編集禁止、TypeScript strictness を緩めないこと、emitter-only update をしないこと、temporary fix を分離する条件を guard として明記する。
   - skill の指示に沿って再実行した verification output を確認し、skill 自身が今回の回避策だけでなく次回の安全な upgrade 手順として使えることを検証する。

5. **release と handoff**
   - local/CI gate 成功後に PR を作成し、最新 head の required checks と review を確認する。
   - merge 後は `online-release-staging` の target SHA を確認し、Pages/Render deploy が実行され、`/version` がその full SHA、`/healthz` の API/worker が `OK` となることを確認する。staging release の pending/skip は成功扱いにしない。
   - 実装 PR ではこの plan を削除し、plan PR と Git history から追跡可能にする。

## 依存関係・順序

1. spec update と temporary correction は、staging unblock の最短経路として先行する。
2. package compatibility discovery と CI script/workflow design は並行可能だが、generated output の最終 commit は package update/rebuild 後に限る。
3. skill は successful update と verification evidence に依存するため最後に作る。
4. staging acceptance は merge と latest-head CI の後であり、ローカル build や dispatch acceptance のみでは代替しない。

## 検証

- current-pin temporary correction: `pnpm --dir operator-ui install --frozen-lockfile && pnpm --dir operator-ui run build`
- updated compatibility set: `pnpm --dir typespec install`, `pnpm --dir typespec run build`, generated artifact diff review, `pnpm --dir operator-ui install --frozen-lockfile`, `pnpm --dir operator-ui run build`
- generated drift: clean checkout で共通 verification target を実行し、TypeSpec frozen install が lock どおりに完了すること、`typespec/generated/openapi/operator/` と `operator-ui/src/generated/operator-api/` に tracked diff と untracked file のいずれもないことを確認する。
- CI contract: TypeSpec-only change と generated-client/runtime-only change の双方で `typespec-client` が起動すること、push と manual dispatch の staging prepare が同 workflow の failure を deploy 前に拒否することを workflow test/fixture で確認する。manual dispatch は resolved target SHA の parent diff を使用し、diff 取得不能時に deploy を許可しないことも確認する。
- regression: generated `AiArenaClient` と `SharedClient` の unused context による `TS6133` が起きないことを strict build で確認し、`operator-ui/src/lib/operatorApiClient.ts` の imports/typecheck を保つ。
- release acceptance: latest merged SHA の `online-release-staging` で DB migration、operator UI build、Pages deploy、Render trigger、exact `/version`、ready `/healthz` がすべて success であることを GitHub Actions logs から確認する。
- quality/PR: applicable workflow lint、skill/document lint、dedicated CI、existing required checks、`review-task` の latest-head follow-up を完了する。
