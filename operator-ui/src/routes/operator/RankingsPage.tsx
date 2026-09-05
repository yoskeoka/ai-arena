import { FormEvent, useEffect, useMemo, useState } from "react";

import { OperatorApiClient, RankingScope, ResultListItem, StoredRankingSnapshot } from "../../lib/operatorApiClient";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

type RankingsPageProps = {
  baseUrl: string;
};

export function RankingsPage({ baseUrl }: RankingsPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [completedItems, setCompletedItems] = useState<ResultListItem[]>([]);
  const [scopeListState, setScopeListState] = useState<LoadState>("loading");
  const [scopeListError, setScopeListError] = useState<string>();
  const [scope, setScope] = useState<RankingScope>({ gameId: "", gameVersion: "", rulesetVersion: "" });
  const [snapshot, setSnapshot] = useState<StoredRankingSnapshot>();
  const [snapshotState, setSnapshotState] = useState<LoadState>("idle");
  const [snapshotError, setSnapshotError] = useState<string>();

  const loadCompleted = async () => {
    setScopeListState((current) => (current === "ready" ? current : "loading"));
    try {
      const items = await client.listCompletedMatches();
      setCompletedItems(items);
      setScopeListState("ready");
      setScopeListError(undefined);
      const firstOfficial = items.find((item) => item.lifecycleState === "completed" && item.official);
      if (firstOfficial) {
        setScope((current) =>
          current.gameId || current.gameVersion || current.rulesetVersion
            ? current
            : {
                gameId: firstOfficial.gameId,
                gameVersion: firstOfficial.gameVersion,
                rulesetVersion: firstOfficial.rulesetVersion,
              },
        );
      }
    } catch (error) {
      setScopeListState("error");
      setScopeListError(messageOf(error));
    }
  };

  useEffect(() => {
    void loadCompleted();
  }, [client]);

  const loadSnapshot = async (nextScope: RankingScope) => {
    setSnapshotState("loading");
    setSnapshotError(undefined);
    try {
      const response = await client.getRanking(nextScope);
      setSnapshot(response);
      setSnapshotState("ready");
    } catch (error) {
      setSnapshot(undefined);
      setSnapshotState("error");
      setSnapshotError(messageOf(error));
    }
  };

  const quickScopes = dedupeScopes(completedItems.filter((item) => item.lifecycleState === "completed" && item.official));

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await loadSnapshot(scope);
  };

  return (
    <section className="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
      <Panel
        title="Ranking Scope"
        subtitle="Read one durable snapshot by game scope."
        status={scopeListState}
        error={scopeListError}
        hint={hintFor(scopeListError)}
        testId="operator-form-rankings"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <TextField label="Game ID" value={scope.gameId} onChange={(value) => setScope((current) => ({ ...current, gameId: value }))} required />
          <TextField
            label="Game Version"
            value={scope.gameVersion}
            onChange={(value) => setScope((current) => ({ ...current, gameVersion: value }))}
            required
          />
          <TextField
            label="Ruleset Version"
            value={scope.rulesetVersion}
            onChange={(value) => setScope((current) => ({ ...current, rulesetVersion: value }))}
            required
          />
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
            Load ranking snapshot
          </button>
        </form>

        {quickScopes.length ? (
          <div className="mt-5 space-y-2">
            <p className="text-sm font-medium text-black/70">Quick scopes from official completed runs</p>
            <div className="flex flex-wrap gap-2">
              {quickScopes.map((item) => {
                const scopeId = scopeKey(item);
                return (
                  <button
                    key={scopeId}
                    type="button"
                    data-testid={`ranking-scope-${scopeId}`}
                    className="rounded-full border border-black/15 bg-paper px-3 py-2 text-xs font-semibold uppercase tracking-wide text-black/70 transition hover:border-accent hover:text-accent"
                    onClick={() => {
                      setScope(item);
                      void loadSnapshot(item);
                    }}
                  >
                    {item.gameId}@{item.gameVersion}
                  </button>
                );
              })}
            </div>
          </div>
        ) : null}
      </Panel>

      <Panel
        title="Ranking Snapshot"
        subtitle="Current durable aggregate for the selected scope."
        status={snapshotState}
        error={snapshotError}
        hint={hintFor(snapshotError)}
        testId="operator-panel-rankings"
      >
        {snapshot ? (
          <div className="space-y-4">
            <div className="rounded-3xl bg-paper p-4 text-sm">
              <p className="font-semibold">{snapshot.snapshot.scope.gameId}</p>
              <p className="mt-1 text-black/70">
                {snapshot.snapshot.scope.gameVersion} / {snapshot.snapshot.scope.rulesetVersion}
              </p>
              <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                <span>locator: {snapshot.locator}</span>
                <span>completed matches: {snapshot.snapshot.completedMatches}</span>
                <span>last applied run: {snapshot.snapshot.lastAppliedRunId || "n/a"}</span>
              </div>
            </div>
            {snapshot.snapshot.entries?.length ? (
              <div className="space-y-3">
                {snapshot.snapshot.entries.map((entry) => (
                  <article
                    key={entry.botId}
                    className="rounded-3xl border border-black/10 bg-white p-4"
                    data-testid={`ranking-entry-${encodeURIComponent(entry.botId)}`}
                  >
                    <p className="font-semibold">{entry.botName}</p>
                    <p className="mt-1 text-xs text-black/60">bot: {entry.botId}</p>
                    <p className="mt-1 text-sm text-black/70">last player: {entry.lastPlayerId}</p>
                    <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                      <span>matches: {entry.matchesPlayed}</span>
                      <span>first places: {entry.firstPlaces}</span>
                      <span>last run: {entry.lastRunId}</span>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <p className="text-sm text-black/60">No ranking entries stored for this scope yet.</p>
            )}
          </div>
        ) : (
          <p className="text-sm text-black/60">Load one scope to inspect its stored ranking snapshot.</p>
        )}
      </Panel>
    </section>
  );
}

function dedupeScopes(items: ResultListItem[]) {
  const seen = new Set<string>();
  const scopes: RankingScope[] = [];
  for (const item of items) {
    const scope = {
      gameId: item.gameId,
      gameVersion: item.gameVersion,
      rulesetVersion: item.rulesetVersion,
    };
    const key = scopeKey(scope);
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    scopes.push(scope);
  }
  return scopes;
}

function scopeKey(scope: RankingScope) {
  return `${scope.gameId}-${scope.gameVersion}-${scope.rulesetVersion}`.replace(/[^a-zA-Z0-9_-]+/g, "_");
}

function TextField({
  label,
  value,
  onChange,
  required,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
}) {
  return (
    <label className="flex flex-col gap-2 text-sm">
      <span className="font-medium text-black/70">{label}</span>
      <input
        className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm outline-none transition focus:border-accent"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        required={required}
      />
    </label>
  );
}
