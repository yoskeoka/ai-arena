import { ReactNode } from "react";

import { Panel } from "../../shared/ui/Panel";
import { hintFor } from "./operatorPageSupport";
import { CompletedDetailPanel } from "./CompletedDetailPanel";
import { MatchTable } from "./MatchTable";
import { useOperatorPageState } from "./useOperatorPageState";

type OperatorPageProps = {
  baseUrl: string;
  detailActions?: ReactNode;
};

export function OperatorPage({ baseUrl, detailActions }: OperatorPageProps) {
  const state = useOperatorPageState(baseUrl);

  return (
    <>
      <section className="grid gap-6">
        <CompletedDetailPanel
          detail={state.detail}
          detailState={state.detailState}
          detailError={state.detailError}
          onRefreshDetail={state.reloadDetail}
          actions={detailActions}
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
            emptyMessage="No active runs are currently queued or running."
            onSelect={(item) => state.setSelectedRunId(item.runId)}
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
            emptyMessage="No completed runs yet."
            onSelect={(item) => state.setSelectedRunId(item.runId)}
            selectedRunId={state.selectedRunId}
          />
        </Panel>
      </section>
    </>
  );
}
