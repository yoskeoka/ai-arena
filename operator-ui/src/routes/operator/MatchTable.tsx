import { ResultListItem } from "../../lib/operatorApiClient";
import { Badge } from "../../shared/ui/Badge";

export function MatchTable({
  items,
  emptyMessage,
  onSelect,
  selectedRunId,
}: {
  items: ResultListItem[];
  emptyMessage: string;
  onSelect: (item: ResultListItem) => void;
  selectedRunId?: string;
}) {
  if (items.length === 0) {
    return <p className="text-sm text-black/60">{emptyMessage}</p>;
  }

  return (
    <div className="space-y-3">
      {items.map((item) => {
        const selected = item.runId === selectedRunId;
        return (
          <button
            key={item.runId}
            type="button"
            onClick={() => onSelect(item)}
            data-testid={`match-row-${item.runId}`}
            className={`w-full rounded-3xl border p-4 text-left shadow-sm transition ${
              selected ? "border-accent bg-accent/10" : "border-black/10 bg-paper hover:border-black/25"
            }`}
          >
            <div className="flex flex-wrap items-center gap-2">
              <Badge>service: {item.lifecycleState}</Badge>
              {item.terminalStatus ? <Badge tone="teal">match: {item.terminalStatus}</Badge> : null}
              {item.official ? <Badge tone="moss">official</Badge> : null}
            </div>
            <div className="mt-3 flex flex-col gap-1">
              <p className="font-semibold">{item.matchId}</p>
              <p className="text-xs text-black/60">{item.runId}</p>
              <p className="text-sm text-black/70">
                {item.gameId}@{item.gameVersion} / {item.rulesetVersion}
              </p>
            </div>
            <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
              <span>attempt: {item.attemptCount}</span>
              <span>turn: {typeof item.turn === "number" ? item.turn : "n/a"}</span>
              <span>worker: {item.workerId || "n/a"}</span>
            </div>
            {item.error ? <p className="mt-3 text-sm text-red-700">{item.error}</p> : null}
          </button>
        );
      })}
    </div>
  );
}
