# Go の exported doc comment を lint で強制していない

## Summary

`ai-arena` の `make lint` は現在 `goimports` / `go vet` / `noctx` / `staticcheck` / `gosec`
のみを実行しており、exported package / type / const / func の doc comment を強制していない。

そのため、`games/dungeon/*` のように public API が増えても、`go doc` / `pkgsite` 向けの基本的な
説明コメントが抜けたまま CI を通せてしまう。

## Why It Matters

- exported API の意図がコードだけでは読み取りづらくなる
- package comment がない package が増えても CI では検出できない
- Go の公開 API として一般的な保守水準を repo 全体で揃えにくい

## Proposed Solution

次のどちらかを採用し、少なくとも exported identifier と package comment を機械的に検出できるようにする。

1. `revive` などの comment/style rule を `make lint` に追加する
2. `golangci-lint` を導入せずに済ませたいなら、既存 toolchain に合わせて軽量な comment checker を追加する

判断時には次を明確にする。

- package comment まで必須にするか
- exported const / var / type / func / method のどこまで対象にするか
- 既存未整備コードへの段階導入にするか、一括是正してから有効化するか

## Scope Boundary

- 今回の PR では `dungeon` 関連へ必要な comment を追加する
- repo 全体の lint policy 変更は別 PR で扱う
