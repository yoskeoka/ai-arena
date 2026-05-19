# Specifications

`docs/specs/` は、`ai-arena` が提供する product / platform / runtime / game contract の正本を置く。

ここに置くべきもの:

- platform / runner / runtime / registry / game master の公開契約
- game 非依存 DTO、lifecycle、metadata、artifact、transport 契約
- repo 内に同梱する game の payload / ruleset / scoring 契約

ここに置かないもの:

- contributor workflow
- repo-local quality gate
- CI maintenance
- bot / hook / linter 運用
- consumer repo の開発手順、asset 配置、ownership、tagged import 採用手順

development harness 文書は `docs/development/` に置く。

## Spec Writing Checklist

spec / plan review では、少なくとも次を確認する。

- その文は current implementation の package / type / method 名ではなく、責務・境界・入力・出力・観測可能なふるまいを主語にできているか
- symbol 名を残している場合、それは単なる current implementation detail ではなく、reader が contract を共有するための安定した抽象概念か
- transport method 名や schema field 名のように concrete な名前を書く場合も、「何の contract を固定しているのか」が同じ段落で読めるか
- development harness や repo-local workflow の説明が `docs/specs/` に混入していないか
- product spec の本文が、consumer repo の ownership や開発運用へ踏み込む guide 文書になっていないか
