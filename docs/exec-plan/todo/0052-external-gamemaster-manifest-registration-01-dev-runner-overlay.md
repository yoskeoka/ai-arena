# external-gamemaster-manifest-registration-01-dev-runner-overlay
**Execution**: Use `/execute-task` to implement this plan.

Addresses: `docs/issues/0018-external-gamemaster-manifest-registration.md`

## Objective

built-in registry に未登録の external game master でも、consumer repo が `arena-runner` に
game master manifest file を渡すだけで、開発中バージョンをローカル検証できる入口を追加する。
この段階では registry の persisted record や built-in lookup を変更せず、
runner が opt-in の local-subprocess descriptor overlay として consumer-supplied manifest から
local-subprocess descriptor を構築して
fresh run を開始できることを到達点に置く。

## Context

- `docs/specs/platform-game-registry.md` は persisted record と process-local descriptor を分離している
- `docs/specs/game-master.md` は local subprocess と future external adapter で共通の論理 API を固定している
- issue 0018 の本質は「built-in registry 非登録 game を tagged runner で自己完結検証できない」ことであり、
  正式 registration policy の決定までは要求していない
- user decision:
  - game master metadata の source of truth は CLI ではなく manifest とする
  - executable path は manifest file 基準の relative path または absolute path で解決する
  - 初期スコープは fresh run only とし、resume / replay / history build は後続へ分離する

## Scope

- `arena-runner` に game master manifest file を受け取る opt-in entry を追加する
- runner が manifest から game metadata と runtime entrypoint を読み、local-subprocess descriptor overlay として local-subprocess descriptor を構築する
- local subprocess game master binary の path を manifest file 基準で解決する
- built-in registry lookup と同じ metadata compatibility / build flow へ接続する
- external consumer repo fixture で fresh run e2e を通す

この plan では以下を扱わない。

- built-in registry の persisted record や default resolver の変更
- official registered game としての admission policy
- Docker / WASM/WASI など別 runtime kind の採否
- resume / replay / history build の manifest entry
- online service / DB-backed registration flow

## Spec Changes

### `docs/specs/platform-game-registry.md`

- built-in registry に加えて、runner が opt-in の local-subprocess descriptor overlay として
  consumer-supplied manifest から local-subprocess descriptor を構築できる経路を追記する
- この経路は persisted `DescriptorRecord` を追加せず、runner local overlay から local-subprocess descriptor を構築するものとして扱うことを明記する
- registry lookup key や built-in lookup contract は変更しないことを明記する

### `docs/specs/game-master.md`

- external game master 開発用 manifest の責務を追加する
- 初期スコープは `local-subprocess` + fresh run only とする
- manifest に少なくとも以下を持たせることを定義する
  - `game_id`
  - `game_version`
  - `ruleset_version` または同等の fresh-run compatibility に必要な metadata
  - runtime kind
  - command / args または同等の entrypoint
- executable path 解決は manifest file 基準で行うことを明記する

### `docs/specs/platform.md`

- runner が game master manifest file を受け取る opt-in dev-only entry を持てることを追記する
- manifest が与えられた場合、game master metadata の source of truth は manifest であることを明記する
- metadata 不整合や path 解決失敗は match loop 開始前に fail-fast することを明記する

## Expected Code Changes

- `cmd/arena-runner/`
  - game master manifest file を受け取る flag 追加
  - manifest parse / validation / path resolution
  - local-subprocess descriptor overlay path の追加
- game master manifest schema / loader package
- runner-local descriptor adapter
- external fixture or test helper for a consumer-supplied local subprocess game master
- fresh-run e2e / compatibility failure coverage

## Sub-tasks

- [ ] runner opt-in manifest entry の CLI contract を追加する
- [ ] game master manifest schema と validation rules を定義する
- [ ] manifest file 基準の path resolution を実装する
- [ ] built-in registry path と並立する local-subprocess descriptor overlay path を追加する
- [ ] manifest metadata を source of truth にした compatibility validation を追加する
- [ ] external fixture を用意し、fresh run e2e を追加する
- [ ] manifest 不正 / path 不正 / metadata mismatch の failure coverage を追加する

## Parallelism

- spec drafting と fixture 準備は並行で進められる
- manifest loader と runner integration は contract 固定後に並行で進められる

## Dependencies

- informs: `0053-external-gamemaster-manifest-registration-02-official-runtime-admission.md`
- blocks: `0054-external-gamemaster-manifest-registration-03-resume-replay-follow-up.md`

## Risks and Mitigations

- manifest 入口が ad hoc CLI flag 群の置き換えになり切らず、metadata source of truth が曖昧になる
  - mitigation: game master metadata は manifest のみを正本とし、CLI は manifest path の受け渡しに限定する
- built-in registry path に accidental regression が入る
  - mitigation: dev-only overlay path を opt-in に限定し、default lookup とテストを明確に分ける
- cwd 基準で command path を解決して consumer repo で再現不能になる
  - mitigation: relative path は必ず manifest file 基準で解決する
- fresh run と resume/replay を同時に抱えて scope が膨らむ
  - mitigation: 初回は fresh run only とし、history / snapshot entry は `0054` へ送る

## Design Decisions

- external game master の開発用入口は「registry 一時登録」ではなく「runner-local local-subprocess descriptor overlay」として扱う
- built-in registry の persisted shape と default resolver は変更しない
- game master metadata の source of truth は manifest とする
- local subprocess はこの plan では開発用 runtime kind としてのみ扱う
