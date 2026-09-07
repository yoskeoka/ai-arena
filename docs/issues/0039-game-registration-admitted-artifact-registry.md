# admitted artifact の game registration が built-in registry に阻まれる

## Summary

staging で operator が Reversi game bundle を upload して admission に成功した後、
返された `artifact_id` を使って `/operator/games` から registration を作成すると、
次のエラーで失敗した。

```text
service: bad request: registry: unsupported game "reversi"
```

## Observed Evidence

- 環境: `https://staging.ai-arena.pages.dev/operator/games`
- registration ID: `reversi-stg-20260907-1026-yoske`
- upload response: HTTP `201`
- admitted artifact: `sha256:98bd46609016dc763bcbfff747c6705c7f1608a86d164d77b38db208d0d1c0df`
- admitted game: `reversi@1.0.0`
- admitted ruleset: `standard`
- build mode: `wasm-wasi`
- registration response: HTTP `400`

したがって bundle の archive、manifest、digest、game version、ruleset admission は成功しており、
失敗は admission 後の registration validation に限定される。

## Suspected Cause

arena-service は upload 時に `registry.NewWASIOverlay` から作った writable registry へ game descriptor を
登録する。一方、general submission service は registry を `nil` で生成されるため `registry.Default()` を
使い、built-in の echo / janken descriptor だけを参照する。

さらに artifact-backed registration でも、指定された digest の exact lookup ではなく
`game_id + game_version` の通常 lookup を先に行う。そのため admitted artifact に descriptor が存在しても、
`reversi` が built-in registry にないことを理由に拒否される。

## Expected Behavior

- admitted game artifact を指定した registration は、built-in game の事前登録や game ID の whitelist を
  必須条件にしてはならない。
- selected artifact の manifest と descriptor を source of truth とし、game identity、version、ruleset、
  build metadata、artifact digest の整合性を検証する。
- artifact が admitted でなく、kind・digest・version・ruleset の整合性検証に失敗した場合だけ、
  registration を拒否する。

## Scope

この issue は、admission 済み artifact からの game registration validation と service wiring を対象とする。
Games page の file-picker upload UX は open な `games-inline-bundle-upload` 計画の対象であり、ここでは扱わない。
artifact admission 後に activation されなかった bundle の retention / cleanup も対象外とする。

## Follow-up

`docs/exec-plan/todo/0110-reversi-game-registration-admitted-artifact.md` で、
共有 overlay registry と exact admitted-artifact lookup の修正、および回帰テストを計画する。
