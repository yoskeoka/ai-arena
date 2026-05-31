# Japanese textlint

`ai-arena` は、pull request で変更された `docs/**/*.md` のうち、日本語を含む Markdown に対して
`textlint` を実行する。

## Scope

- trigger: `main` 向け pull request
- target path: `docs/**/*.md`
- non-goal: repository 内の全 Markdown を毎回 lint すること
- non-goal: 実装コード comment や product UI 文言の lint

## Changed Japanese Markdown

対象 file は、少なくとも次を満たす。

1. `git diff --name-only --diff-filter=AMR <base-ref>...HEAD` に含まれる
2. checkout 後も file が存在する
3. path が `docs/` 配下の `.md`
4. content に日本語文字が 1 文字以上含まれる

日本語判定は次の codepoint range を使う。

- Hiragana と fullwidth Katakana (`U+3040`-`U+30FF`)
- Katakana Phonetic Extensions (`U+31F0`-`U+31FF`)
- CJK Unified Ideographs Extension A (`U+3400`-`U+4DBF`)
- CJK Unified Ideographs (`U+4E00`-`U+9FFF`)
- CJK Compatibility Ideographs (`U+F900`-`U+FAFF`)
- Halfwidth Katakana (`U+FF65`-`U+FF9F`)

## Helper Contract

changed file 判定の source of truth は `tools/list-changed-japanese-markdown.sh` とする。

```sh
tools/list-changed-japanese-markdown.sh [base-ref]
```

この helper は:

- `base-ref` 未指定時に `origin/main` を使う
- `${base_ref}...HEAD` を比較する
- 条件を満たす path を 1 行 1 file で出力する
- deleted file を無視する

## textlint Runtime

- `package.json` は pinned `pnpm` version を宣言する
- `textlint` と `textlint-rule-preset-ai-writing` を devDependency として固定する
- `.textlintrc.json` は `preset-ai-writing` を有効化する
- workflow と local command は `--rulesdir ./tools/textlint-rules` を使う
- repo-local dictionary rule は `config/textlint/terms.jsonl` を読む
- `pnpm run textlint` は `git ls-files` で tracked `docs/**/*.md` を集める
- local command は次を提供する
  - `pnpm run textlint`: tracked `docs/**/*.md` 全体を対象にする
  - `pnpm run textlint:file -- <target.md>...`: 指定 file を対象にする

## Dictionary Contract

`config/textlint/terms.jsonl` は 1 行 1 JSON object の JSONL とする。

各 line は次を持つ。

- `pattern`: JavaScript regular expression source
- `replacement`: preferred replacement text

example:

```json
{"pattern":"\\btaxonomy\\b","replacement":"分類"}
```

- `pattern` は JSON string 内に入る JavaScript regular-expression source とする
- `\\b` は word boundary を表し、`\\btaxonomy\\b` は standalone な `taxonomy` だけに一致する
- よく使う pattern 例:
  - `\\bword\\b`: standalone な English word
  - `^text$`: 行全体の完全一致
  - `foo.*bar`: 同一行の `foo` から次の `bar` まで
  - `[0-9]`: 1 digit
  - `[A-Za-z0-9_-]+`: ASCII letter / digit / `_` / `-` の 1 回以上
  - `\\.` `\\(` `\\)` `\\\\`: literal `.`, `(`, `)`, `\`
- shell glob ではなく JavaScript regex syntax を使う
- malformed JSONL は failure とする
- invalid regular expression は failure とする
- custom rule は global match 以外の flag を自動追加しない

## Workflow Contract

`.github/workflows/japanese-textlint.yml` は次を満たす。

- `main` 向け pull request で起動する
- changed `docs/**/*.md` と、この quality gate 自体を構成する workflow / helper / config / runtime file の更新で起動する
- `pinact` で管理された pinned `actions/checkout` と `actions/setup-node` を使う
- diff 前に Node runtime を用意し、PR base branch を fetch する
- changed Japanese Markdown が 0 件なら success で終了する
- changed Japanese Markdown が 0 件でも、workflow / helper / config / runtime file 自体が changed なら smoke check 用に repo-local 日本語 doc 1 件へ `textlint` を流す
- 対象 file が 1 件以上あるときだけ `pnpm install --frozen-lockfile` を実行する
- 1 回の `pnpm exec textlint --rulesdir ./tools/textlint-rules --format json <files...>` で全対象 file を lint する
- finding を GitHub Actions warning annotation と step summary に変換する
- stable marker 付き PR comment を upsert し、rerun や push ごとに comment を増やさない
- changed Japanese Markdown が 0 件の rerun では、既存 marker comment があれば削除して stale findings を残さない
- prose finding では job を fail させない
- dependency install、dictionary parse、tool output parse は failure とする
- `GITHUB_TOKEN` は dependency install と `textlint` subprocess へ渡さず、PR comment delete/upsert 用 API request にだけ渡す
- PR comment delete/upsert の GitHub API timeout/error は warning として扱い、job は継続する

## Reporting Contract

warning annotation には次を含める。

- file path
- line
- column
- rule id
- lint message

PR comment と step summary には次を含める。

- checked file count
- finding count
- finding table

## Local Verification

helper 単体確認:

```sh
tools/list-changed-japanese-markdown.sh origin/main
```

CI 相当の focused check:

```sh
pnpm install --frozen-lockfile
mapfile -t files < <(tools/list-changed-japanese-markdown.sh origin/main)
[ "${#files[@]}" -eq 0 ] || pnpm run textlint:file -- "${files[@]}"
```
