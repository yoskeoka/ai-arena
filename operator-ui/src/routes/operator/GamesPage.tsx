import { FormEvent, useEffect, useMemo, useState } from "react";

import { GameRegistration, OperatorApiClient } from "../../lib/operatorApiClient";
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
  const [rulesetVersion, setRulesetVersion] = useState("");
  const [artifactID, setArtifactID] = useState("");
  const [registrationID, setRegistrationID] = useState("");
  const [gameID, setGameID] = useState("");
  const [gameVersion, setGameVersion] = useState("");

  const load = async () => {
    setListState((current) => (current === "ready" ? current : "loading"));
    try {
      const response = await client.listGameRegistrations();
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
      await client.createGameRegistration({
        registrationId: registrationID.trim() || undefined,
        game: gameID.trim() ? { gameId: gameID.trim(), gameVersion: gameVersion.trim(), rulesetVersion: rulesetVersion.trim() } : undefined,
        artifactId: artifactID.trim() || undefined,
        rulesetVersion: rulesetVersion.trim() || undefined,
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
        title="Activate uploaded game"
        subtitle="Select an admitted game bundle and one ruleset for a stable competition scope."
        status={writeState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-games"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <TextField label="Registration ID" value={registrationID} onChange={setRegistrationID} placeholder="legacy compatibility id" />
          <TextField label="Game ID" value={gameID} onChange={setGameID} placeholder="derived from uploaded artifact when omitted" />
          <TextField label="Game Version" value={gameVersion} onChange={setGameVersion} placeholder="derived from uploaded artifact when omitted" />
          <TextField
            label="Ruleset Version"
            value={rulesetVersion}
            onChange={setRulesetVersion}
            placeholder="regular"
            required
          />
          <TextField label="Uploaded game artifact ID" value={artifactID} onChange={setArtifactID} placeholder="SHA-256 digest from bundle upload" />
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
            Activate competition scope
          </button>
        </form>
      </Panel>

      <Panel
        title="Competition scopes"
        subtitle="Active exact game releases and their stable major/ruleset scope identities."
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
                key={item.registrationId}
                className="rounded-3xl border border-black/10 bg-paper p-4"
                data-testid={`game-row-${item.registrationId}`}
              >
                <p className="font-semibold">{item.registrationId}</p>
                <p className="mt-1 text-sm text-black/70">
                  {item.game.gameId}@{item.game.gameVersion} / {item.game.rulesetVersion}
                </p>
                <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                  <span>build: {item.buildMode}</span>
                  <span>builder: {item.builderId}</span>
                  <span>rulesets: {item.supportedRulesets.join(", ") || "n/a"}</span>
                  <span>artifact: {item.artifactId || "builtin"}</span>
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
