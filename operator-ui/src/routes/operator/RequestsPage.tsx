import { FormEvent, useEffect, useMemo, useState } from "react";

import { EligibleBot, MatchRequest, MatchRequestParticipant, OperatorApiClient } from "../../lib/operatorApiClient";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";
import { hrefForRunDetail } from "./operatorRoutes";

type RequestsPageProps = {
  baseUrl: string;
};

export function RequestsPage({ baseUrl }: RequestsPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [items, setItems] = useState<MatchRequest[]>([]);
  const [listState, setListState] = useState<LoadState>("loading");
  const [listError, setListError] = useState<string>();
  const [writeState, setWriteState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [games, setGames] = useState<Array<{ registrationId: string; playerCount?: number }>>([]);
  const [scopeID, setScopeID] = useState("");
  const [eligible, setEligible] = useState<EligibleBot[]>([]);
  const [selected, setSelected] = useState<EligibleBot[]>([]);
  const [legacyScopeID, setLegacyScopeID] = useState("");
  const [legacyOutputDir, setLegacyOutputDir] = useState("");
  const [legacyParticipants, setLegacyParticipants] = useState<MatchRequestParticipant[]>([{ playerId: "p1", botId: "", aiSubmissionId: "" }, { playerId: "p2", botId: "", aiSubmissionId: "" }]);

  const load = async () => {
    setListState((current) => (current === "ready" ? current : "loading"));
    try {
      const response = await client.listMatchRequests();
      setItems(response);
      setListState("ready");
      setListError(undefined);
    } catch (error) {
      setListState("error");
      setListError(messageOf(error));
    }
  };

  useEffect(() => {
    void load();
    void client.listGameRegistrations().then(setGames).catch((error) => setListError(messageOf(error)));
  }, [client]);

  const playerCount = games.find((game) => game.registrationId === scopeID)?.playerCount ?? 0;
  useEffect(() => {
    if (!scopeID) { setEligible([]); setSelected([]); return; }
    void client.listEligibleBots(scopeID).then((items) => { setEligible(items); setSelected(shuffle(items).slice(0, playerCount)); }).catch((error) => setWriteError(messageOf(error)));
  }, [client, playerCount, scopeID]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setWriteState("submitting");
    setWriteError(undefined);
    try {
      if (scopeID) {
        await client.createComposedMatchRequest(scopeID, selected.map((bot) => bot.botId));
      } else {
        await client.createMatchRequest({ gameRegistrationId: legacyScopeID, outputDir: legacyOutputDir, participants: legacyParticipants } as never);
      }
      setWriteState("success");
      await load();
    } catch (error) {
      setWriteState("error");
      setWriteError(messageOf(error));
    }
  };

  const shuffleSelection = () => setSelected(shuffle(eligible).slice(0, playerCount));
  const hasLegacyScope = games.some((game) => !game.playerCount);

  return (
    <section className="grid gap-6 xl:grid-cols-[1fr_1fr]">
      <Panel
        title="Create Match Request"
        subtitle="Schedule one logical match from admitted AI submissions."
        status={writeState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-requests"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <label className="flex flex-col gap-2 text-sm"><span className="font-medium text-black/70">Competition scope</span><select value={scopeID} onChange={(event) => setScopeID(event.target.value)}><option value="">Select a scope</option>{games.map((game) => <option key={game.registrationId} value={game.registrationId}>{game.registrationId}</option>)}</select></label>
          <div className="space-y-2"><p className="text-sm font-medium text-black/70">Selected seats ({selected.length}/{playerCount})</p>{selected.map((bot, index) => <p key={bot.botId} className="rounded-2xl bg-white px-3 py-2 text-sm">p{index + 1}: {bot.botName}</p>)}</div>
          <button className="rounded-full border border-ink px-5 py-3 text-sm font-semibold" type="button" onClick={shuffleSelection} disabled={eligible.length < playerCount}>Shuffle</button>
          {scopeID && eligible.length < playerCount ? <p className="text-sm text-red-700">Not enough eligible bots for this scope.</p> : null}
          {hasLegacyScope && !scopeID ? <div className="space-y-3"><TextField label="Game Registration ID" value={legacyScopeID} onChange={setLegacyScopeID} required /><TextField label="Output Dir" value={legacyOutputDir} onChange={setLegacyOutputDir} required />{legacyParticipants.map((participant, index) => <div key={participant.playerId} className="grid gap-3 md:grid-cols-2"><TextField label={`Player ${index + 1} ID`} value={participant.playerId} onChange={(value) => setLegacyParticipants((items) => items.map((item, itemIndex) => itemIndex === index ? { ...item, playerId: value } : item))} required /><TextField label={`Player ${index + 1} AI Submission ID`} value={participant.aiSubmissionId} onChange={(value) => setLegacyParticipants((items) => items.map((item, itemIndex) => itemIndex === index ? { ...item, aiSubmissionId: value } : item))} required /></div>)}</div> : null}
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit" disabled={scopeID ? selected.length !== playerCount : !hasLegacyScope || !legacyScopeID || !legacyOutputDir || legacyParticipants.some((participant) => !participant.playerId || !participant.aiSubmissionId)}>
            Create match request
          </button>
        </form>
      </Panel>

      <Panel
        title="Accepted Requests"
        subtitle="Latest run visibility for manual and preset-sourced logical matches."
        status={listState}
        error={listError}
        hint={hintFor(listError)}
        testId="operator-panel-requests"
      >
        {items.length === 0 ? (
          <p className="text-sm text-black/60">No accepted requests yet.</p>
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <article
                key={item.requestId}
                className="rounded-3xl border border-black/10 bg-paper p-4"
                data-testid={`request-row-${item.requestId}`}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold">{item.requestId}</p>
                    <p className="mt-1 text-sm text-black/70">
                      {item.game.gameId}@{item.game.gameVersion} / {item.game.rulesetVersion}
                    </p>
                  </div>
                  <span className="rounded-full bg-white px-3 py-1 text-xs font-semibold uppercase tracking-wide text-black/65">
                    {item.lifecycleState}
                  </span>
                </div>
                <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                  <span>match: {item.matchId}</span>
                  <span>latest run: {item.latestRunId}</span>
                  <span>official run: {item.officialRunId || "n/a"}</span>
                </div>
                <ul className="mt-3 space-y-2 text-sm">
                  {item.participants.map((participant) => (
                    <li key={`${item.requestId}-${participant.playerId}`} className="rounded-2xl bg-white px-3 py-2">
                      {participant.playerId}: {participant.aiSubmissionId}
                    </li>
                  ))}
                </ul>
                <div className="mt-3">
                  <a className="text-sm font-semibold text-teal no-underline hover:text-ink" href={hrefForRunDetail(item.latestRunId)}>
                    Open latest run detail
                  </a>
                </div>
              </article>
            ))}
          </div>
        )}
      </Panel>
    </section>
  );
}

function shuffle<T>(items: T[]): T[] {
  const result = [...items];
  for (let index = result.length - 1; index > 0; index -= 1) {
    const next = Math.floor(Math.random() * (index + 1));
    [result[index], result[next]] = [result[next], result[index]];
  }
  return result;
}

function TextField({ label, value, onChange, required }: { label: string; value: string; onChange: (value: string) => void; required?: boolean }) {
  return <label className="flex flex-col gap-2 text-sm"><span className="font-medium text-black/70">{label}</span><input className="rounded-2xl border border-black/15 bg-white px-4 py-3" value={value} onChange={(event) => onChange(event.target.value)} required={required} /></label>;
}
