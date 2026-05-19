# spec-boundary-cleanup
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の spec 群について、product/platform が提供する契約と contributor workflow /
development harness の説明を分離し、`docs/specs/` を再び「ai-arena が提供する共通契約の正本」
として読める状態へ戻す。

この plan は以下を同時に達成する。

- `docs/issues/0016-specs-should-not-contain-development-harness-contracts.md` の指摘どおり、`docs/specs/` 配下の
  development harness 寄り文書を棚卸しし、product spec として残すべき境界を明確にする
- `docs/issues/0017-platform-spec-should-describe-provided-contracts-not-consumer-guidance.md` の指摘どおり、
  `platform.md` / `game-master.md` と隣接 spec を、consumer repo の開発運用説明ではなく
  ai-arena が提供する contract の記述へ揃える
- 今後同種の drift を review で早めに検出できるよう、spec index / checklist に境界ルールを残す

Addresses:

- `docs/issues/0016-specs-should-not-contain-development-harness-contracts.md`
- `docs/issues/0017-platform-spec-should-describe-provided-contracts-not-consumer-guidance.md`

## Context

- `docs/project-plan.md` は ai-arena を複数 game を載せる platform として定義しており、spec の主語も
 「ai-arena が固定する共通契約」である必要がある
- `docs/design-decisions/core-beliefs.md` の AI-first / correctness over speed により、spec の分類境界も
  実装都合より durable な contract 読解を優先して整えるべきである
- `docs/design-decisions/adr.md` にはこの issue のために新しい architecture decision を追加する必要はなく、
  今回は spec の責務境界整理として扱う
- `docs/lessons.md` にはすでに
  「platform spec は consumer repo に許可を与える書き方をしない」
  「ai-arena の spec は platform / runner / SDK が何を提供するかだけを書く」
  という rule があり、execution ではこれを正本として周辺 spec を監査する
- 現状の `docs/specs/platform.md`、`docs/specs/game-master.md`、`docs/specs/platform-common-contract.md`、
  `docs/specs/ai-runtime.md` には、provided contract と consumer-side ownership / verification /
  tagged import 運用の説明が混在している
- `docs/specs/workflow-linter.md` は product behavior ではなく repo-local workflow maintenance の契約であり、
  `docs/specs/` に置き続けると issue 0016 の境界に反する

## Scope

- `docs/specs/` 配下の文書を product contract と development harness で棚卸しする
- `platform` / `game-master` / `platform-common-contract` / `ai-runtime` の provided contract wording を整理する
- workflow/development harness 文書の移設先を決め、index と相互参照を更新する
- issue close に必要な docs 更新と issue file move を行う

この plan では以下は扱わない。

- platform / runtime / sidecar の code 実装変更
- 新しい public API や DTO の追加
- external repo 向け migration guide の全面整備
- docs 全体の大規模ディレクトリ再編

## Option Review

### Option A: 問題文だけを狭く修正し、spec 置き場はほぼ維持する

- 利点: diff が小さく、既存リンクの更新も少ない
- 欠点: issue 0016 の「product spec と development harness の分離」が不十分なまま残る
- 欠点: `workflow-linter.md` のような repo-local contract が `docs/specs/` に残り、再発しやすい

### Option B: `docs/specs/` を product contract のみに寄せ、workflow/development harness 文書を分離する

- 利点: issue 0016 と 0017 を 1 本の方針で同時に閉じられる
- 利点: spec index と review checklist に durable な境界ルールを残せる
- 欠点: 文書移設と参照更新が増え、狭い wording fix より diff は大きくなる

### Recommendation

Option B を採る。

- `docs/project-plan.md` の「platform が何を提供するか」を spec の正本へ戻す目的に最も整合する
- 既存 lesson の「provided contract だけを書く」「consumer repo への許可文型を使わない」と一致する
- 文書の移設は増えるが、code change を伴わない docs-only で完結でき、今ここで整理する価値が高い

## Design Decision

追加 ADR は作らない。

- `docs/specs/` は ai-arena が提供する product/platform/runtime contract の正本として保つ
- contributor workflow、repo-local lint/hook、tooling maintenance のような development harness 文書は
  `docs/specs/` 外の development/workflow 向け置き場へ移す
- product spec 本文では、consumer repo の ownership・asset 配置・tagged import 採用手順のような
  運用説明を正本化しない
- external repo 側の guide が必要なら、spec ではなく guide / issue / migration note 側へ逃がす

## Spec Changes

### `docs/specs/platform.md`

- platform が提供する責務、公開境界、registry/runtime/session との契約だけを残す
- consumer repo の verification asset 配置、repo ownership、tagged import 運用の説明を除く
- 非責務は「ai-arena が規定しない」と書き、外部に許可を与える文型を避ける

