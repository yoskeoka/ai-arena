import { FormEvent, useEffect, useMemo, useState } from "react";

import { MatchRequest, MatchRequestParticipant, OperatorApiClient } from "../../api";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";
import { hrefForRunDetail } from "./operatorRoutes";

type RequestsPageProps = {
  baseUrl: string;
};

type EditableParticipant = MatchRequestParticipant & { id: string };

export function RequestsPage({ baseUrl }: RequestsPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [items, setItems] = useState<MatchRequest[]>([]);
  const [listState, setListState] = useState<LoadState>("loading");
  const [listError, setListError] = useState<string>();
  const [writeState, setWriteState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [gameRegistrationID, setGameRegistrationID] = useState("");
  const [outputDir, setOutputDir] = useState("");
  const [participants, setParticipants] = useState<EditableParticipant[]>([
    { id: "p1", player_id: "p1", ai_submission_id: "" },
    { id: "p2", player_id: "p2", ai_submission_id: "" },
  ]);

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
  }, [client]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setWriteState("submitting");
    setWriteError(undefined);
    try {
      await client.createMatchRequest({
        game_registration_id: gameRegistrationID.trim(),
        output_dir: outputDir.trim(),
        participants: participants.map(({ player_id, ai_submission_id }) => ({
          player_id: player_id.trim(),
          ai_submission_id: ai_submission_id.trim(),
        })),
      });
      setWriteState("success");
      await load();
    } catch (error) {
      setWriteState("error");
      setWriteError(messageOf(error));
    }
  };

  const updateParticipant = (id: string, field: "player_id" | "ai_submission_id", value: string) => {
    setParticipants((current) => current.map((item) => (item.id === id ? { ...item, [field]: value } : item)));
  };

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
          <TextField
            label="Game Registration ID"
            value={gameRegistrationID}
            onChange={setGameRegistrationID}
            placeholder="echo-count-v2"
            required
          />
          <TextField label="Output Dir" value={outputDir} onChange={setOutputDir} placeholder="tmp/operator-ui-request-01" required />
          <div className="space-y-3">
            <p className="text-sm font-medium text-black/70">Participants</p>
            {participants.map((participant, index) => (
              <div key={participant.id} className="grid gap-3 rounded-3xl border border-black/10 bg-paper p-4 md:grid-cols-2">
                <TextField
                  label={`Player ${index + 1} ID`}
                  value={participant.player_id}
                  onChange={(value) => updateParticipant(participant.id, "player_id", value)}
                  placeholder={`p${index + 1}`}
                  required
                />
                <TextField
                  label={`Player ${index + 1} AI Submission ID`}
                  value={participant.ai_submission_id}
                  onChange={(value) => updateParticipant(participant.id, "ai_submission_id", value)}
                  placeholder="ai-..."
                  required
                />
              </div>
            ))}
          </div>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
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
                key={item.request_id}
                className="rounded-3xl border border-black/10 bg-paper p-4"
                data-testid={`request-row-${item.request_id}`}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold">{item.request_id}</p>
                    <p className="mt-1 text-sm text-black/70">
                      {item.game.game_id}@{item.game.game_version} / {item.game.ruleset_version}
                    </p>
                  </div>
                  <span className="rounded-full bg-white px-3 py-1 text-xs font-semibold uppercase tracking-wide text-black/65">
                    {item.lifecycle_state}
                  </span>
                </div>
                <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                  <span>match: {item.match_id}</span>
                  <span>latest run: {item.latest_run_id}</span>
                  <span>official run: {item.official_run_id || "n/a"}</span>
                </div>
                <ul className="mt-3 space-y-2 text-sm">
                  {item.participants.map((participant) => (
                    <li key={`${item.request_id}-${participant.player_id}`} className="rounded-2xl bg-white px-3 py-2">
                      {participant.player_id}: {participant.ai_submission_id}
                    </li>
                  ))}
                </ul>
                <div className="mt-3">
                  <a className="text-sm font-semibold text-teal no-underline hover:text-ink" href={hrefForRunDetail(item.latest_run_id)}>
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

function TextField({
  label,
  value,
  onChange,
  placeholder,
  required,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  required?: boolean;
}) {
  return (
    <label className="flex flex-col gap-2 text-sm">
      <span className="font-medium text-black/70">{label}</span>
      <input
        className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm outline-none transition focus:border-accent"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        required={required}
      />
    </label>
  );
}
