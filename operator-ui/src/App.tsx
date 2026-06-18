import { useEffect, useMemo, useState } from "react";

import { AuthPrincipal, OperatorApiClient } from "./api";
import { LoginPage } from "./routes/login/LoginPage";
import { OperatorPage } from "./routes/operator/OperatorPage";
import { defaultBaseUrl, messageOf, normalizeBaseUrl } from "./routes/operator/operatorPageSupport";
import { AppShell } from "./shared/layout/AppShell";

export default function App() {
  const location = currentLocation();
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(defaultBaseUrl())), []);

  if (isLoginPathname(location.pathname)) {
    return (
      <AppShell>
        <LoginPage onAuthenticatedReturn={navigateTo} />
      </AppShell>
    );
  }

  if (isOperatorPathname(location.pathname)) {
    return (
      <AppShell>
        <ProtectedOperatorRoute client={client} targetPath={location.pathname + location.search} />
      </AppShell>
    );
  }

  return (
    <AppShell>
      <UnknownRoute pathname={location.pathname} />
    </AppShell>
  );
}

function ProtectedOperatorRoute({ client, targetPath }: { client: OperatorApiClient; targetPath: string }) {
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [principal, setPrincipal] = useState<AuthPrincipal>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      try {
        const session = await client.session();
        if (canceled) {
          return;
        }
        if (session.auth_mode === "enabled" && !session.authenticated) {
          navigateTo(loginURLForTarget(targetPath));
          return;
        }
        setPrincipal(session.principal);
        setState("ready");
      } catch (loadError) {
        if (canceled) {
          return;
        }
        if (messageOf(loadError).toLowerCase().includes("authentication required")) {
          navigateTo(loginURLForTarget(targetPath));
          return;
        }
        setState("error");
        setError(messageOf(loadError));
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [client, targetPath]);

  if (state === "loading") {
    return (
      <section className="rounded-[28px] border border-black/10 bg-white/80 p-8 shadow-sm backdrop-blur">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-teal">Auth Check</p>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight">Checking session</h1>
        <p className="mt-3 text-sm text-black/70">The operator route waits for session confirmation before loading.</p>
      </section>
    );
  }

  if (state === "error") {
    return (
      <section className="rounded-[28px] border border-red-200 bg-red-50 p-8 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-red-700">Auth Error</p>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight text-red-900">Session check failed</h1>
        <p className="mt-3 text-sm text-red-800">{error ?? "unknown error"}</p>
      </section>
    );
  }

  return <OperatorPage principal={principal} onLogout={() => void logoutAndReturnToLogin(client)} />;
}

function UnknownRoute({ pathname }: { pathname: string }) {
  return (
    <section className="rounded-[28px] border border-black/10 bg-white/80 p-8 shadow-sm backdrop-blur">
      <p className="text-sm font-medium uppercase tracking-[0.2em] text-teal">Unknown Route</p>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">No page is registered for this path.</h1>
      <p className="mt-3 text-sm text-black/70">
        The current operator UI is available at <code>/</code>, <code>/operator</code>, and <code>/login</code>.
      </p>
      <p className="mt-2 text-sm text-black/60">Path: {pathname}</p>
    </section>
  );
}

function currentLocation() {
  if (typeof window === "undefined") {
    return { pathname: "/", search: "" };
  }
  return { pathname: window.location.pathname, search: window.location.search };
}

function isOperatorPathname(pathname: string) {
  const normalized = normalizePathname(pathname);
  return normalized === "/" || normalized === "/operator";
}

function isLoginPathname(pathname: string) {
  return normalizePathname(pathname) === "/login";
}

function normalizePathname(pathname: string) {
  return pathname.endsWith("/") && pathname !== "/" ? pathname.slice(0, -1) : pathname;
}

function absoluteURL(targetPath: string) {
  if (typeof window === "undefined") {
    return `http://localhost:4173${targetPath}`;
  }
  return new URL(targetPath, window.location.origin).toString();
}

function loginURLForTarget(targetPath: string) {
  const params = new URLSearchParams({ return_to: absoluteURL(targetPath) });
  return `/login?${params.toString()}`;
}

async function logoutAndReturnToLogin(client: OperatorApiClient) {
  await client.logout();
  navigateTo("/login");
}

function navigateTo(target: string) {
  if (typeof window === "undefined") {
    return;
  }
  window.location.assign(target);
}
