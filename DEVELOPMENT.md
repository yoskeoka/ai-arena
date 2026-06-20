# Development Setup

Setup guide for contributor local development.

## 0. Prerequisites

Go
Nodejs, pnpm

## 1. Create local secrets

1. Copy `.env.example` to `.env`.
2. Issue developer-local secrets and place them in `.env`.
3. For GitHub OAuth local verification, register this callback URL in your local OAuth app:
   `http://localhost:10000/auth/github/callback`
4. If you use the repo's `.envrc`, run `direnv allow`.

`.env` is developer-local and must not be committed.

## 2. Install extra local tools

Install these repo-level dependencies before running the local stack:

- `atlas`
- `aws` CLI
- `direnv`
- `docker` / `docker compose`
- `psql`
- `seaweedfs`

## 3. Bootstrap local dependencies and seed data

Run these commands from the repo root:

```sh
make up
make migrate
make local-invite-url
```

## 4. Start the local services

Use two terminals.

Terminal 1:

```sh
make start-backend-local
```

This keeps the backend process in the foreground and writes logs to the current terminal.

Terminal 2:

```sh
make start-frontend-local
```

This starts the Vite dev server on `http://localhost:5173` and also keeps logs attached to the current terminal.

## 5. Manual auth verification

After both processes are up:

1. Run `make local-invite-url`
2. Open the returned `invite_url`
3. Complete GitHub login
4. Confirm that the browser returns to the operator surface

Do not mix `127.0.0.1` and `localhost` during the auth flow.
The local GitHub OAuth callback is registered on `localhost`, so the browser-facing frontend origin should also stay on `localhost` for that verification lane.

For the Playwright-owned browser regression and artifact capture lanes, see `docs/development/operator-ui-local-verification.md`.
