# ai-arena Japanese textlint rollout
**Execution**: Use `/execute-task` to implement this plan.

## Objective

workspace で確立した日本語 `textlint` CI を `ai-arena` にロールアウトし、`docs/*` の plan / issues / specs / ADR / development docs を含む日本語 Markdown 更新に対して PR comment review を出せるようにする。

## Depends on

- workspace plan `0042-japanese-textlint-01-workspace.md` が実装され、辞書形式・custom rule contract・PR comment shape が固まっていること

## Scope and outcome

- `ai-arena` repo で変更された `docs/**/*.md` を対象に、日本語を含む file のみ `textlint` を実行する
- workspace で試した `textlint-rule-preset-ai-writing` と JSONL 辞書 rule を `ai-arena` に repo-local 導入する
- 初期辞書として `{"pattern":"\\btaxonomy\\b","replacement":"分類"}` を同様に入れる
- `ai-arena/README.md` に textlint セクションを追加し、辞書メンテ方法と実行方法を書く

## Non-goals

- `ai-arena` 実装コードや product UI 文言への適用
- child repo 全体への一括展開
- workspace と `ai-arena` の辞書を単一共有 package にまとめること

## Design notes

- `ai-arena` の ADR では `docs/*` の内部ドキュメントを当面日本語で統一しているため、対象 path は `docs/**/*.md` を基本にする
- workspace と同じ UX を優先し、annotation + stable PR comment upsert を採用する
- 導入初期はノイズ抑制より coverage を優先し、plan / issue / done/ 配下も path では除外しない
- workspace 実装で helper や custom rule に改善点が出た場合は、その確定 shape を `ai-arena` 側へ持ち込む

## Code changes

- `ai-arena/.github/workflows/` に日本語 textlint CI workflow を追加する
- `ai-arena/tools/` に changed Japanese Markdown helper を追加するか、repo-local 実装として同等機能を置く
- `ai-arena/package.json` と `.textlintrc` 系設定を追加または更新する
- `ai-arena/tools/textlint-rules/` に repo-local custom rule を追加する
- `ai-arena/config/textlint/terms.jsonl` を追加する
- `ai-arena/README.md` に運用説明を追加する

## Spec changes

- `ai-arena/docs/specs/` に日本語 textlint CI spec を追加する
- `ai-arena/docs/specs/README.md` に新 spec を列挙する
- spec では以下を明記する
  - `docs/**/*.md` を対象にした changed Markdown 判定
  - 日本語 file 判定
  - `textlint-rule-preset-ai-writing` と JSONL 辞書 rule の採用
  - annotation / PR comment 契約
  - local verification 手順

## Verification

- `ai-arena/docs/specs/` と `docs/exec-plan/` の日本語 sample で `textlint` が動くことを確認する
- `taxonomy -> 分類` の辞書 rule が PR comment と summary に反映されることを確認する
- workflow YAML の構文、pin 更新、repo-local helper の実行を確認する

## Sub-tasks

- [ ] [parallel] workspace 実装から流用する helper / custom rule / comment logic を棚卸しする
- [ ] [parallel] `ai-arena` の対象 path と local verification コマンドを確定する
- [ ] [depends on: workspace implementation shape, rollout inventory] `ai-arena` に workflow / helper / config / dictionary を実装する
- [ ] [depends on: implementation] `ai-arena/README.md` と spec を更新し、辞書メンテ方法を明記する
- [ ] [depends on: implementation, docs] ローカル verification を実行し、PR comment 出力を確認する

## Success criteria

- `ai-arena` の `docs/*` 日本語更新が PR 上で機械的に review される
- `plan` / `issue` の文言癖が後続 spec に流入する前に visible になる
- 辞書追加手順が `README.md` から追える
