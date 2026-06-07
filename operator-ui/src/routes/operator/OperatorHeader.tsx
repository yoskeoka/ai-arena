type OperatorHeaderProps = {
  baseUrl: string;
  onBaseUrlChange: (value: string) => void;
};

export function OperatorHeader({ baseUrl, onBaseUrlChange }: OperatorHeaderProps) {
  return (
    <header className="rounded-[28px] border border-black/10 bg-white/80 p-6 shadow-sm backdrop-blur">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="space-y-2">
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-teal">Phase 6 Operator Surface</p>
          <h1 className="text-3xl font-semibold tracking-tight">AI Arena Minimal Operator UI</h1>
          <p className="max-w-3xl text-sm text-black/70">
            Active and completed match polling, preset queue actions, and delegated artifact access for the first
            online confirmation lane.
          </p>
        </div>
        <label className="flex min-w-80 flex-col gap-2 text-sm">
          <span className="font-medium text-black/70">Operator API base URL</span>
          <input
            className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm outline-none transition focus:border-accent"
            value={baseUrl}
            onChange={(event) => onBaseUrlChange(event.target.value)}
            placeholder="Leave blank for local Vite proxy, or set https://ai-arena-service.onrender.com"
          />
        </label>
      </div>
    </header>
  );
}
