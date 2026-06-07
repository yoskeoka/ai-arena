import { ReactNode } from "react";

export function Badge({ children, tone = "accent" }: { children: ReactNode; tone?: "accent" | "teal" | "moss" }) {
  const color =
    tone === "teal"
      ? "bg-teal/10 text-teal"
      : tone === "moss"
        ? "bg-moss/10 text-moss"
        : "bg-accent/10 text-accent";
  return <span className={`rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-wide ${color}`}>{children}</span>;
}
