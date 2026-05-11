# spec は実装コード名ではなく責務とふるまいを記述する

## Summary

`platform-game-registry.md` の修正中に、spec の説明を補強するために
`registry.Registry.Lookup*` のような実装コード名へ寄せた書き方を許容しかけた。

今回は該当箇所を責務ベースの表現へ修正したが、同じ drift は registry 以外の spec でも起こり得る。
spec が concrete な関数名・メソッド名・型名に引きずられると、責務や入出力の契約が変わっていなくても、
小さな refactor のたびに spec 更新が必要になる。

この issue では、`ai-arena` の spec 群について以下の方針へ明示的に軌道修正する。

- spec はまず責務・境界・入力・出力・観測可能なふるまいを書く
- 実装コード名は、安定した抽象概念として spec の主語になっている場合にだけ限定的に使う
- 現在の package / type / method 構成を、そのまま spec の構造へ写さない

## Impact

- 現状の spec では、文脈によっては実装コードの現在形に寄りすぎる記述が混ざる余地がある。
- 将来の refactor で behavior が不変でも spec 修正ノイズが発生しやすい。
- spec-review 時に「contract の変更」と「実装の書き換え」を切り分けにくくなる。

## Follow-up

- `docs/specs/platform-game-registry.md` を起点に、platform/runner/registry 周辺 spec の wording を見直し、実装コード名に依存する記述を洗い出す。
- 「抽象概念として許容する型/interface名」と「単なる current implementation detail」の線引きを、`docs/lessons.md` と必要なら design decision に反映する。
- 新規 spec/plan review で、concrete symbol 名が契約上必須かどうかを確認する lightweight checklist を持つ。
