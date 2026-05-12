# Specifications

Place detailed specifications here.
Start with `system-overview.md` or similar high-level docs.

- `go-quality-gates.md`: Go module の `make test` / `make fmt` / `make lint` 契約
- `github-actions-pinning.md`: GitHub Actions の `uses:` 参照を `pinact` で管理する運用契約
- `game-master.md`: platform と game master の標準論理 API / transport / lifecycle 契約

## Spec Writing Checklist

spec / plan review では、少なくとも次を確認する。

- その文は current implementation の package / type / method 名ではなく、責務・境界・入力・出力・観測可能なふるまいを主語にできているか
- symbol 名を残している場合、それは単なる current implementation detail ではなく、reader が contract を共有するための安定した抽象概念か
- transport method 名や schema field 名のように concrete な名前を書く場合も、「何の contract を固定しているのか」が同じ段落で読めるか
