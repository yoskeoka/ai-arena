# dungeon の観察導線は compact artifact を既定にしたい

## Summary

現状の dungeon manual/e2e 確認導線では、human や Codex のような AI Agent 実装者が
長い event log や large JSON をそのまま読み込んでしまいがちである。

- `Makefile` の `run-dungeon-local` は `arena-runner` の structured log をそのまま `stdout`
  に流す前提で、quiet/summary 向けの別 target を持たない
  (`Makefile:134-146`)
- `cmd/arena-runner/main.go` は `structured-log.ndjson` を開いて `stdout` と同じ系列の
  structured log を流しつつ、同時に `record.json` / `snapshot.json` /
  `exported-snapshot.json` / `history.json` も常に書き出す
  (`cmd/arena-runner/main.go:304-340`)
- `AGENTS.md` は docs 言語ポリシーのみで、artifact をどの順番で読むか、
  `event_log` を既定で切り捨てるか、といった observation discipline を持っていない
  (`AGENTS.md:1-19`)
- e2e でも `terminal_summary` の存在確認はしているが、human/agent 向けに
  `summary-first` を強制する派生 artifact はまだない
  (`e2e/arena_runner_test.go:123-125`, `e2e/arena_runner_test.go:203-205`)

このままだと、少し長い dungeon match を確認するたびに、人間は読みにくい JSON を追い、
AI Agent は不要なログを大量に context に積みやすい。

## Why It Matters

- dungeon の要素追加に合わせて 1 match あたりの event 数が増えると、local verification の
  読みやすさと token cost が悪化する
- `record.json` / `history.json` は source-of-truth として必要だが、通常の bot 実装や
  gameplay 確認では情報量が過剰
- 「まず compact result を見る、それで足りないときだけ詳細へ降りる」という導線がないと、
  human も AI Agent も毎回フルログへ吸い込まれる

## Proposed Solution

1. `arena-runner` の derived artifact として `result-summary.json` を追加する
   - placement, score breakdown, finished turn, remaining chests, selected public fields,
     artifact path 参照程度までに絞る
2. `structured-log.ndjson` / `stdout` の full stream を読まなくても確認できるよう、
   `--log-output none` または `make run-dungeon-*-quiet` を用意する
3. `AGENTS.md` に dungeon/debug artifact の既定読取順を追加する
   - まず `result-summary.json`
   - 次に `exported-snapshot.json` / `snapshot.json`
   - per-turn log や `record.json.event_log` は因果追跡時だけ
4. 必要なら compact helper (`jq` or small Go helper) を用意し、
   selected fields だけを即座に見られるようにする

## Scope Boundary

- この issue は観察導線と derived artifact の改善を扱う
- source-of-truth artifact の contract 自体は変えない
- dungeon 固有 artifact の compact 化を先に扱い、他 game への横展開は別判断にする
