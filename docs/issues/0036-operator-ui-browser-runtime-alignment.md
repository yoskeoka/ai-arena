# operator-ui-browser-runtime-alignment

## Summary

`operator-ui` verification では、local / CI / Postgres / SeaweedFS の
bootstrap 責務が lane ごとに不揃いである。

- Playwright browser bootstrap は local helper 起点で自動化されている
- Postgres は workflow や human が先に起動する前提と、
  script 内 reset/bootstrap が混在している
- SeaweedFS は一部 lane で script 内自動起動を期待している
- staging verify の remote lane は Playwright Docker runtime 済みだが、
  `operator-ui-browser` CI lane は host-runner + helper bootstrap 前提である

この asymmetry 自体が current failure の直接原因でなくても、
runtime contract を読みづらくし、CI / local の挙動差や保守コストを増やしている。

## Why This Needs Separate Planning

- current `operator-ui-browser` hang はまず止血が必要であり、
  helper regression の切り分けと runtime/topology 再設計を同じ変更に混ぜると scope が広がりすぎる
- 一方で、`postgres` と `SeaweedFS` の扱いの不揃いは
  将来 `Plan C` 系の Docker/runtime 再設計を考えるときの主要論点になる
- そのため immediate fix とは別に、
  runtime/bootstrap alignment を専用 exec-plan として確保しておく価値がある

## Suspected Problem Areas

- local canonical lane と CI lane が同じ helper surface を共有しているが、
  実際には required runtime が異なる
- Postgres は external service 前提と self-bootstrap 前提が lane ごとに混ざっている
- SeaweedFS は script 内 `make seaweed-up` に依存する lane があり、
  CI container 化や service-based topology へ寄せにくい
- browser provisioning を helper に隠すか workflow に出すかの boundary が揺れている

## Desired Future Direction

- local / CI / remote で、どの runtime を誰が起動するのかを明示的に揃える
- Postgres / SeaweedFS / Playwright browser の各 runtime について、
  helper responsibility と workflow responsibility を分離する
- `operator-ui-browser` の file-backed lane と postgres lane が
  Docker runtime を採るとしても、service topology を説明可能な contract にする
