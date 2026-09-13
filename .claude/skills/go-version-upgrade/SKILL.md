---
name: go-version-upgrade
description: Verify Go toolchain upgrades before changing module directives.
---

# Go Version Upgrade

Before updating `go` or `toolchain` directives, confirm the official target
release and that pinned Go tools and the WASI runtime support it. Run every
quality gate with `GOTOOLCHAIN=go<target>` inherited by fixture builds:

- `make test`
- `make test-postgres`
- `make lint`
- `make test-wasm-go`
- `make test-wasm-rust`

Verify Go-WASM AI and game-master artifacts through filesystem and
S3-compatible stores, including each admitted artifact's declared memory
limit. If a pinned tool or runtime is unsupported, do not bypass lint or build
fixtures with an older compiler. Create and complete a focused compatibility
plan first. Create an upgrade PR only after local and CI evidence is green.
