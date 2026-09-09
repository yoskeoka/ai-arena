# online-release-production-readiness-rollback
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

production release を staging と同じ backend evidence で閉じる。`online-release-production.yml` は
deploy 前に current serving backend の full SHA を保存し、target SHA の `/version` と
`/healthz` (`api=OK`, `worker=OK`) を確認する。target verification が失敗した場合は、保存した
previous SHA を Render deploy hook で再デプロイし、rollback backend も同じ version/readiness
contract で確認する。

完了境界は、production workflow summary に target、previous、observed、rollback（該当時）の full
SHA と evidence を残し、target 成功または previous SHA への verified backend rollback のどちらかで
workflow を終えることである。target failure は rollback が成功しても release failure として記録する。

この plan は backend runtime の deployment identity、worker readiness、code rollback に限定する。
Cloudflare Pages frontend の deployed commit identity と frontend rollback の証明は別 exec-plan とし、
この plan は frontend が reachable であることを backend success の根拠にしない。

## Preconditions and Current References

この plan は次の implementation が `main` にあることを前提とする。

- `0114-online-release-version-verification`
  - public `/version`、full SHA build identity、version polling helper。
- `0115-worker-lock-retry-on-render-rollout`
  - bounded single-worker handoff、queue safety。
- `0116-online-release-worker-readiness-verification`
  - public HTTP `200` liveness と api/worker body、health polling helper。

current references:

- `.github/workflows/online-release-production.yml:1-154`
  - tag/dispatch input、migration、Pages upload、Render deploy hook、summary の current production path。
- `.github/workflows/online-release-staging.yml:271-294`
  - target SHA deploy hook と staging evidence path。production helper reuse の reference。
- `tools/dev/wait-for-remote-version.sh` と `tools/dev/wait-for-remote-health.sh`
  - prerequisites が追加する bounded public endpoint verification helpers。
- `docs/development/platform-service-online-deploy.md:454-590`
  - production release、rollback、migration compatibility の runbook 正本。
- `README.md:94-99`
  - production tag は same-SHA staging verification 後に作る contract。

## Adopted Design

### Capture and validate the rollback target

production workflow は migration や deploy hook の前に `${PRODUCTION_BACKEND_URL}/version` を取得する。

- response は valid JSON、full 40-character SHA、repository 内で canonical に解決できる commit でなければならない
- value を `previous_version_sha` として job output に保存する
- current version が endpoint 未導入の legacy deployment で取得できない最初の rollout だけは、manual dispatch の
  required `previous_commit_sha` input を使う。入力値も canonical full SHA に正規化・検証する
- push/tag trigger では legacy fallback を暗黙に推測しない。endpoint を持つ既知の previous revision が
  なければ production deploy を開始せず、operator が explicit dispatch と previous SHA を指定する
- target SHA と previous SHA が同じ場合は no-op rollback を起動せず、target verification を行った上で summary に記録する

`previous_version_sha` は code rollback の target であり、DB migration の rollback target ではない。production
migration は既存 runbook の backward-compatible expand/dual-read-write contract を満たさなければならず、
code rollback で schema を戻してはならない。

### Target verification and automatic backend rollback

production target deploy hook の直後に、prerequisite helper を順に実行する。

1. `/version.version_sha == target_sha` を 15 秒 interval・最大 20 分で確認する
2. target version が一致した後、`/healthz` が HTTP `200`、`api=OK`、`worker=OK` を 10 秒 interval・最大 7 分で確認する

version/health timeout、mismatch、malformed body、HTTP/transport error は target verification failure とする。
failure 時は次を順に行う。

1. `previous_version_sha` が target と異なることを確認する
2. `ref=${previous_version_sha}` を付けて production Render deploy hook を 1 回起動する
3. same `/version` と `/healthz` helpers で previous SHA と ready worker を確認する
4. target/previous/last observed fields と rollback result を `always()` summary に残す

rollback hook、rollback version verification、rollback readiness verification のいずれかが失敗した場合は
workflow を incident failure として終了する。rollback が成功しても target release は failed とし、tag を
release success として扱わない。automatic retry loops や alternate SHA の探索は行わない。

### Production scope and operator boundary

- `RENDER_PRODUCTION_DEPLOY_HOOK_URL` だけを deploy/rollback mutation identity とする。Render API token、
  machine account、OIDC provider は追加しない。
- workflow は deploy hook secret、access cookie、connection string を summary/log に出力しない。
- target preflight と rollback target は public `/version` だけで読み取る。operator に SHA を転記させない。
- existing Pages upload は維持するが、frontend artifact の target identity と rollback は proof scope 外である。
- target backend failure 後の frontend/backend version skew は backward-compatible API release contract で短時間許容し、
  frontend identity/rollback plan が導入されるまで production promotion evidence と混同しない。

