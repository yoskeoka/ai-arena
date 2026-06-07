import { presetCatalog } from "../../presets";
import { hintFor } from "./operatorPageSupport";
import { EnqueueState } from "./operatorPageSupport";
import { Panel } from "../../shared/ui/Panel";

type PresetQueuePanelProps = {
  enqueueState: EnqueueState;
  enqueueError?: string;
  onEnqueue: (presetId: string) => void;
};

export function PresetQueuePanel({ enqueueState, enqueueError, onEnqueue }: PresetQueuePanelProps) {
  return (
    <Panel
      title="Preset Queue"
      subtitle="One-click enqueue against server-known presets."
      status={enqueueState}
      error={enqueueError}
      hint={hintFor(enqueueError)}
      testId="operator-panel-preset-queue"
    >
      <div className="grid gap-3">
        {presetCatalog.map((preset) => (
          <button
            key={preset.presetId}
            type="button"
            data-testid={`preset-queue-action-${preset.presetId}`}
            className="rounded-3xl border border-black/10 bg-white p-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-accent hover:shadow-md disabled:cursor-wait disabled:opacity-70"
            onClick={() => onEnqueue(preset.presetId)}
            disabled={enqueueState === "submitting"}
          >
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="font-semibold">{preset.title}</h2>
                <p className="mt-1 text-sm text-black/70">{preset.description}</p>
              </div>
              <span className="rounded-full bg-accent px-3 py-1 text-xs font-semibold uppercase tracking-wide text-white">
                queue
              </span>
            </div>
          </button>
        ))}
      </div>
    </Panel>
  );
}
