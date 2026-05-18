# platform spec should describe provided contracts, not consumer guidance

## Summary

`docs/specs/platform.md` と `docs/specs/game-master.md` の書き方に、
「ai-arena が何を提供し、どこまでを仕様として固定するか」と
「外部 repo で game / AI player をどう開発・配置・検証するか」の説明が混在している。

現状の問題は次の 2 点。

1. ai-arena の spec が、platform / runner / SDK の外形仕様ではなく、
   external consumer 向けの利用ガイドや ownership 議論まで抱え込んでいる
2. 非責務領域を説明するときに、
   「consumer repo 側で持ってよい」「game 開発側が決める」のような
   repo 間の運用判断へ踏み込み、platform spec の主語がぶれている

今回の会話で user が明示した意図は、
ai-arena spec は「ai-arena platform / runner / SDK は何を提供し、どの共通契約を固定するか」
を外形レベルで簡潔に記述すべきであり、
外部で game を作る人向けの開発ガイドや repo ownership 説明まで含めるべきではない、というもの。

関連箇所:

- `docs/specs/platform.md`
- `docs/specs/game-master.md`

## Proposed Solution

次の方針で spec を整理する。

1. `platform.md` は platform / runner / SDK が提供する contract と責務だけを書く
2. `game-master.md` は ai-arena が期待する game master contract だけを書く
3. external repo の開発方法、verification asset 配置、ownership、運用フローは
   必要なら別の guide / migration note / execution plan に寄せる
4. spec 本文では「consumer repo 側で持ってよい」のような許可文型を使わず、
   ai-arena が規定する責務と、規定しない境界だけを書く

## Priority

中。

このままでも直ちに実装は壊れないが、platform spec の責務境界が読みにくくなり、
今後 external game repo 前提の wording を増やすほど記述が散らかる。
Phase 3/5 の platform contract を durable に保つには、早めに整理した方がよい。
