import { ReactNode } from "react";

import { AppShell } from "./shared/layout/AppShell";
import { OperatorPage } from "./routes/operator/OperatorPage";

type RouteDefinition = {
  name: string;
  match: (pathname: string) => boolean;
  render: () => ReactNode;
};

const routes: RouteDefinition[] = [
  {
    name: "operator",
    match: isOperatorPathname,
    render: () => <OperatorPage />,
  },
];

export default function App() {
  const pathname = currentPathname();
  const route = routes.find((candidate) => candidate.match(pathname));

  return <AppShell>{route ? route.render() : <UnknownRoute pathname={pathname} />}</AppShell>;
}

function UnknownRoute({ pathname }: { pathname: string }) {
  return (
    <section className="rounded-[28px] border border-black/10 bg-white/80 p-8 shadow-sm backdrop-blur">
      <p className="text-sm font-medium uppercase tracking-[0.2em] text-teal">Unknown Route</p>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">No page is registered for this path.</h1>
      <p className="mt-3 text-sm text-black/70">
        The current operator UI is available at <code>/</code> and <code>/operator</code>.
      </p>
      <p className="mt-2 text-sm text-black/60">Path: {pathname}</p>
    </section>
  );
}

function currentPathname() {
  if (typeof window === "undefined") {
    return "/";
  }
  return window.location.pathname;
}

function isOperatorPathname(pathname: string) {
  const normalized = pathname.endsWith("/") && pathname !== "/" ? pathname.slice(0, -1) : pathname;
  return normalized === "/" || normalized === "/operator";
}
