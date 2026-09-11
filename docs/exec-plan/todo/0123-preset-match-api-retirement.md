# preset-match-api-retirement
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

0122 が general WASI bundle lane の regression を固定した後、preset match を service、HTTP/TypeSpec、fixture、
configuration、developer tooling から完全に撤去する。完了時、operator match creation は registered game scope と
admitted bot を選ぶ `POST /api/v1/match-requests` だけで行い、preset config file や `POST /api/v1/preset-matches`
は存在しない。

既存に保存済みの `source=preset` match read data は削除・書換えず、read/detail/ranking/replay を維持する。この
plan は registry failure contract や external-first lookup priority を変更しない。

## Existing References

- `typespec/namespaces/operator/api.tsp:81-83` と `typespec/namespaces/shared.tsp:252-257`: retiring wire
  operation/model source of truth。
- `internal/platform/service/preset.go:1-128`、`request.go:203-235`、`general.go:27-35, 338-377`:
  preset catalog/materialization と source marker。
- `internal/platform/service/http.go:107-214, 532-556` と `http_test.go`: handler ownership と coverage。
- `cmd/arena-service/main.go:543-567, 610-623`:
  preset config load と output resolution wrapper。
- `cmd/operator-ui-fixture/main.go:75-92`、`config/platform-service/presets*.json`、
  `tools/dev/verify-local-object-storage.go:30-70`: fixture/config/developer tool consumers。
- `operator-ui/src/generated/operator-api/`: TypeSpec regeneration output。手編集しない。

## Code Change Map

- `(MODIFY) docs/specs/platform-service-operator-ui.md` と `(MODIFY) docs/specs/platform-service-general-submission.md`:
  match creation の唯一の operator write lane を registered scope + bot request にし、historical `source=preset`
  read compatibility のみを明記する。
- `(MODIFY) typespec/namespaces/operator/api.tsp` と `(MODIFY) typespec/namespaces/shared.tsp`:
  preset operation/model を削除し、generated OpenAPI/client を canonical generator command で更新する。
- `(DELETE) internal/platform/service/preset.go` と preset-only materialization paths/tests:
  catalog、preset request、`CreatePreset`、`SourcePreset` を削除する。ただし stored string を読める read model は
  backward-compatible に扱う。
- `(MODIFY) internal/platform/service/http.go`、`http_test.go`、`request.go`、`general.go`:
  API construction/route/error test を regular general lane に収束させ、removed route は registered handler を持たない
  404 とする。
- `(MODIFY) cmd/arena-service/main.go` と `main_test.go`: preset config env/flag/load/wrapper を削除し、normal output
  resolution と worker startup を維持する。
- `(DELETE) config/platform-service/presets.example.json`、`presets.operator-ui-file-backed.json`、
  `presets.operator-ui-postgres.json`、`presets.remote-bootstrap.json` と preset bot build/config consumers。
- `(MODIFY) cmd/operator-ui-fixture/main.go`、`tools/dev/operator-ui-backend.sh`、
  `tools/dev/verify-local-object-storage.go`、deployment/local verification docs、browser/service tests:
  preset fixture/config/request を general bundle regression へ置換する。

## Black-box Contract

- `POST /api/v1/preset-matches` と preset request schema は retired され、new preset-origin registration/match は作れない。
- operator は registered scope と eligible admitted bots を用いる normal match-request flow で queue/run/ranking を操作する。
- historical run、request、ranking/replay の読み取りは既存 durable record を失わない。retirement は DB/R2 cleanup を
  意味しない。

## Dependencies and Order

1. 0121 と 0122 の merged PR/latest-head verification を prerequisite とする。
2. behavioral spec と TypeSpec source を先に削除し、generated client/OpenAPI を再生成する。
3. service/CLI/config/fixture/tooling を一括で削除する。generated output を手編集しない。
4. 0122 の e2e regression と historical read compatibility test を通してから API removal を handoff する。

## Verification

- TypeSpec compile/generation が成功し、OpenAPI/generated client に preset operation/model/import が残らない。
- removed endpoint が 404、normal match request が 201/terminal/ranking update、historical preset-source record の
  list/detail/replay read が維持されることを HTTP/service test で確認する。
- `make test-postgres`、`make test`、`make lint`、`pnpm run verify:ci:postgres`、
  `git diff --check`、`./tools/workflow-lint.sh --mode=pre-push` を通す。

