# staging-production-observability-evidence-gap

## Summary

staging / production の実際の deploy 状態、現在動いているプロセスの commit、起動失敗や異常終了の
log を確認するための証跡が、現状は主に Render の Dashboard と service log に依存している。
低コストの Render 運用では、起動中プロセスの log を常時参照・検索・保持できることが保証されず、
deploy が成功したか、どの commit が live か、異常の原因が何かを Render の状態だけに頼らず確認しにくい。

## Observed Evidence

2026-09-10 時点で、staging service は次の状態として観測されている。

- repository: `yoskeoka/ai-arena`
- branch: `main`
- live commit: `b5982c4`（PR #321 のマージ時点）
- URL: `https://ai-arena-staging-p4ml.onrender.com`

PR #323 は `main` に `bcaaa792f4080586f194df66b33b70dcee032cf0` としてマージされたが、同 commit
を対象にした Render deploy は、worker の起動時に `service: another worker already owns this queue` を
出力して失敗した。後続ログに `Detected service running on port 10000` があっても、Render Dashboard の
最終状態は `Deploy failed` であり、live commit の同定根拠にはならなかった。

アプリケーションの read-only endpoint は `/healthz` の `{"status":"ok"}` だけで、プロセスの commit
や build version を返す endpoint は存在しない。GitHub Actions の staging workflow も Render deploy
hook の起動と target SHA の記録までで、Render 側の最終 deploy 状態や live process identity を永続的に
集約しない。

## Impact

- stg / prod の URL がどの commit、build、起動世代で動いているかを外部から検証できない。
- Render の deploy log が参照できない、または保持されない場合、fatal、startup failure、restart loop、
  worker lock 競合などの原因調査に必要な証跡を失う。
- GitHub Actions の deploy trigger 成功、Render の deploy 成功、runtime の稼働確認を同じ証跡として
  扱えず、release acceptance と障害対応の判断が provider の画面状態に依存する。

## Follow-up Boundary

この issue では具体的な provider、SaaS、endpoint、log forwarding 方式を選定・実装しない。別途、
低コスト運用を維持したまま、少なくとも次の観測可能性を定義してから解決策を選ぶ。

- deploy status、target SHA、live process identity を stg / prod で相関できること
- fatal、startup failure、異常終了、restart loop などの最小限の error evidence を Render 外にも保持できること
- log / error payload の retention、secret・credential・個人情報の除外、provider 障害時の扱いを定義すること
- GitHub Actions、Render、runtime の各状態を混同せず、どの観測がどの判断を支えるかを記録できること
