# Phase 8 Reversi Public Replay Fixture Contract

## Summary

The merged Phase 8 A public spectator API defines the opaque terminal replay
transport, but it does not provide a versioned cross-repository Reversi fixture
or an executable consumer-compatibility contract. The Reversi reference viewer
cannot demonstrate that its decoder accepts the bytes that `ai-arena` will
publish without a shared fixture boundary.

## Why It Remains Open

The generic public API correctly treats `format`, `version`, and `payload` as
opaque game-owned data. That boundary must remain intact. However, a Reversi
provider needs one stable, public fixture pair containing the terminal replay
response and corresponding final exported-state response, plus documented
provenance and compatibility expectations. It must be generated from the
public game-master transport, never reconstructed from `record.json`,
`history.json`, or another private artifact.

## Follow-up Boundary

- Define the ownership and publication location for a versioned, public
  cross-repository fixture pair usable by provider-hosted viewers.
- Add a contract-level compatibility check that exercises the anonymous public
  endpoints and verifies replay metadata, payload bytes, and final exported
  state stay mutually consistent.
- Keep Reversi-specific payload fields and their semantic decoding in
  `reversi-ai-arena`; do not add them to TypeSpec or derive them in the
  platform.
- Do not change access policy, introduce credentials, expose private artifact
  locators, or make the platform host a provider visualizer as part of this
  follow-up.
