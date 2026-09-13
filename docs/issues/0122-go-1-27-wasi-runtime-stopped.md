# Go 1.27 WASI runtime-stopped

## Summary

Go 1.27.1 で実行した WASI bundle admission regression が、filesystem と S3-compatible
bundle store の両方で player initialization 中に `runtime-stopped` となる。Go version
directive / preferred toolchain だけを変更した branch で再現し、`go test` の単独実行でも
再現した。

## Context

- observed on: `2026-09-13`
- selected toolchain: `go1.27.1 linux/amd64`
- command: `CI=true CACHE_ROOT=/tmp/ai-arena-go-quality-gates go test ./internal/platform/service -run '^TestArtifactSubmissionUploadToWASIStartAcrossBundleStores$' -count=1`
- failing test: `TestArtifactSubmissionUploadToWASIStartAcrossBundleStores`
- result: filesystem and S3-compatible subtests both return `init failed for p1: runtime-stopped`

`TestArenaRunnerCanceledPath` also failed once in the first full `make test` execution with
`context deadline exceeded`, but its isolated rerun passed; it is not included as the primary
reproducible failure here.

## Impact

- Go 1.27.1 does not currently have passing local evidence for the existing Go-WASM bundle
  admission behavior.
- The Go 1.27 toolchain plan cannot be completed until the adapter or fixture incompatibility is
  diagnosed and its behavior is covered by a focused regression.

## Next Steps

- Capture the WASI process stderr and exit status for the failed initialization before changing
  runtime behavior.
- Determine whether Go 1.27's generated WASI module, the pinned WASI runtime dependency, or
  the local subprocess/session lifecycle causes the early transport close.
- Add a focused reproduction and update the relevant behavioral contract before implementing a
  fix; keep unrelated dependency updates in a separate approved plan.
