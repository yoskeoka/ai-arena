# operator-ui-browser-playwright-hang

## Summary

`operator-ui-browser` workflow run
`https://github.com/yoskeoka/ai-arena/actions/runs/28283819740`
では、`operator-ui-browser-file-backed` と
`operator-ui-browser-postgres` の両 job が
Playwright test step 開始直後に長時間無音のままぶら下がり、約 6 時間後に cancel された。

`PR255` で対処した
`online-release-staging-verify` の host-runner install step hang と似て見えるが、
今回の job log では workflow 上の独立 install step ではなく、
`verify:ci:*` から呼ばれた helper 内の browser bootstrap が最後の出力になっている。
したがって first suspect は `PR276` で導入した
repo-owned Playwright helper / bootstrap 変更であり、
`operator-ui-browser.yml` 全体を直ちに staging verify と同形の Docker 化へ寄せる前に、
CI browser lane 固有の cause を切り分ける必要がある。

## Context

- observed on: `2026-06-27`
- target workflow run:
  `https://github.com/yoskeoka/ai-arena/actions/runs/28283819740`
- related PR:
  `https://github.com/yoskeoka/ai-arena/pull/276`
- related earlier fix:
  `https://github.com/yoskeoka/ai-arena/pull/255`

## Observed Behavior

- `operator-ui-browser-file-backed` job
  - `pnpm run verify:ci:file-backed` までは開始している
  - `../tools/dev/run-operator-ui-playwright.sh` が
    `bootstrapping Playwright Chromium for operator-ui` を出した後、
    Chromium download progress が 100% まで進んだところで止まる
  - その後 Playwright test 本体の出力、backend/frontend 起動ログ、
    healthcheck success、artifact upload には進まず cancel された
- `operator-ui-browser-postgres` job
  - `Start SeaweedFS harness` までは完了している
  - `pnpm run verify:ci:postgres` の helper が同様に
    browser bootstrap 出力を最後に止まる
  - Postgres service 自体は healthy だが、
    Playwright test 本体や app startup log へは進まない

## Why This Matters

- current `operator-ui-browser.yml` は Phase 7 operator surface の dedicated CI browser lane であり、
  polling / queue / completed detail / artifact access の regression 検知を担っている
- file-backed と Postgres durable lane の両方が同時に止まるため、
  browser verification 全体が失効している
- local canonical lane と remote staging verify lane は repository 内で異なる runtime contract を持つため、
  one-size-fits-all の Docker 化で済むとは限らない

## Suspected Cause

- `PR276` で `verify:ci:file-backed` / `verify:ci:postgres` が
  direct `playwright test` から
  `tools/dev/run-operator-ui-playwright.sh` 経由へ変わった
- 同 helper は CI でも `tools/dev/ensure-operator-ui-playwright-browser.sh`
  を常に呼び、`pnpm exec playwright install chromium` を実行しうる
- job log では download progress が 100% になった後に helper process が戻らず、
  workflow 上の test 実行や app startup へ進んでいない
- したがって旧 `playwright install --with-deps chromium` 問題の再発というより、
  「install step を workflow から helper 内へ移した」形での再発の可能性が高い

## Constraint

- `online-release-staging-verify` は remote URL を叩くだけなので
  Playwright job container 化が比較的単純だった
- `operator-ui-browser-postgres` lane は host runner 上で
  Postgres service container と `make seaweed-up` を使うため、
  staging verify と同形の job container 化は service topology の再設計を伴う
- したがって remediation は
  `file-backed` と `postgres` を分けて判断する余地がある

## Proposed Investigation

- `PR276` 直前/直後の `verify:ci:*` command surface と helper call graph を比較し、
  delegated browser bootstrap が direct cause か確認する
- `ensure-operator-ui-playwright-browser.sh` の
  executablePath 判定、install 実行、post-install return のどこで止まるかを
  CI log 上で可視化する
- CI lane では browser provisioning を
  workflow-managed runtime か explicit bootstrap step に寄せ、
  opaque helper 内の長時間無音処理を避ける設計へ戻せるか評価する
- `file-backed` lane は Playwright Docker 化候補として再評価する
- `postgres` lane は host-runner 維持のまま fix する案と、
  SeaweedFS/service topology ごと再設計して Docker 化する案を比較する
