export function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="font-medium text-black/60">{label}</dt>
      <dd className="mt-1 break-all text-black/85">{value}</dd>
    </div>
  );
}