## Code and Documentation Change Map

- `(MODIFY) .github/workflows/online-release-production.yml`
  - legacy-aware previous SHA input を追加する。
  - migration/deploy 前に current backend version を capture / validate する。
  - target version/readiness verification、single previous-SHA rollback、rollback verification を追加する。
  - `always()` summary に target/previous/observed/rollback evidence を安全に残す。
- `(MODIFY) tools/dev/wait-for-remote-version.sh`
  - production/staging 共用の expected SHA、last observation、exit-code contract を確認し、必要なら reusable inputs を追加する。
- `(MODIFY) tools/dev/wait-for-remote-health.sh`
  - production/staging 共用の api/worker readiness、last observation、exit-code contract を確認し、必要なら reusable inputs を追加する。
- `(NEW) tools/dev/validate-release-commit-sha.sh`
  - full SHA、repository reachability、target/previous non-equality を shell-safe に検証する helper を追加する。
- `(MODIFY) docs/development/platform-service-online-deploy.md`
  - production preflight、legacy bootstrap dispatch、target verification、single rollback attempt、DB rollback exclusion、incident handling を記録する。
- `(MODIFY) README.md`
  - tag-triggered production release の success/failure と verified backend rollback boundary を簡潔に更新する。
- `(MODIFY) docs/specs/platform-service-operator-ui.md`
  - public version/readiness endpoint が production deployment evidence に使われることを記録する。
- `(DELETE) N/A`

## Black-Box Specification Changes

### Production backend release completion

- target success: `/version` が target full SHA、続く `/healthz` が HTTP `200` / `api=OK` / `worker=OK`
- target failure: target backend is not accepted; previous SHA がある場合は exactly one rollback deploy を試みる
- rollback success: previous full SHA と ready worker が観測される。workflow は target failure として non-zero で終了する
- rollback failure: incident failure。workflow は recovery succeeded を主張しない
- initial legacy bootstrap: operator-supplied `previous_commit_sha` が full/reachable SHA でなければ deploy mutation を開始しない
- schema: code rollback は schema rollback を含まない。migration compatibility は release 前提である

### Evidence boundary

production summary は少なくとも target SHA、previous SHA、target version/readiness observation、rollback attempt
の有無と result、backend URL、verification helper artifact/log location を含む。secret、cookie、deploy hook URL、
database DSN を含めない。frontend deployment identity はこの contract の証明対象外である。

## Subtasks and Dependencies

1. `0114`、`0115`、`0116` の latest-main implementation と staging acceptance evidence を確認する。
2. production release spec/runbook に preflight、target verification、single rollback、schema boundary を先に記録する。
3. SHA validation と version/health helper の reusable exit-code contract を実装し、local fake-server test を追加する。
4. production workflow に previous capture、target verification、rollback/verification、safe summary を接続する。
5. explicit legacy bootstrap dispatch、target success、target failure plus rollback success、rollback failure を dry-run/local script test で確認する。
6. approved staging rehearsal の後、production では human-approved maintenance window に 1 回 target success path を acceptance する。

Steps 2 and 3 は並行できる。Workflow mutation is dependent on both. Production acceptance は staging
rehearsal と all repository quality gates の後にだけ行う。

## Verification

- local fake HTTP server で target/previous version、ready/pending/malformed health、timeout を helper-level に確認する。
- SHA helper が short SHA、unknown SHA、target==previous、legacy dispatch input を拒否/受理する境界を確認する。
- workflow syntax / shell lint、`./tools/workflow-lint.sh --mode=pre-push`、`git diff --check` を実行する。
- `make test`、`make lint`、textlint を実行する。
- staging rehearsal で target failure の single rollback sequence を non-production hook / controlled endpoint で確認し、production secret を使わない。
- human-approved production acceptance で target success を 1 回確認し、summary に full SHA evidence が残ることを確認する。
- production target failure scenario は live incident を意図的に起こさない。rollback logic の failure branches は local/staging rehearsal で確認する。

## Non-goals and Rejection Conditions

- frontend deployed commit identity、Pages rollback、frontend/backend skew の恒久解消を実装しない。
- database down migration、destructive data rollback、release artifact deletion を行わない。
- Render API token、OIDC、staging/production machine account、additional operator credential を追加しない。
- previous SHA 以外への自動 fallback、unbounded retry、automatic re-promotion を行わない。
- staging verification を省略して production tag を成功扱いにしない。
