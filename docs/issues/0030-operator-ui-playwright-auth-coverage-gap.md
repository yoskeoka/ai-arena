# operator-ui Playwright auth coverage gap

## Summary

current `operator-ui` の Playwright lane は auth-disabled backend または
login を通らない managed backend に依存しており、
GitHub login を含む browser auth flow を repo-owned 自動 verification できていない。

## Why this matters

- project 方針として、agent が auth を含めて自律確認できない状態は弱い
- current `0087` では human manual GitHub login で前進できるが、
  browser automation の正本 verification が auth を素通りしている
- login route / callback / session cookie / return flow の regression を
  CI や local Playwright だけでは捕まえられない
- login page の `Skip to operator route` link も、
  auth-enabled backend では `/operator -> /login` へ戻されるだけで
  Playwright の auth 回避経路として機能していない

## Current acceptable stop condition

- first operator signup と GitHub login を human manual local lane で確認できること
- Playwright lane は当面、operator UI の auth 後 surface regression に限定すること
- login page の `Skip to operator route` は
  auth-enabled verification で意味を持たない暫定 UI として扱い、
  follow-up で削除または役割整理すること

## Follow-up direction

- local で扱える test auth provider か GitHub flow stub を導入し、
  browser automation から login -> callback -> session establishment を通せるようにする
- もしくは frontend/backend same-origin 化や auth test seam を先に整え、
  CI/local の managed lane でも auth cookie contract を検証できるようにする
- auth-enabled backend を local / Playwright lane で起動するときに、
  auth table が未作成でも起動失敗だけで詰まらないよう
  schema apply bootstrap をどの entrypoint が責務として持つか明確化する
- auth verification seam を入れるまでの interim でも、
  login page の `Skip to operator route` を
  misleading な導線として残すかどうかを判断し、
  残すなら明確に test-only であることを示す
