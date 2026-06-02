import { useEffect, useMemo, useState } from "react";

import { MatchDetailResponse, OperatorApiClient, ResultListItem } from "./api";
import { presetCatalog } from "./presets";

const ACTIVE_POLL_MS = 5_000;
const COMPLETED_POLL_MS = 10_000;
const DETAIL_POLL_MS = 15_000;

type LoadState = "idle" | "loading" | "ready" | "error";
type EnqueueState = "idle" | "submitting" | "success" | "error";

export default function App() {
  const [baseUrl, setBaseUrl] = useState(() => defaultBaseUrl());
  const client = useMemo(() => new OperatorApiClient(baseUrl), [baseUrl]);

  const [activeItems, setActiveItems] = useState<ResultListItem[]>([]);
  const [completedItems, setCompletedItems] = useState<ResultListItem[]>([]);
  const [activeState, setActiveState] = useState<LoadState>("loading");
  const [completedState, setCompletedState] = useState<LoadState>("loading");
  const [detailState, setDetailState] = useState<LoadState>("idle");
  const [enqueueState, setEnqueueState] = useState<EnqueueState>("idle");
  const [activeError, setActiveError] = useState<string>();
  const [completedError, setCompletedError] = useState<string>();
  const [detailError, setDetailError] = useState<string>();
  const [enqueueError, setEnqueueError] = useState<string>();
  const [selectedSubmissionId, setSelectedSubmissionId] = useState<string>();
  const [detail, setDetail] = useState<MatchDetailResponse>();
  const [detailReloadToken, setDetailReloadToken] = useState(0);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      setActiveState((current) => (current === "ready" ? current : "loading"));
      try {
        const items = await client.listActive();
        if (canceled) {
          return;
        }
        setActiveItems(items);
        setActiveState("ready");
        setActiveError(undefined);
      } catch (error) {
        if (canceled) {
          return;
        }
        setActiveState("error");
        setActiveError(messageOf(error));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, ACTIVE_POLL_MS);
    return () => {
      canceled = true;
      window.clearInterval(timer);
    };
  }, [client]);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      setCompletedState((current) => (current === "ready" ? current : "loading"));
      try {
        const items = await client.listCompleted();
        if (canceled) {
          return;
        }
        setCompletedItems(items);
        setCompletedState("ready");
        setCompletedError(undefined);
        setSelectedSubmissionId((current) => {
          if (current && items.some((item) => item.submission_id === current)) {
            return current;
          }
          return items[0]?.submission_id;
        });
      } catch (error) {
        if (canceled) {
          return;
        }
        setCompletedState("error");
        setCompletedError(messageOf(error));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, COMPLETED_POLL_MS);
    return () => {
      canceled = true;
      window.clearInterval(timer);
    };
  }, [client]);

  useEffect(() => {
    if (!selectedSubmissionId) {
      setDetail(undefined);
      setDetailState("idle");
      setDetailError(undefined);
      return;
    }

    let canceled = false;

    const load = async () => {
      setDetailState((current) => (current === "ready" ? current : "loading"));
      try {
        const response = await client.getMatchDetail(selectedSubmissionId);
        if (canceled) {
          return;
        }
        setDetail(response);
        setDetailState("ready");
        setDetailError(undefined);
      } catch (error) {
        if (canceled) {
          return;
        }
        setDetailState("error");
        setDetailError(messageOf(error));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, DETAIL_POLL_MS);
    return () => {
      canceled = true;
      window.clearInterval(timer);
    };
  }, [client, detailReloadToken, selectedSubmissionId]);

  const handleEnqueue = async (presetId: string) => {
    setEnqueueState("submitting");
    setEnqueueError(undefined);
    try {
      await client.enqueuePreset(presetId);
      setEnqueueState("success");
      const items = await client.listActive();
      setActiveItems(items);
      setActiveState("ready");
      setActiveError(undefined);
    } catch (error) {
      setEnqueueState("error");
      setEnqueueError(messageOf(error));
    }
  };

  const artifactEntries = Object.entries(detail?.artifact_access ?? {});

  return (
    <div className="min-h-screen bg-paper text-ink">
      <main className="mx-auto flex max-w-7xl flex-col gap-6 px-4 py-6 md:px-6 lg:px-8">
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
                onChange={(event) => setBaseUrl(event.target.value)}
                placeholder="https://ai-arena-service.onrender.com"
              />
            </label>
          </div>
        </header>

        <section className="grid gap-6 lg:grid-cols-[1.1fr_1fr]">
          <Panel
            title="Preset Queue"
            subtitle="One-click enqueue against server-known presets."
            status={enqueueState}
            error={enqueueError}
          >
            <div className="grid gap-3">
              {presetCatalog.map((preset) => (
                <button
                  key={preset.presetId}
                  type="button"
                  className="rounded-3xl border border-black/10 bg-white p-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-accent hover:shadow-md disabled:cursor-wait disabled:opacity-70"
                  onClick={() => void handleEnqueue(preset.presetId)}
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

          <Panel title="Completed Detail" subtitle="Result-summary first, delegated artifacts second." status={detailState} error={detailError}>
            {detail ? (
              <div className="space-y-5">
                <div className="rounded-3xl bg-ink p-5 text-paper">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge>{detail.lifecycle_state}</Badge>
                    {detail.terminal_status ? <Badge tone="teal">{detail.terminal_status}</Badge> : null}
                  </div>
                  <h2 className="mt-4 text-xl font-semibold">{detail.match_id}</h2>
                  <p className="mt-1 text-sm text-paper/70">{detail.submission_id}</p>
                  <dl className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                    <Meta label="Game" value={`${detail.game_id}@${detail.game_version}`} />
                    <Meta label="Ruleset" value={detail.ruleset_version} />
                    <Meta label="Output Dir" value={detail.output_dir} />
                    <Meta label="Result Summary" value={detail.result_summary_path ?? "n/a"} />
                  </dl>
                </div>

                <div className="grid gap-4 xl:grid-cols-2">
                  <section className="rounded-3xl border border-black/10 bg-white p-4">
                    <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-teal">Summary</h3>
                    {detail.result_summary ? (
                      <div className="mt-3 space-y-3 text-sm">
                        <Meta label="Status" value={detail.result_summary.status} />
                        <Meta label="Turn" value={String(detail.result_summary.turn)} />
                        <Meta label="Error" value={detail.result_summary.error ?? "none"} />
                        <div>
                          <p className="font-medium">Placements</p>
                          <ul className="mt-2 space-y-2">
                            {(detail.result_summary.placements ?? []).map((placement) => (
                              <li key={placement.player_id} className="rounded-2xl bg-paper px-3 py-2">
                                {placement.rank}. {placement.player_id}
                                {typeof placement.score === "number" ? ` (${placement.score})` : ""}
                              </li>
                            ))}
                          </ul>
                        </div>
                      </div>
                    ) : (
                      <p className="mt-3 text-sm text-black/60">No decoded result summary available.</p>
                    )}
                  </section>

                  <section className="rounded-3xl border border-black/10 bg-white p-4">
                    <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-teal">Players</h3>
                    <ul className="mt-3 space-y-2 text-sm">
                      {detail.players.map((player) => (
                        <li key={player.player_id} className="rounded-2xl bg-paper px-3 py-2">
                          <p className="font-medium">{player.player_id}</p>
                          <p className="mt-1 break-all text-black/65">{player.artifact_ref}</p>
                        </li>
                      ))}
                    </ul>
                  </section>
                </div>

                <section className="rounded-3xl border border-black/10 bg-white p-4">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-teal">Artifacts</h3>
                    <button
                      type="button"
                      className="rounded-full border border-black/15 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-black/70 transition hover:border-accent hover:text-accent"
                      onClick={() => setDetailReloadToken((current) => current + 1)}
                    >
                      refresh detail
                    </button>
                  </div>
                  <div className="mt-3 space-y-3">
                    {artifactEntries.length === 0 ? (
                      <p className="text-sm text-black/60">No artifact access metadata returned yet.</p>
                    ) : (
                      artifactEntries.map(([kind, artifact]) => (
                        <article key={kind} className="rounded-2xl bg-paper p-3 text-sm">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-semibold">{kind}</span>
                            <Badge tone={artifact.download_url ? "teal" : "moss"}>{artifact.status ?? "unknown"}</Badge>
                          </div>
                          <p className="mt-2 break-all text-black/70">{artifact.locator}</p>
                          <div className="mt-2 flex flex-wrap gap-4 text-xs text-black/60">
                            <span>issuer: {artifact.issuer ?? "n/a"}</span>
                            <span>expiry: {artifact.expires_at ?? "n/a"}</span>
                          </div>
                          {artifact.download_url ? (
                            <a
                              className="mt-3 inline-flex rounded-full bg-teal px-3 py-2 text-xs font-semibold uppercase tracking-wide text-white no-underline transition hover:bg-ink"
                              href={artifact.download_url}
                              target="_blank"
                              rel="noreferrer"
                            >
                              open delegated download
                            </a>
                          ) : null}
                        </article>
                      ))
                    )}
                  </div>
                </section>

                <section className="rounded-3xl border border-black/10 bg-white p-4">
                  <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-teal">Replay Inputs</h3>
                  <dl className="mt-3 grid gap-3 text-sm md:grid-cols-2">
                    <Meta label="Record" value={detail.replay_inputs?.record_path ?? detail.record_path ?? "n/a"} />
                    <Meta label="Snapshot" value={detail.replay_inputs?.snapshot_path ?? "n/a"} />
                    <Meta label="History" value={detail.replay_inputs?.history_path ?? "n/a"} />
                    <Meta label="Exported Snapshot" value={detail.replay_inputs?.exported_snapshot_path ?? "n/a"} />
                  </dl>
                  {detail.replay_inputs?.verification?.issues?.length ? (
                    <div className="mt-4 rounded-2xl border border-accent/30 bg-accent/10 p-3 text-sm text-black/80">
                      <p className="font-medium">Verification issues</p>
                      <ul className="mt-2 space-y-1">
                        {detail.replay_inputs.verification.issues.map((issue) => (
                          <li key={issue}>{issue}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                </section>
              </div>
            ) : (
              <EmptyDetailState />
            )}
          </Panel>
        </section>

        <section className="grid gap-6 xl:grid-cols-2">
          <Panel title="Active Matches" subtitle="Polled every 5 seconds." status={activeState} error={activeError}>
            <MatchTable
              items={activeItems}
              emptyMessage="No active submissions are currently queued or running."
              onSelect={(item) => setSelectedSubmissionId(item.submission_id)}
              selectedSubmissionId={selectedSubmissionId}
            />
          </Panel>

          <Panel title="Completed Matches" subtitle="Polled every 10 seconds." status={completedState} error={completedError}>
            <MatchTable
              items={completedItems}
              emptyMessage="No completed submissions yet."
              onSelect={(item) => setSelectedSubmissionId(item.submission_id)}
              selectedSubmissionId={selectedSubmissionId}
            />
          </Panel>
        </section>
      </main>
    </div>
  );
}

function Panel({
  title,
  subtitle,
  status,
  error,
  children,
}: {
  title: string;
  subtitle: string;
  status: LoadState | EnqueueState;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-[28px] border border-black/10 bg-white/75 p-5 shadow-sm backdrop-blur">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">{title}</h2>
          <p className="mt-1 text-sm text-black/65">{subtitle}</p>
        </div>
        <Badge tone={status === "error" ? "accent" : "moss"}>{status}</Badge>
      </div>
      {error ? <p className="mt-3 rounded-2xl bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p> : null}
      <div className="mt-4">{children}</div>
    </section>
  );
}

function MatchTable({
  items,
  emptyMessage,
  onSelect,
  selectedSubmissionId,
}: {
  items: ResultListItem[];
  emptyMessage: string;
  onSelect: (item: ResultListItem) => void;
  selectedSubmissionId?: string;
}) {
  if (items.length === 0) {
    return <p className="text-sm text-black/60">{emptyMessage}</p>;
  }

  return (
    <div className="space-y-3">
      {items.map((item) => {
        const selected = item.submission_id === selectedSubmissionId;
        return (
          <button
            key={item.submission_id}
            type="button"
            onClick={() => onSelect(item)}
            className={`w-full rounded-3xl border p-4 text-left shadow-sm transition ${
              selected ? "border-accent bg-accent/10" : "border-black/10 bg-paper hover:border-black/25"
            }`}
          >
            <div className="flex flex-wrap items-center gap-2">
              <Badge>{item.lifecycle_state}</Badge>
              {item.terminal_status ? <Badge tone="teal">{item.terminal_status}</Badge> : null}
            </div>
            <div className="mt-3 flex flex-col gap-1">
              <p className="font-semibold">{item.match_id}</p>
              <p className="text-xs text-black/60">{item.submission_id}</p>
              <p className="text-sm text-black/70">
                {item.game_id}@{item.game_version} / {item.ruleset_version}
              </p>
            </div>
            <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
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

function EmptyDetailState() {
  return (
    <div className="rounded-3xl border border-dashed border-black/15 bg-paper p-8 text-center text-sm text-black/60">
      Select a completed submission to inspect result-summary, replay inputs, and delegated artifact links.
    </div>
  );
}

function Badge({ children, tone = "accent" }: { children: React.ReactNode; tone?: "accent" | "teal" | "moss" }) {
  const color =
    tone === "teal"
      ? "bg-teal/10 text-teal"
      : tone === "moss"
        ? "bg-moss/10 text-moss"
        : "bg-accent/10 text-accent";
  return <span className={`rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-wide ${color}`}>{children}</span>;
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="font-medium text-black/60">{label}</dt>
      <dd className="mt-1 break-all text-black/85">{value}</dd>
    </div>
  );
}

function defaultBaseUrl() {
  const envValue = import.meta.env.VITE_OPERATOR_API_BASE_URL;
  if (typeof envValue === "string" && envValue.trim() !== "") {
    return envValue;
  }
  if (typeof window !== "undefined" && window.location.hostname === "localhost") {
    return "http://127.0.0.1:10000";
  }
  return "";
}

function messageOf(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}
