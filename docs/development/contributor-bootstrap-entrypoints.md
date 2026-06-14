# contributor bootstrap entrypoints

fresh worktree で setup runbook を先に辿らなくても、
repo-owned な canonical command をそのまま実行すれば
必要最小限の local bootstrap が missing 時だけ走る。

正本の contract は `docs/specs/contributor-bootstrap-entrypoints.md` とする。
この文書は contributor / agent 向けの短い運用入口だけを持つ。

## Canonical commands

root textlint:

```sh
pnpm run textlint
pnpm run textlint:file -- docs/specs/platform.md
```

`operator-ui` fixture local verify:

```sh
cd operator-ui
pnpm run verify:local
```

`operator-ui` real local verify:

```sh
cd operator-ui
pnpm run verify:local:real
```

## What auto-bootstrap does

wrapper は command 実行直前にだけ次を確認する。

- root textlint:
  root `node_modules` が missing なら `pnpm install --frozen-lockfile`
- `operator-ui` verify:
  `operator-ui/node_modules` が missing なら `pnpm install --frozen-lockfile`
- `operator-ui` verify:
  Playwright Chromium executable が missing なら browser install helper

dependencies と browser が既に揃っている通常時は再install しない。

## What auto-bootstrap does not solve

次は wrapper の外側に残る。

- `pnpm` 自体の未導入
- Playwright host library 不足
- package registry へ到達できない環境
- browser install 後も残る Playwright launch failure

この種の failure は、各 command の stderr と次の runbook を見て切り分ける。

- `docs/development/japanese-textlint.md`
- `docs/development/operator-ui-local-verification.md`
