import { ResultListItem } from "../../api";
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
        const selected = item.run_id === selectedRunId;
        return (
          <button
            key={item.run_id}
            type="button"
            onClick={() => onSelect(item)}
            data-testid={`match-row-${item.run_id}`}
            className={`w-full rounded-3xl border p-4 text-left shadow-sm transition ${
              selected ? "border-accent bg-accent/10" : "border-black/10 bg-paper hover:border-black/25"
            }`}
          >
            <div className="flex flex-wrap items-center gap-2">
              <Badge>service: {item.lifecycle_state}</Badge>
              {item.terminal_status ? <Badge tone="teal">match: {item.terminal_status}</Badge> : null}
              {item.official ? <Badge tone="moss">official</Badge> : null}
            </div>
            <div className="mt-3 flex flex-col gap-1">
              <p className="font-semibold">{item.match_id}</p>
              <p className="text-xs text-black/60">{item.run_id}</p>
              <p className="text-sm text-black/70">
                {item.game_id}@{item.game_version} / {item.ruleset_version}
              </p>
            </div>
            <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
              <span>attempt: {item.attempt_count}</span>
              <span>turn: {typeof item.turn === "number" ? item.turn : "n/a"}</span>
              <span>worker: {item.worker_id || "n/a"}</span>
            </div>
            {item.error ? <p className="mt-3 text-sm text-red-700">{item.error}</p> : null}
          </button>
        );
      })}
    </div>
  );
}
