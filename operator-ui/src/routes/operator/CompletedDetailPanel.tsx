import { MatchDetailResponse } from "../../api";
import { Badge } from "../../shared/ui/Badge";
import { Meta } from "../../shared/ui/Meta";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState } from "./operatorPageSupport";

type CompletedDetailPanelProps = {
  detail?: MatchDetailResponse;
  detailState: LoadState;
  detailError?: string;
  onRefreshDetail: () => void;
};

export function CompletedDetailPanel({
  detail,
  detailState,
  detailError,
  onRefreshDetail,
}: CompletedDetailPanelProps) {
  const artifactEntries = Object.entries(detail?.artifact_access ?? {});

  return (
    <Panel
      title="Completed Detail"
      subtitle="Result-summary first, delegated artifacts second."
      status={detailState}
      error={detailError}
      hint={hintFor(detailError)}
      testId="operator-panel-completed-detail"
    >
      {detail ? (
        <div className="space-y-5" data-testid={`match-detail-${detail.run_id}`}>
          <div className="rounded-3xl bg-ink p-5 text-paper">
            <div className="flex flex-wrap items-center gap-2">
              <Badge>service: {detail.lifecycle_state}</Badge>
              {detail.terminal_status ? <Badge tone="teal">match: {detail.terminal_status}</Badge> : null}
              {detail.official ? <Badge tone="moss">official</Badge> : null}
            </div>
            <h2 className="mt-4 text-xl font-semibold">{detail.match_id}</h2>
            <p className="mt-1 text-sm text-paper/70">{detail.run_id}</p>
            <dl className="mt-4 grid gap-3 text-sm md:grid-cols-2">
              <Meta label="Attempt" value={String(detail.attempt_count)} />
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
                  <dl className="grid gap-3">
                    <Meta label="Status" value={detail.result_summary.status} />
                    <Meta label="Turn" value={String(detail.result_summary.turn)} />
                    <Meta label="Error" value={detail.result_summary.error ?? "none"} />
                  </dl>
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
                <div className="mt-3 space-y-2 text-sm text-black/60">
                  <p>No decoded result summary available.</p>
                  <p>Use the artifact entries below to inspect the persisted locator directly.</p>
                </div>
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
                onClick={onRefreshDetail}
              >
                refresh detail
              </button>
            </div>
            <div className="mt-3 space-y-3">
              {artifactEntries.length === 0 ? (
                <p className="text-sm text-black/60">No artifact access metadata returned yet.</p>
              ) : (
                artifactEntries.map(([kind, artifact]) => (
                  <article key={kind} className="rounded-2xl bg-paper p-3 text-sm" data-testid={`artifact-entry-${kind}`}>
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
                        rel="noopener noreferrer"
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
  );
}

function EmptyDetailState() {
  return (
    <div className="rounded-3xl border border-dashed border-black/15 bg-paper p-8 text-center text-sm text-black/60">
      Select a completed run to inspect result-summary, replay inputs, and delegated artifact links.
    </div>
  );
}
