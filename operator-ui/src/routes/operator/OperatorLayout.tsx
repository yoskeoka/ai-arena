import { ReactNode } from "react";

import { AuthPrincipal } from "../../api";
import { operatorNavItems, OperatorRoute, operatorRouteTitle } from "./operatorRoutes";

type OperatorLayoutProps = {
  route: OperatorRoute;
  baseUrl: string;
  onBaseUrlChange: (value: string) => void;
  principal?: AuthPrincipal;
  onLogout?: () => void;
  children: ReactNode;
};

export function OperatorLayout({ route, baseUrl, onBaseUrlChange, principal, onLogout, children }: OperatorLayoutProps) {
  return (
    <>
      <header className="overflow-hidden rounded-[28px] border border-black/10 bg-white/80 shadow-sm backdrop-blur">
        <div className="border-b border-black/10 px-6 py-6">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-2">
              <p className="text-sm font-medium uppercase tracking-[0.2em] text-teal">Phase 7 Operator Surface</p>
              <h1 className="text-3xl font-semibold tracking-tight">AI Arena Operator Console</h1>
              <p className="max-w-3xl text-sm text-black/70">
                Registration, invite issuance, submission, request, run follow-up, and ranking verification in one route family.
              </p>
            </div>
            <div className="flex min-w-80 flex-col gap-3 text-sm">
              <label className="flex flex-col gap-2">
                <span className="font-medium text-black/70">Operator API base URL</span>
                <input
                  className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm outline-none transition focus:border-accent"
                  value={baseUrl}
                  onChange={(event) => onBaseUrlChange(event.target.value)}
                  placeholder="Leave blank for local Vite proxy, or set https://ai-arena-service.onrender.com"
                />
              </label>
              {principal ? (
                <div className="flex items-center justify-between rounded-2xl border border-black/10 bg-paper px-4 py-3">
                  <span className="text-black/65">Signed in as @{principal.provider_login}</span>
                  {onLogout ? (
                    <button
                      className="rounded-full border border-black/15 px-3 py-1 text-xs font-medium uppercase tracking-[0.16em] text-black/70 transition hover:border-black/30 hover:text-black"
                      onClick={onLogout}
                      type="button"
                    >
                      Logout
                    </button>
                  ) : null}
                </div>
              ) : null}
            </div>
          </div>
        </div>

        <nav className="flex flex-wrap items-center gap-3 px-6 py-4">
          {operatorNavItems.map((item) => {
            const active = route.kind === item.kind;
            return (
              <a
                key={item.kind}
                href={item.href}
                data-testid={item.testId}
                className={`rounded-full px-4 py-2 text-sm font-semibold no-underline transition ${
                  active ? "bg-ink text-paper" : "bg-paper text-black/75 hover:bg-white hover:text-black"
                }`}
              >
                {item.label}
              </a>
            );
          })}
          <div className="ml-auto rounded-full border border-black/10 bg-paper px-4 py-2 text-xs uppercase tracking-[0.16em] text-black/55">
            {operatorRouteTitle(route)}
          </div>
        </nav>
      </header>

      <main className="space-y-6" data-testid={`operator-panel-${route.kind === "run-detail" ? "run-detail" : route.kind}`}>
        {children}
      </main>
    </>
  );
}
