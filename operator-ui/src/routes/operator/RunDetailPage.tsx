import { useEffect, useMemo, useState } from "react";

import { MatchDetailResponse, OperatorApiClient } from "../../api";
import { CompletedDetailPanel } from "./CompletedDetailPanel";
import { EnqueueState, isAbortError, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

type RunDetailPageProps = {
  baseUrl: string;
  runId: string;
};

export function RunDetailPage({ baseUrl, runId }: RunDetailPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [detail, setDetail] = useState<MatchDetailResponse>();
  const [detailState, setDetailState] = useState<LoadState>("loading");
  const [detailError, setDetailError] = useState<string>();
  const [actionState, setActionState] = useState<EnqueueState>("idle");
  const [actionError, setActionError] = useState<string>();
  const [reloadToken, setReloadToken] = useState(0);

  const load = async (signal?: AbortSignal) => {
    setDetailState((current) => (current === "ready" ? current : "loading"));
    try {
      const response = await client.getMatchDetail(runId, signal);
      setDetail(response);
      setDetailState("ready");
      setDetailError(undefined);
    } catch (error) {
      if (isAbortError(error)) {
        return;
      }
      setDetail(undefined);
      setDetailState("error");
      setDetailError(messageOf(error));
    }
  };

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [client, reloadToken, runId]);

  const runAction = async (action: "cancel" | "retry" | "rerun" | "promote") => {
    setActionState("submitting");
    setActionError(undefined);
    try {
      switch (action) {
        case "cancel":
          await client.cancelRun(runId);
          break;
        case "retry":
          await client.retryRun(runId);
          break;
        case "rerun":
          await client.rerunRun(runId);
          break;
        case "promote":
          await client.promoteRun(runId);
          break;
      }
      setActionState("success");
      setReloadToken((current) => current + 1);
    } catch (error) {
      setActionState("error");
      setActionError(messageOf(error));
    }
  };

  return (
    <CompletedDetailPanel
      detail={detail}
      detailState={detailState}
      detailError={detailError}
      onRefreshDetail={() => setReloadToken((current) => current + 1)}
      title="Run Detail"
      subtitle="Follow-up actions stay close to the compact summary and artifacts."
      testId="operator-panel-run-detail"
      emptyMessage="The requested run detail is not available yet."
      actions={
        <>
          {detail ? <RunActionButtons detail={detail} actionState={actionState} onAction={runAction} /> : null}
          {actionError ? <p className="w-full text-sm text-red-200">{actionError}</p> : null}
        </>
      }
    />
  );
}

function RunActionButtons({
  detail,
  actionState,
  onAction,
}: {
  detail: MatchDetailResponse;
  actionState: EnqueueState;
  onAction: (action: "cancel" | "retry" | "rerun" | "promote") => void;
}) {
  const actions: Array<{ kind: "cancel" | "retry" | "rerun" | "promote"; label: string }> = [];
  if (detail.lifecycle_state === "queued") {
    actions.push({ kind: "cancel", label: "Cancel queued run" });
  }
  if (detail.lifecycle_state === "failed") {
    actions.push({ kind: "retry", label: "Retry failed run" });
  }
  if (detail.lifecycle_state === "completed") {
    actions.push({ kind: "rerun", label: "Create rerun candidate" });
    if (!detail.official) {
      actions.push({ kind: "promote", label: "Promote as official" });
    }
  }
  if (actions.length === 0) {
    return <p className="text-sm text-paper/70">No run follow-up action is available for the current lifecycle.</p>;
  }
  return (
    <>
      {actions.map((action) => (
        <ActionButton
          key={action.kind}
          kind={action.kind}
          label={action.label}
          disabled={actionState === "submitting"}
          onClick={() => onAction(action.kind)}
        />
      ))}
      {actionState === "success" ? <p className="text-sm text-paper/70">Run action accepted. Refreshing detail…</p> : null}
    </>
  );
}

function ActionButton({
  kind,
  label,
  disabled,
  onClick,
}: {
  kind: string;
  label: string;
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      data-testid={`run-action-${kind}`}
      className="rounded-full border border-white/20 bg-white/10 px-4 py-2 text-sm font-semibold text-paper transition hover:bg-white/20 disabled:cursor-not-allowed disabled:opacity-60"
      disabled={disabled}
      onClick={onClick}
    >
      {label}
    </button>
  );
}
