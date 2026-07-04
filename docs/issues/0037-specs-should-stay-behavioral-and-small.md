# specs-should-stay-behavioral-and-small

## Summary

`ai-arena` の一部 `docs/specs/` は、observable behavior の説明に加えて、
wire-level の request/response field inventory や API family の詳細まで抱え込みすぎている。
その結果、関連箇所を読むために markdown 全体を広く追う必要があり、
symbol ベースで必要箇所へすばやく飛べる code よりも token / review cost が高くなっている。

## Why This Matters

- spec が大きすぎると、実装確認のたびに全文走査が必要になりやすい
- 同じ contract が spec / code / generated artifact に重複すると、更新漏れと読解コストが増える
- behavioral spec まで wire-level で埋めると、`docs/specs/` が code の mirror になってしまう

## Observed Pattern

- route / payload / response の詳細が Markdown に長く並び、実装確認時に読了コストが高い
- 実際には source of truth にしたい内容が TypeSpec や generated artifact 側へ寄せられるべきなのに、Markdown に残り続けている
- spec が大きいほど、関連 file を広く読む必要が生まれ、レビューの局所性が落ちる

## Desired Direction

- `docs/specs/` は observable behavior、責務境界、topology、policy に絞る
- wire-level contract は repo-owned schema source と generated artifact に寄せる
- 仕様が長文化したら、まず「Markdown で持つべき内容か」を再判定する

## Follow-up

必要なら後続 plan で、他の大きい spec についても同じ方針でスリム化する。
