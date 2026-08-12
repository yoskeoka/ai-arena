# Reversi release artifact verification

Phase 7 の external game acceptance は、Reversi の公開 release asset を bundle-only runner input として実行する。download 前に release の `SHA256SUMS` と固定 digest を確認し、repo-local checkout や legacy AI entrypoint を代用しない。

最初の検証対象は `yoskeoka/reversi-ai-arena` の `v0.1.0` release とする。

- `reversi-game-v0.1.0.arena.zip`: `sha256:98bd46609016dc763bcbfff747c6705c7f1608a86d164d77b38db208d0d1c0df`
- `reversi-rust-reference-ai-v0.1.0.arena.zip`: `sha256:15ec6133f92002f68f291b39df69ac9beeb177debc93dd1d9c725c77db042326`

```sh
make verify-reversi-release-artifacts
```

この command は release ZIP と checksum manifest を temporary directory に取得し、checksum を検証したうえで game bundle と同じ AI bundle を両 seat に渡して一試合を完走させる。network を必要とするため、通常の hermetic Go test lane には含めない。service の upload → registry → queue → materialize → WASI start は `internal/platform/service` の hermetic integration test が filesystem と S3-compatible HTTP store の両方で担保する。
