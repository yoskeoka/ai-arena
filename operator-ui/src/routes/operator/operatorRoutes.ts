export type OperatorRoute =
  | { kind: "overview" }
  | { kind: "invites" }
  | { kind: "games" }
  | { kind: "submissions" }
  | { kind: "requests" }
  | { kind: "rankings" }
  | { kind: "run-detail"; runId: string };

export const operatorNavItems = [
  { kind: "overview", label: "Overview", href: "/operator", testId: "operator-nav-overview" },
  { kind: "invites", label: "Invites", href: "/operator/invites", testId: "operator-nav-invites" },
  { kind: "games", label: "Games", href: "/operator/games", testId: "operator-nav-games" },
  { kind: "submissions", label: "Submissions", href: "/operator/submissions", testId: "operator-nav-submissions" },
  { kind: "requests", label: "Requests", href: "/operator/requests", testId: "operator-nav-requests" },
  { kind: "rankings", label: "Rankings", href: "/operator/rankings", testId: "operator-nav-rankings" },
] as const;

export function parseOperatorRoute(pathname: string): OperatorRoute | undefined {
  const normalized = normalizePathname(pathname);
  if (normalized === "/" || normalized === "/operator") {
    return { kind: "overview" };
  }
  if (normalized === "/operator/invites") {
    return { kind: "invites" };
  }
  if (normalized === "/operator/games") {
    return { kind: "games" };
  }
  if (normalized === "/operator/submissions") {
    return { kind: "submissions" };
  }
  if (normalized === "/operator/requests") {
    return { kind: "requests" };
  }
  if (normalized === "/operator/rankings") {
    return { kind: "rankings" };
  }
  const runMatch = normalized.match(/^\/operator\/runs\/([^/]+)$/);
  if (runMatch) {
    return { kind: "run-detail", runId: decodeURIComponent(runMatch[1]) };
  }
  return undefined;
}

export function operatorRouteTitle(route: OperatorRoute) {
  switch (route.kind) {
    case "overview":
      return "Overview";
    case "invites":
      return "Invites";
    case "games":
      return "Games";
    case "submissions":
      return "Submissions";
    case "requests":
      return "Requests";
    case "rankings":
      return "Rankings";
    case "run-detail":
      return "Run Detail";
  }
}

export function hrefForRunDetail(runId: string) {
  return `/operator/runs/${encodeURIComponent(runId)}`;
}

function normalizePathname(pathname: string) {
  return pathname.endsWith("/") && pathname !== "/" ? pathname.slice(0, -1) : pathname;
}
