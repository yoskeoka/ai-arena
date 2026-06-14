# local-playwright-and-worktree-dependency-bootstrap

## Summary

`operator-ui` の local / CI-like browser verify は、この worktree では code change ではなく環境 bootstrap の欠如で止まった。

- `pnpm build` は通る
- `pnpm verify:ci:file-backed` はデフォルト `chrome` channel で `/opt/google/chrome/chrome` 不在により失敗する
- bundled Chromium を `/tmp` に install して `OPERATOR_UI_BROWSER_CHANNEL=chromium` へ切り替えても、この環境では `chrome_crashpad_handler: --database is required` で launch failure になる
- root / `operator-ui` ともに fresh `ww create` worktree では `node_modules` 不在のため、`pnpm textlint:file` や `pnpm build` の前に毎回 install が必要になる

関連箇所:

- `operator-ui/playwright.config.js`
- `operator-ui/package.json`
- `tools/run-textlint.sh`

## Proposed Solution

別 PR で、local 開発 bootstrap を repo-owned に固定する。

候補:

- `ww create` 後の hook で root と `operator-ui` の依存を worktree 側へ自動展開する
- Playwright browser path を worktree 跨ぎで共有する既定値に寄せる
- local verify が system Chrome 前提なら、その install 手順を repo-owned runbook に固定する
- あるいは `ci` / `local` の browser selection を見直し、worktree ごとに追加 install を要求しない既定へ寄せる

## Priority

中。今回の `0079` 実装そのものは backend / spec / frontend build / Go test で検証できたが、fresh worktree から browser verify へ入るたびに環境準備をやり直す必要があり、継続的な operator UI 開発と agent 実行効率を落とす。
