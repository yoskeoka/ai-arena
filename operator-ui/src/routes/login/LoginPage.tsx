import { useEffect, useMemo, useState } from "react";

import { OperatorApiClient } from "../../api";
import { defaultBaseUrl, messageOf, normalizeBaseUrl } from "../operator/operatorPageSupport";

type LoginPageProps = {
  onAuthenticatedReturn: (target: string) => void;
};

export function LoginPage({ onAuthenticatedReturn }: LoginPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(defaultBaseUrl())), []);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [errorMessage, setErrorMessage] = useState<string>();
  const [authMode, setAuthMode] = useState<"disabled" | "enabled">("enabled");

  const inviteToken = queryParam("invite_token");
  const loginError = queryParam("error");
  const returnTo = resolvedReturnTo();

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      try {
        const session = await client.session();
        if (canceled) {
          return;
        }
        if (session.auth_mode === "enabled" && session.authenticated) {
          onAuthenticatedReturn(returnTo);
          return;
        }
        setAuthMode(session.auth_mode);
        setState("ready");
      } catch (error) {
        if (canceled) {
          return;
        }
        setState("error");
        setErrorMessage(messageOf(error));
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [client, onAuthenticatedReturn, returnTo]);

  const loginHref = client.githubLoginURL(returnTo, inviteToken);

  return (
    <section className="rounded-[32px] border border-black/10 bg-white/85 p-8 shadow-sm backdrop-blur">
      <p className="text-sm font-medium uppercase tracking-[0.2em] text-teal">Operator Login</p>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">Sign in with GitHub</h1>
      <p className="mt-3 max-w-2xl text-sm leading-6 text-black/70">
        The first landing keeps OAuth callback and session issuance on the backend. This page only starts the login
        flow and returns you to the operator surface after the backend issues the session cookie.
      </p>
      {inviteToken ? (
        <p className="mt-4 rounded-2xl border border-black/10 bg-paper px-4 py-3 text-sm text-black/65">
          Invite token detected. The first successful GitHub login will consume it for account bootstrap.
        </p>
      ) : null}
      {loginError ? (
        <p className="mt-4 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {messageForLoginError(loginError)}
        </p>
      ) : null}
      {state === "error" && errorMessage ? (
        <p className="mt-4 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {errorMessage}
        </p>
      ) : null}
      <div className="mt-6 flex flex-wrap items-center gap-3">
        {authMode === "enabled" ? (
          <a
            className="inline-flex items-center rounded-2xl bg-ink px-5 py-3 text-sm font-medium text-paper transition hover:opacity-90"
            href={loginHref}
          >
            Continue with GitHub
          </a>
        ) : (
          <a
            className="inline-flex items-center rounded-2xl bg-ink px-5 py-3 text-sm font-medium text-paper transition hover:opacity-90"
            href="/operator"
          >
            Continue to operator UI
          </a>
        )}
        <a className="text-sm text-black/65 underline decoration-black/20 underline-offset-4" href="/operator">
          Skip to operator route
        </a>
      </div>
      {state === "ready" && authMode === "disabled" ? (
        <p className="mt-4 text-sm text-black/60">
          Auth is disabled in the current backend configuration, so the operator route remains directly reachable.
        </p>
      ) : null}
      {state === "loading" ? <p className="mt-4 text-sm text-black/60">Checking current session...</p> : null}
    </section>
  );
}

function resolvedReturnTo() {
  const raw = queryParam("return_to");
  if (raw) {
    return raw;
  }
  if (typeof window === "undefined") {
    return "http://localhost:4173/operator";
  }
  return new URL("/operator", window.location.origin).toString()
}

function queryParam(name: string) {
  if (typeof window === "undefined") {
    return "";
  }
  return new URLSearchParams(window.location.search).get(name) ?? "";
}

function messageForLoginError(code: string) {
  switch (code) {
    case "signup_invite_required":
      return "An invite token is required to create a new account.";
    case "signup_invite_invalid":
      return "The invite token is invalid, expired, or already consumed.";
    case "token_exchange_failed":
      return "GitHub token exchange failed.";
    case "github_profile_failed":
      return "GitHub profile lookup failed.";
    case "missing_code":
      return "GitHub callback did not include an authorization code.";
    default:
      return "Login failed. Retry the GitHub sign-in flow.";
  }
}
