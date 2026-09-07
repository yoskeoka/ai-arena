# reversi-game-registration-admitted-artifact
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0039-game-registration-admitted-artifact-registry.md`

## Objective

admission 済みの game bundle artifact を operator が game registration に activate できるようにする。
registration は built-in game の事前登録、game ID の whitelist、または別の metadata-only registration に
依存せず、指定された immutable artifact の manifest と descriptor を source of truth として検証する。

完了境界は、built-in registry に存在しない game ID を持つ有効な game bundle を同一 service process で
upload/admit した後、その `artifact_id` を指定した registration が成功し、作成された scope が manifest
由来の game identity、ruleset、build metadata、同じ artifact digest を返すこととする。upload UI、AI bundle、
bot、queue、ranking、未 activation artifact の cleanup、service restart 後の registry record 再構築は対象外とする。

## Context and Evidence

- staging の Reversi bundle upload は HTTP `201` で成功したが、続く registration は
  `service: bad request: registry: unsupported game "reversi"` で HTTP `400` になった。
- `internal/platform/service/artifact_admission.go:26-55` は bundle manifest を検証し、
  `DescriptorRecord` を渡された registry に登録している。
- `cmd/arena-service/main.go:285-311` は upload 用の `admissionRegistry` を作るが、general submission service を
  `nil` registry で生成している。
- `internal/platform/service/general.go:163-182` は nil registry を `registry.Default()` に置き換える。
- `internal/platform/service/general.go:187-205` は artifact manifest から game metadata を導出するが、
  `internal/platform/service/general.go:377-386` で `LookupVersion` を先に実行する。
- `internal/platform/registry/defaults.go:14-42` の default registry は built-in descriptor のみを持つ。
- `internal/platform/service/artifact_submission_e2e_test.go:68-104` は admission に overlay registry を使うが、
  uploaded game artifact 自体の registration を検証していない。

## Options Considered

### Option A: general service に overlay registry を渡すだけ

最小変更として `admissionRegistry` を general service に渡し、既存の `LookupVersion` を維持する。

- 利点: wiring の変更が小さい。
- 欠点: 指定 digest の exact release ではなく、同一 major の最新 release を参照する。
  patch release が複数ある場合に build metadata と selected artifact が混ざるため、immutable artifact
  activation の契約を満たさない。

### Option B: shared overlay と exact artifact lookup を採用する（推奨）

admission と registration に同じ overlay registry を渡し、`artifact_id` が指定された path では exact
  artifact lookup を使う。lookup 結果と manifest の game identity、version、ruleset、build metadata を検証して
  registration record を作る。

- 利点: selected digest が activation target の唯一の identity になり、built-in game と uploaded game の
  境界も明確になる。
- 欠点: registration の validation path を分け、exact descriptor と manifest の consistency test が必要になる。

### Option C: default registry に Reversi を built-in 登録する

Reversi 固有の descriptor を `registry.Default()` に追加する。

- 却下理由: uploaded artifact の admission が eligibility を決めるという contract に反し、game ID の
  hard-code / whitelist を拡張するだけになる。外部 game を追加するたびに platform core の変更が必要になる。

## Black-box Spec Changes

### `docs/specs/platform-service-general-submission.md` (MODIFY)

次の observable contract を明記する。

- `artifact_id` 付きの新規 game activation は、その digest に対応する admitted game artifact を exact に解決する。
- artifact manifest が game identity、exact version、ruleset、runtime/build metadata の source of truth である。
- built-in registry への事前登録や game ID whitelist の存在は、artifact-backed activation の前提ではない。
- selected artifact が未 admission、game artifact でない、digest/manifest/descriptor が不整合、または ruleset が
  manifest にない場合は同期的に拒否する。
- metadata-only compatibility input は既存利用者向けに残せるが、新規 artifact-backed operation の validation
  path を代替しない。

### `docs/specs/platform-game-registry.md` (MODIFY)

official sandboxed artifact の admission registry は built-in registry の単なる名前 lookup ではなく、
admitted descriptor record を exact artifact identity で解決する path を持つことを明記する。built-in descriptor
と uploaded descriptor は同一 `GameDescriptor` 抽象へ解決するが、uploaded descriptor の eligibility は bundle
admission と整合性検証で決まり、built-in descriptor の hard-code へフォールバックしてはならない。

## Code Change Map

- `docs/issues/0039-game-registration-admitted-artifact-registry.md` (NEW)
  - staging の admission 成功 / registration 失敗と再現可能な原因を記録する。
- `docs/specs/platform-service-general-submission.md` (MODIFY)
  - artifact-backed activation の exact identity、manifest source of truth、validation failure boundary を追加する。
- `docs/specs/platform-game-registry.md` (MODIFY)
  - built-in と admitted artifact の registry path、および hard-code fallback 禁止を明確化する。
- `cmd/arena-service/main.go` (MODIFY)
  - artifact admission と general submission が同一の writable WASI overlay registry を参照するよう wiring を修正する。
- `internal/platform/service/general.go` (MODIFY)
  - artifact ID 指定時に exact admitted artifact descriptor を解決し、descriptor / manifest / requested ruleset の
    整合性を確認して registration metadata を構築する。game ID/version の通常 lookup を artifact-backed path の
    eligibility gate にしない。
- `internal/platform/service/general_test.go` (MODIFY)
  - built-in registry に存在しない admitted game artifact の registration 成功、unknown artifact、ruleset mismatch、
    artifact metadata mismatch の回帰を追加する。
- `internal/platform/service/artifact_submission_e2e_test.go` (MODIFY)
  - admission 後に同じ artifact を registration へ渡し、digest と manifest-derived scope metadata が一致することを
    service-level E2E で確認する。既存の unrelated AI / queue assertions は維持する。
- `internal/platform/service/http_test.go` (MODIFY, conditional)
  - shared registry を使う upload -> registration HTTP path が必要な場合に追加し、HTTP status と response body を固定する。

## Execution Steps

1. `platform-service-general-submission.md` と `platform-game-registry.md` を先に更新し、artifact-backed path の
   source of truth と failure boundary を固定する。TypeSpec の既存 upload / registration wire contract は変更しない。
2. `GeneralSubmissionService.RegisterGame` の artifact-backed path を整理する。artifact ID から bundle を読み、
   exact admitted descriptor を lookup し、artifact manifest と descriptor の game ID、exact game version、ruleset、
   build mode、builder ID、artifact digest を consistency check する。requested ruleset の player count と bot limit は
   selected manifest から取得する。
3. `newCLIApp` の service construction を修正し、artifact admission、general registration、worker invocation が同じ
   overlay registry を共有することを確認する。`registry.Default()` を mutable global registry として拡張したり、
   `reversi` を built-in に追加したりしない。
4. unit / service E2E に non-built-in game artifact の成功 path と validation failure を追加する。特に同一 major の
   別 release がある場合に、requested artifact の digest と build metadata が selected record に残ることを確認する。
5. staging deploy 後、released Reversi game ZIP を再 upload し、HTTP `201` admission、registration success、scope row の
   `reversi@1.0.0 / standard` と同じ artifact digest を確認する。前回失敗の HTTP `400` response と対比した evidence を残す。
6. 実装 branch では検証 evidence を取得してからこの plan と `Addresses:` の解決済み issue を削除し、実装 PR の
   conditional closure metadata から issue の履歴を辿れるようにする。

## Dependencies and Parallelism

- spec 更新は implementation より先に行う。
- `cmd/arena-service/main.go` の wiring 修正と `general.go` の validation refactor は同じ registry lifetime contract に
  依存するため、1 worktree では直列に実施する。
- unit test の fixture 整備と spec の review は並行できるが、service E2E は exact lookup の実装後に行う。
- open な PR #316 (`plan/games-inline-bundle-upload`) は Games page の upload UX を扱う。#316 はこの backend bug を
  解決しないため、0110 の実装・verification の dependency として明記し、どちらか一方の merge だけで全体完了とは扱わない。
- Postgres / R2 の restart 後に process-local registry descriptor を再構築する設計は別計画とし、この plan の same-process
  admission-to-registration completion boundary を越えて拡張しない。

## Verification

- `go test ./internal/platform/registry ./internal/platform/service` が pass する。
- admitted artifact が built-in registry に存在しない game ID でも registration に成功する。
- registration record の game ID、exact game version、ruleset、build mode、builder ID、artifact digest が selected
  artifact の manifest / descriptor と一致する。
- artifact ID が unknown、artifact kind が game でない、ruleset が manifest にない、manifest と descriptor が不一致の
  場合は registration が失敗し、scope record を保存しない。
- 同一 major に複数 release がある場合、通常の latest lookup を使わず、指定 artifact digest の release が選択される。
- `make test`、`make lint`、`git diff --check` を実行する。TypeSpec / generated client は wire contract 無変更である
  ことを確認する。
- staging で Reversi release asset `v0.1.0` の upload が HTTP `201`、manifest の game version `1.0.0` に対する registration が成功し、scope に `reversi@1.0.0 / standard` と
  `sha256:98bd46609016dc763bcbfff747c6705c7f1608a86d164d77b38db208d0d1c0df` が表示される。

## Risks and Mitigations

- shared registry を渡すだけで exact artifact selection を実装しないと、latest release と selected digest が混ざる。
  - mitigation: artifact-backed path は `LookupArtifact` を起点にし、selected descriptor を record metadata の source にする。
- built-in fallback を残すと、未知 game ID が再び hard-code catalog によって拒否される。
  - mitigation: artifact ID がある request は built-in `LookupVersion` を eligibility gate にせず、admitted artifact の検証結果だけを使う。
- manifest と registry descriptor のどちらか一方だけを信頼すると、metadata drift を見逃す。
  - mitigation: digest、game identity、version、ruleset、build metadata を両方から照合し、不一致時は保存前に拒否する。
- staging で同じ digest を再 upload すると既存 artifact / release と競合する。
  - mitigation: artifact admission の既存 idempotency semantics を使い、registration ID だけ新しい一意値にする。
- #316 の UI 側が手入力 fields を削除する前に backend だけを修正しても、現在の staging UI の upload UX は変わらない。
  - mitigation: 0110 は backend contract の成立を証明し、#316 の UI verification と別々の acceptance evidence として扱う。
