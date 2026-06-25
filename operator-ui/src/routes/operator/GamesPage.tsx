import { FormEvent, useEffect, useMemo, useState } from "react";

import { GameRegistration, OperatorApiClient } from "../../api";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

type GamesPageProps = {
  baseUrl: string;
};

export function GamesPage({ baseUrl }: GamesPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [items, setItems] = useState<GameRegistration[]>([]);
  const [listState, setListState] = useState<LoadState>("loading");
  const [listError, setListError] = useState<string>();
  const [writeState, setWriteState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [registrationID, setRegistrationID] = useState("");
  const [gameID, setGameID] = useState("");
  const [gameVersion, setGameVersion] = useState("");
  const [rulesetVersion, setRulesetVersion] = useState("");

  const load = async () => {
    setListState((current) => (current === "ready" ? current : "loading"));
    try {
      const response = await client.listGames();
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
      await client.createGame({
        registration_id: registrationID.trim() || undefined,
        game: {
          game_id: gameID.trim(),
          game_version: gameVersion.trim(),
          ruleset_version: rulesetVersion.trim(),
        },
      });
      setWriteState("success");
      await load();
    } catch (error) {
      setWriteState("error");
      setWriteError(messageOf(error));
    }
  };

  return (
    <section className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <Panel
        title="Register Game"
        subtitle="Add one operator-visible game registration."
        status={writeState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-games"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <TextField label="Registration ID" value={registrationID} onChange={setRegistrationID} placeholder="optional stable id" />
          <TextField label="Game ID" value={gameID} onChange={setGameID} placeholder="echo-count" required />
          <TextField label="Game Version" value={gameVersion} onChange={setGameVersion} placeholder="2.0.0" required />
          <TextField
            label="Ruleset Version"
            value={rulesetVersion}
            onChange={setRulesetVersion}
            placeholder="phase2-simultaneous-2turn"
            required
          />
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
            Create game registration
          </button>
        </form>
      </Panel>

      <Panel
        title="Registered Games"
        subtitle="Insertion-order list from the operator API."
        status={listState}
        error={listError}
        hint={hintFor(listError)}
        testId="operator-panel-games"
      >
        {items.length === 0 ? (
          <p className="text-sm text-black/60">No registered games yet.</p>
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <article
                key={item.registration_id}
                className="rounded-3xl border border-black/10 bg-paper p-4"
                data-testid={`game-row-${item.registration_id}`}
              >
                <p className="font-semibold">{item.registration_id}</p>
                <p className="mt-1 text-sm text-black/70">
                  {item.game.game_id}@{item.game.game_version} / {item.game.ruleset_version}
                </p>
                <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                  <span>build: {item.build_mode}</span>
                  <span>builder: {item.builder_id}</span>
                  <span>rulesets: {item.supported_rulesets.join(", ") || "n/a"}</span>
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
