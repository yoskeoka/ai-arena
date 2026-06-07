import { ReactNode } from "react";

import { Badge } from "./Badge";

type PanelStatus = "idle" | "loading" | "ready" | "error" | "submitting" | "success";

export function Panel({
  title,
  subtitle,
  status,
  error,
  hint,
  children,
  testId,
}: {
  title: string;
  subtitle: string;
  status: PanelStatus;
  error?: string;
  hint?: string;
  children: ReactNode;
  testId: string;
}) {
  return (
    <section className="rounded-[28px] border border-black/10 bg-white/75 p-5 shadow-sm backdrop-blur" data-testid={testId}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">{title}</h2>
          <p className="mt-1 text-sm text-black/65">{subtitle}</p>
        </div>
        <Badge tone={status === "error" ? "accent" : "moss"}>{status}</Badge>
      </div>
      {error ? (
        <div className="mt-3 rounded-2xl bg-red-50 px-3 py-2 text-sm text-red-700">
          <p>{error}</p>
          {hint ? <p className="mt-1 text-red-700/80">{hint}</p> : null}
        </div>
      ) : null}
      <div className="mt-4">{children}</div>
    </section>
  );
}
