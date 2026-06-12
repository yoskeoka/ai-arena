# online-release-staging-verify-playwright-install-hang

## Summary

`online-release-staging-verify` が `Install Playwright browser`
(`pnpm exec playwright install --with-deps chromium`) で長時間 `in_progress` のまま止まり、
同じ commit SHA に対する後続 verify run も concurrency group によって `pending` のまま詰まる事象がある。

2026-06-12 の観測では、このハング自体は staging deploy skip の直接原因ではなかったが、
staging verification の自動再試行性と release 状態の見通しを悪化させる独立問題として残っている。

## Context

- observed on: `2026-06-12`
- target SHA:
  `08c878d3a10e1cf9d19e8a4eca64736c12a172a3`
- hanging workflow run:
  `https://github.com/yoskeoka/ai-arena/actions/runs/27434501235`
- hanging job:
  `https://github.com/yoskeoka/ai-arena/actions/runs/27434501235/job/81092910991`
- blocked follow-up workflow run:
  `https://github.com/yoskeoka/ai-arena/actions/runs/27434537414`
- blocked follow-up job:
  `https://github.com/yoskeoka/ai-arena/actions/runs/27434537414/job/81093035863`

## Observed Behavior

- `verify-staging` job は setup / checkout / `pnpm install` までは完了している
- `Install Playwright browser` step が開始後に進捗を出さず、そのまま job 全体が `in_progress` に留まる
- `online-release-staging-verify-08c878d3a10e1cf9d19e8a4eca64736c12a172a3` の concurrency group が埋まり、
  同じ SHA の後続 verify run は `pending` のまま待たされる

## Why This Is Separate From The Release-Gate Fix

- staging deploy skip の直接原因は `online-release-staging.yml` の gate / duplicate 判定であり、
  Playwright verify hang ではない
- 一方で verify hang は、staging deploy を手動 rerun した後の verification 再試行や
  artifact 採取を詰まらせるため、独立した workflow reliability issue として追う価値がある

## Suspected Cause

- `playwright install --with-deps chromium` 自体の download / package install が止まっている
- あるいは install step の内部 subprocess は終わっており、
  後続 cleanup / child process wait が戻ってこない
- 少なくとも GitHub Actions UI 上の step 表示だけでは、
  browser download・APT dependency install・post-install wait のどこで止まっているかを切り分けられていない

## Proposed Investigation

- `Install Playwright browser` の前後に timing / process list / disk usage を出して、
  browser download 中なのか child process hang なのかを切り分ける
- `playwright install --with-deps chromium` を
  `playwright install chromium` と host dependency install に分離できるか検討する
- hang 発生時に workflow cancellation 後の rerun で再現するか、runner image 依存かを確認する
- concurrency group を維持したままでも stale run で詰まり続けないよう、
  timeout や operator-facing recovery 手順を runbook に追加するか検討する
