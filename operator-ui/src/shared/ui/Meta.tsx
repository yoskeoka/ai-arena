type MetaProps = {
  label: string;
  value: string;
  variant?: "default" | "dark";
};

export function Meta({ label, value, variant = "default" }: MetaProps) {
  const dark = variant === "dark";

  return (
    <div>
      <dt className={dark ? "font-medium text-paper/70" : "font-medium text-black/60"}>{label}</dt>
      <dd className={dark ? "mt-1 break-all text-paper" : "mt-1 break-all text-black/85"}>{value}</dd>
    </div>
  );
}
