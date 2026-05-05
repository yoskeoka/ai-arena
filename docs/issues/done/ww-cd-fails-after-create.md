# `ww cd` が `ww create` 直後に失敗する

## 概要

Phase 1 タスク `001-game-concept` の実行中、想定されている以下のセットアップ手順:

```sh
ww create --repo ai-arena docs/001-game-concept
cd "$(ww cd --repo ai-arena docs/001-game-concept)"
```

が、ドキュメントどおりには動作しなかった。

## 実際の挙動

- `ww create --repo ai-arena docs/001-game-concept` は成功した
- 作成された worktree path:
  `.../.worktrees/ai-arena@docs-001-game-concept`
- 直後の `ww cd --repo ai-arena docs/001-game-concept` は以下で失敗した:
  `no worktree found for branch "docs/001-game-concept"`

## 期待する挙動

`ww create` が成功した直後であれば、同じ repo と branch を指定した `ww cd` で、新しく作られた worktree path を解決できるべき。

## 影響

- workflow 手順をそのまま実行できなかった
- 作成済み path を直接使うことで作業は継続できた
- code/spec 作業自体は止まらなかったが、標準的な worktree UX は壊れていた

## フォローアップ

`ww cd` が `ww create` と異なる branch key format を期待しているのか、あるいは `docs/*` branch に対する worktree 登録が不完全なのかを調査する。
