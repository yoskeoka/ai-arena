# Go 1.27 staticcheck compatibility

## Summary

The repository-pinned `honnef.co/go/tools@v0.7.0` staticcheck crashes under
Go 1.27.1, so `make lint` cannot complete after the module toolchain upgrade.

## Context

- observed on: `2026-09-13`
- command: `CI=true CACHE_ROOT=/tmp/ai-arena-go-quality-gates make lint`
- selected toolchain: `go1.27.1 linux/amd64`
- failure: `panic: unexpected expr: *ast.KeyValueExpr` from
  `honnef.co/go/tools/go/ir.(*builder).expr0`

The preceding Go toolchain directives are the only module metadata changes in
this branch. `go vet` and `noctx` run before staticcheck; the aggregate target
stops at the staticcheck panic.

## Impact

- The required lint quality gate has no passing Go 1.27.1 evidence.
- Updating the pinned lint tool is outside the approved Go toolchain plan and
  must be assessed for its own module and lint-result changes.

## Next Steps

- Identify a staticcheck release that supports Go 1.27.1 and verify its output
  separately from the Go toolchain change.
- Update the tool pin, module metadata, and any resulting lint findings in a
  dedicated approved plan.
