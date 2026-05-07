# review-task template fallback skill gap

## Summary

`review-task` skill の PR template fallback 説明が、workspace 配下 child repo の実運用を十分に表していない。
今回 `ai-arena` で `.github/PULL_REQUEST_TEMPLATE.md` が見つからなかった際、本来は
`vibe-coding-workspace/.github/PULL_REQUEST_TEMPLATE.md` を見に行くべきだったが、skill の記述だけでは
その判断が弱く、repo 内 template 不在をそのまま手書き PR body 作成へ倒しやすい。

影響箇所:

- `skills/review-task/SKILL.md`
- workspace 配下 child repo の PR 作成フロー全般

## Proposed Solution

- `review-task` skill の template fallback 順を、child repo local template の次に workspace root template を明示する形へ更新する
- fallback 対象 path を具体例付きで書く
  - child repo: `<child>/.github/PULL_REQUEST_TEMPLATE.md`
  - workspace root: `vibe-coding-workspace/.github/PULL_REQUEST_TEMPLATE.md`
  - vendored workflow template / workflow repo template はその後段に回す
- 必要なら `review-task` helper script 側にも同じ順序を実装し、skill 文言と実装を一致させる

## Priority

中。すぐに product code を壊す問題ではないが、workspace 配下 repo で PR body 品質と workflow 一貫性を落としやすい。
