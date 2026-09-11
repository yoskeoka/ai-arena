# wasi-bundle-admission-e2e-regression
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

preset に依存しない operator general lane を、実際の `arena-bundle/v1` ZIP upload から game activation、AI bot
admission、match request、worker execution、ranking 更新まで固定する。`echo-count` と `janken` の両方を
WASI game/AI bundle として package し、filesystem と S3-compatible artifact backend に対して同じ immutable
digest contract を検証する。

完了時、この regression が preset endpoint を使わずに end-to-end の運営 flow を証明する。外部 Reversi release
asset の download/upload を staging CI に追加しない。

## Existing References

- `docs/specs/platform-artifact-bundle.md:5-22`: ZIP admission、digest、registry、worker materialization の
  observable contract。
- `operator-ui/tests/operator-ui.ci.spec.js:117-365`: authenticated service-backed browser lane。現状は echo bundle
  を upload し registration/request/ranking を通す。
- `tools/dev/package-builtin-game-bundles.sh:1-78` と
  `tools/dev/operator-ui-backend.sh:43-117`: generated fixture ZIP と filesystem/S3-compatible test topology。
- `operator-ui/tests/operator-ui.spec.js:29-70`: preset-only fixture assertion。これは 0121 で撤去される。
- `internal/platform/service/artifact_admission.go:19-56`、`general.go:205-245`、
  `worker_local.go:54-77`: game admission、activation、exact artifact execution boundary。
- `.github/workflows/operator-ui-browser.yml:88-160`: Postgres/S3-compatible browser CI lane。

## Code Change Map

- `(MODIFY) docs/specs/platform-artifact-bundle.md`: operator general lane の required regression を、game/AI bundle
  upload、activation、bot revision、match completion、ranking と digest snapshot の観測として明確化する。
- `(MODIFY) tools/dev/package-builtin-game-bundles.sh`: echo-count と janken の game bundle、および各 game と
  compatibility のある AI bundle を deterministic に package する。test-only ZIP は repository に commit しない。
- `(MODIFY) tools/dev/operator-ui-backend.sh` と `(MODIFY) operator-ui/playwright.config.js`:
  test fixture locator を game-family parameterized にし、filesystem と S3-compatible lane が同じ bundle set を
  使うようにする。
- `(MODIFY) operator-ui/tests/operator-ui.ci.spec.js`: game family ごとに ZIP upload -> manifest-derived activation ->
  two bot create/revise -> match request -> terminal run -> ranking assertion を通す。request/ranking が preset source
  でないことと game/seat digest identity を API/read model で確認する。
- `(MODIFY) internal/platform/service/*_test.go` と `(MODIFY) artifactbundle/*_test.go` (必要箇所): browser assertion
  だけでは局所化できない ZIP validation、exact descriptor/AI compatibility、worker materialization、filesystem/S3
  parity を focused test で固定する。
- `(MODIFY) .github/workflows/operator-ui-browser.yml` と `(MODIFY) docs/development/operator-ui-local-verification.md`:
  required bundle builder inputs、CI artifact/evidence、local reproduction command を新 regression に合わせる。

## Black-box Contract

- supported built-in test gamesは、preset config を介さず immutable WASI ZIP admission と general operator request
  だけで match/ranking まで到達する。
- game/AI digest、game identity/version/ruleset、seat compatibility は activation/request acceptance 時に固定され、
  worker は同じ digest を materialize して実行する。
- regression は local/CI artifact backend を対象とし、shared staging への test data mutation を行わない。

## Dependencies and Order

1. 0121 の frontend preset removal を merge して、browser suite に preset replacement を残さない。
2. spec と ZIP fixture build matrix を先に固定する。
3. filesystem lane と S3-compatible lane の common assertions を helper 化し、echo-count と janken を同じ
   admission/request/ranking sequence で走らせる。
4. focused service tests と browser CI を green にしてから 0123 の endpoint removal を開始する。

## Verification

- echo-count と janken の双方について、filesystem と S3-compatible store で upload/activate/two bots/request/
  complete/ranking を通す。
- completed run/read model が selected game digest と two AI digest を保持し、temporary materialization directory を
  残さないことを確認する。
- `make test-postgres`、`make test`、`make lint`、`pnpm run verify:ci:postgres`、
  `git diff --check`、`./tools/workflow-lint.sh --mode=pre-push` を通す。

