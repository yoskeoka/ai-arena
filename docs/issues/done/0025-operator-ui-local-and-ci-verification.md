# operator-ui-local-and-ci-verification

## Summary

`operator-ui/` は local で手動確認できる最小導線を持つようになったが、
実装者の AI Agent が local backend/frontend を立ち上げて表示確認する手順や、
CI で回せる integration/e2e verification が未整備である。

今回のような operator UI は、API contract だけでなく実画面の state visibility、
polling、preset queue、completed detail 表示が正しく組み合わさることを確かめたい。
しかし現状は human manual check へ寄っており、AI Agent の自己検証や CI の継続検証が薄い。

## Context

- current UI implementation:
  `operator-ui/src/App.tsx`
- local runbook:
  `README.md`
- current Pages build contract:
  `docs/development/platform-service-online-deploy.md`

## Impact

- local で backend は動いていても、UI が想定どおり表示されているかを AI Agent が自動確認できない
- polling interval、completed detail、artifact link rendering の regressions が CI で捕まりにくい
- local verification の手順が human knowledge 依存になりやすい

## Proposed Solution

- local verification lane を 2 段に分けて整備する
  - AI Agent 向け:
    local backend/frontend を起動し、想定 UI が表示されているかを headless browser で確認する runbook
  - CI 向け:
    `operator-ui` の minimal e2e / integration test を GitHub Actions で回す lane
- browser automation は 1 つに絞る
  - 候補:
    Playwright CLI、Playwright test、または同等の headless browser harness
- first coverage 候補:
  - health backend 起動確認
  - active/completed panel の描画
  - preset queue action
  - completed detail の result-summary 表示
  - delegated artifact access entry の rendering
- local lane では、AI Agent が subagent に backend/frontend 起動と browser verification を任せられる形まで整える
- CI lane では、可能な限り Postgres-backed local backend と組み合わせた
  deploy-shaped verification を優先する

## Priority

中〜高。
operator UI は今後 view や操作が増える前提であり、manual check 依存のままだと
回帰検知コストが急速に上がる。
