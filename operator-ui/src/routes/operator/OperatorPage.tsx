import { Panel } from "../../shared/ui/Panel";
import { hintFor } from "./operatorPageSupport";
import { CompletedDetailPanel } from "./CompletedDetailPanel";
import { MatchTable } from "./MatchTable";
import { OperatorHeader } from "./OperatorHeader";
import { PresetQueuePanel } from "./PresetQueuePanel";
import { useOperatorPageState } from "./useOperatorPageState";

export function OperatorPage() {
  const state = useOperatorPageState();

  return (
    <>
      <OperatorHeader baseUrl={state.baseUrl} onBaseUrlChange={state.setBaseUrl} />

      <section className="grid gap-6 lg:grid-cols-[1.1fr_1fr]">
        <PresetQueuePanel
          enqueueState={state.enqueueState}
          enqueueError={state.enqueueError}
          onEnqueue={(presetId) => void state.enqueuePreset(presetId)}
        />

        <CompletedDetailPanel
          detail={state.detail}
          detailState={state.detailState}
          detailError={state.detailError}
          onRefreshDetail={state.reloadDetail}
        />
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <Panel
          title="Active Matches"
          subtitle="Polled every 5 seconds."
          status={state.activeState}
          error={state.activeError}
          hint={hintFor(state.activeError)}
          testId="operator-panel-active-matches"
        >
          <MatchTable
            items={state.activeItems}
            emptyMessage="No active submissions are currently queued or running."
            onSelect={(item) => state.setSelectedRunId(item.run_id)}
            selectedRunId={state.selectedRunId}
          />
        </Panel>

        <Panel
          title="Completed Matches"
          subtitle="Polled every 10 seconds."
          status={state.completedState}
          error={state.completedError}
          hint={hintFor(state.completedError)}
          testId="operator-panel-completed-matches"
        >
          <MatchTable
            items={state.completedItems}
            emptyMessage="No completed submissions yet."
            onSelect={(item) => state.setSelectedRunId(item.run_id)}
            selectedRunId={state.selectedRunId}
          />
        </Panel>
      </section>
    </>
  );
}