### `docs/specs/game-master.md`

- game master sidecar が満たすべき contract と transport/lifecycle 要件へ絞る
- external repo の release 運用、import audit、CI 運用のような consumer guidance を外す
- ai-arena が固定する公開境界と、固定しない領域の説明を責務ベースに揃える

### `docs/specs/platform-common-contract.md`

- 共通 DTO / metadata / action status の契約に集中させる
- tagged import や consumer repo 依存面の説明が残るなら、provided vocabulary の説明に必要な最小限まで削る

### `docs/specs/ai-runtime.md`

- runtime contract と support policy のうち、ai-arena が提供する共通 runtime 契約として必要な説明だけを残す
- game-owned lane や ownership 説明が spec 境界を越えている箇所は、必要に応じて別置き場へ移すか最小化する

### `docs/specs/README.md`

- `docs/specs/` は product/platform/runtime contract の置き場であることを明記する
- review checklist に、development harness 文書混入と consumer guidance 混入を検出する確認点を追加する

### development/workflow docs outside `docs/specs/`

- `docs/specs/workflow-linter.md` を `docs/specs/` 外へ移し、repo-local workflow maintenance 文書として位置付け直す
- 棚卸しの結果、`github-actions-pinning.md` や `go-quality-gates.md` など周辺文書にも
  product spec ではなく contributor workflow 文書として扱うべきものがあれば、同じ方針で移設または分類注記を行う
- spec index や関連 spec からのリンク先を新しい置き場に更新する

## Expected Code Changes

なし。execution は docs のみを変更する。

## Sub-tasks

- [ ] Audit `docs/specs/` and classify each document as product contract or development harness / contributor workflow
- [ ] Decide the non-`docs/specs/` destination for workflow/development harness docs and move the identified files there
- [ ] Rewrite `docs/specs/platform.md` and `docs/specs/game-master.md` so they describe provided contracts, responsibilities, and non-responsibilities only
- [ ] Audit `docs/specs/platform-common-contract.md` and `docs/specs/ai-runtime.md` for the same consumer-guidance drift and trim or relocate offending passages
- [ ] Update `docs/specs/README.md` and any affected cross-references so the spec boundary is explicit to future reviewers
- [ ] Move `docs/issues/0016-specs-should-not-contain-development-harness-contracts.md` and `docs/issues/0017-platform-spec-should-describe-provided-contracts-not-consumer-guidance.md` to `docs/issues/done/` when the docs diff fully covers their intent

## Parallelism

- [parallel] `platform.md` と `game-master.md` の wording audit は並行に進められる
- [parallel] `platform-common-contract.md` と `ai-runtime.md` の consumer-guidance audit は、主要方針が固まれば独立に進められる
- [parallel] `docs/specs/` 全体の棚卸しと移設候補の抽出は、provided contract wording の本文修正と別に進められる
- issue close の判断は、本文修正と移設・index 更新の両方が揃ってから行う

## Risks and Mitigations

- harness 文書の移設先を大きく作り込みすぎると、docs-only cleanup から taxonomy 設計へ膨らむ
  - mitigation: 今回は最小の非-`docs/specs/` 置き場を選び、必要な分類 rule だけ残す
- consumer guidance を削りすぎて、公開境界の意味まで読めなくなる
  - mitigation: ai-arena が固定する contract、公開 import surface、非責務境界の説明は残し、採用手順や ownership 運用だけを外す
- issue 0016 の棚卸し対象を狭く見積もると、`workflow-linter.md` 以外の同種文書が残る
  - mitigation: `docs/specs/` 全件を一度分類し、移設しない文書は「なぜ product contract と言えるか」を diff で説明できる状態にする

## Verification

The execution PR is complete when the following are true.

- `docs/specs/` 配下に残る文書が、ai-arena の product/platform/runtime contract として読めるものに限定されている
- `docs/specs/platform.md` と `docs/specs/game-master.md` から consumer repo の開発運用説明が外れ、provided contract が主語になっている
- `docs/specs/platform-common-contract.md` と `docs/specs/ai-runtime.md` に残る external repo 言及が、contract を共有するために必要な最小限に整理されている
- workflow/development harness 文書の移設先と spec index の参照更新が揃い、reviewer が product spec と contributor workflow を混同しない
- `docs/issues/0016-specs-should-not-contain-development-harness-contracts.md` と `docs/issues/0017-platform-spec-should-describe-provided-contracts-not-consumer-guidance.md` を
  `docs/issues/done/` へ移せるだけの coverage が PR diff で確認できる
