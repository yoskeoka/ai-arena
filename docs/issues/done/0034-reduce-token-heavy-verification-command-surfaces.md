# Summary

このセッションでは、実装そのものよりも verification / PR follow-up まわりの command output が
context と token を大きく消費した。
主な無駄は次の 2 系統に分かれる。

1. 成功時は pass/fail summary だけで十分なのに、
   tool / script / Make target が trace level に近い詳細ログを常時出している
2. human / AI agent の実行入口なのに、
   長い env var 列を毎回そのまま指定しないと mode 切替できない command surface が残っている

今回の具体例:

- [Makefile](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/ai-arena@feat-operator-ui-auth-playwright-github-signup-lane/Makefile:60)
  の `make test`
  - `go test ./...` の成功出力自体はまだ許容範囲だが、session 中には複数回再実行されるため累積 token が大きい
- [Makefile](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/ai-arena@feat-operator-ui-auth-playwright-github-signup-lane/Makefile:144)
  の `make lint`
  - `go vet` / `staticcheck` / `gosec` / `revive` の各 tool が成功時にも大量の進捗ログを出しうる
  - 特に `gosec` の package/file 列挙は成功時 summary だけでよい
- [operator-ui/package.json](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/ai-arena@feat-operator-ui-auth-playwright-github-signup-lane/operator-ui/package.json:12)
  の `verify:local:auth`
  - `OPERATOR_UI_TEST_SCENARIO=real-local OPERATOR_UI_BACKEND_MODE=auth-mock ...` のような長い env var 列を毎回 surface に出している
  - 実行パターンは限られているのに、mode 切替責務が script / make 内部へ寄っていない
- [tools/dev/run-operator-ui-playwright.sh](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/ai-arena@feat-operator-ui-auth-playwright-github-signup-lane/tools/dev/run-operator-ui-playwright.sh:1)
  と
  [tools/dev/operator-ui-backend.sh](/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/.worktrees/ai-arena@feat-operator-ui-auth-playwright-github-signup-lane/tools/dev/operator-ui-backend.sh:1)
  - mode は内部で吸収できるのに、実行入口側で env var を大量に組み立てている
- `skills/review-task/scripts/gh-pr-followup poll`
  - JSON は compact 化されているが、成功時でも checks 全件・review body・timeline event を返すため、
    landing loop で繰り返すと累積 token が大きい

# Proposed Solution

削減案の検討対象として、少なくとも次を list up する。

1. verification command surface を短縮する
   - `make verify-auth-local`
   - `make verify-operator-ui-local`
   - `make verify-operator-ui-real`
   のように、代表的 lane は short command へ固定する
   - 実行側が渡すのは `mode=auth` のような少数パラメータだけに寄せる
   - env var の既定値と mode 分岐は Makefile / shell script 内部へ閉じ込める

2. 成功時 output を最小化する
   - success 時は `OK`, `lint passed`, `2 tests passed` のような summary 優先にする
   - `gosec`, `Playwright`, bootstrap helper, backend wrapper の verbose / trace は default off にする
   - 詳細ログは `DEBUG=1` や `VERBOSE=1` のような明示 opt-in に限定する

3. error 時だけ詳細を出す設計へ寄せる
   - command wrapper が標準では quiet mode で走る
   - failure 時だけ対象 step と stderr の要点を出す
   - full log は artifact path や temp file へ逃がし、必要時だけ参照する

4. PR follow-up helper の output tier を分ける
   - default: head SHA / required checks summary / review有無 だけ
   - verbose: review body / inline comments / timeline detail
   - advisory review待ちでなければ compact summary だけ返す

5. agent-facing local verification docs も short command を正本に寄せる
   - `docs/development/operator-ui-local-verification.md`
   - `docs/development/go-quality-gates.md`
   では、human / AI agent が毎回長い env var を覚えなくてよい command を正本にする

# Priority

中〜高。

- data integrity や user-facing bug ではないが、
  AI-centered workflow では context/token 消費そのものが開発効率を落とす
- 実装 1 回ごとの無駄は小さく見えても、
  `make lint` / `make test` / browser verify / PR follow-up の繰り返しで累積コストが大きい
- human にも AI agent にも「短い command」「成功時は静か、失敗時だけ詳しく」が有利なので、
  repo-owned verification surface の改善対象として優先度は十分ある
