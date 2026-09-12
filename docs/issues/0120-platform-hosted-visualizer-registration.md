# platform-hosted-visualizer-registration

## Summary

game 提供者が既定形式で visualizer program を登録し、ai-arena の画面内で game 固有 visualizer を動かす経路を将来検討する。
候補は TypeScript asset または WASM を含むが、実行形式、asset delivery、sandbox、host page との API、version compatibility は未決である。

## Why It Remains Open

Phase 8 の first delivery は、ai-arena を execution engine と anonymous exported-state API に限定し、game 提供者が
visualizer を別 host で自由な形態（web、native application、CLI 等）として提供できるようにする。Reversi の Phaser viewer はその
reference implementation であり、platform が game 内容を静的 link したり provider code を実行したりしない。

platform-hosted visualizer を同じ delivery に含めると、untrusted third-party program の isolation、DOM/canvas access、asset
integrity / signing、browser permission、API credential delegation、cache / rollout / rollback、versioned visualizer ABI を同時に
決める必要がある。その前提は public replay API の成立には不要である。

## Follow-up Boundary

- game provider が optional visualizer を登録しない場合でも game registration、public replay API、external viewer consumer が
  成立することを維持する。
- web program の登録形式（TypeScript bundle、WASM、または別形式）、execution isolation、host-to-visualizer API、artifact / asset
  admission、integrity、capability / network policy、upgrade / rollback の責務を比較する。
- ai-arena の page で canvas 等へ描画する場合の DOM ownership、accessibility、failure containment、resource budget、game content を
  静的 link しない dynamic loading contract を定義する。
- chosen model を public API / authentication policy と混同せず、black-box host / visualizer compatibility と security boundary を
  持つ別 execution plan を review してから実装する。
