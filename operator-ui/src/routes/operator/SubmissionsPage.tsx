import { FormEvent, useEffect, useMemo, useState } from "react";

import { AiBot, OperatorApiClient } from "../../lib/operatorApiClient";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

type SubmissionsPageProps = {
  baseUrl: string;
};

export function SubmissionsPage({ baseUrl }: SubmissionsPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [items, setItems] = useState<AiBot[]>([]);
  const [listState, setListState] = useState<LoadState>("loading");
  const [listError, setListError] = useState<string>();
  const [writeState, setWriteState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [scopeID, setScopeID] = useState("");
  const [botID, setBotID] = useState("");
  const [botName, setBotName] = useState("");
  const [artifactID, setArtifactID] = useState("");

  const load = async () => {
    setListState((current) => (current === "ready" ? current : "loading"));
    try {
      if (!scopeID.trim()) {
        setItems([]);
        setListState("ready");
        return;
      }
      const response = await client.listBots(scopeID.trim(), true);
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
  }, [client, scopeID]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setWriteState("submitting");
    setWriteError(undefined);
    try {
      await client.createOrReviseBot({
        scopeId: scopeID.trim(),
        botId: botID.trim() || undefined,
        botName: botName.trim() || undefined,
        artifactId: artifactID.trim(),
      });
      setWriteState("success");
      await load();
    } catch (error) {
      setWriteState("error");
      setWriteError(messageOf(error));
    }
  };

  const handleRetire = async (botId: string) => {
    setWriteState("submitting");
    setWriteError(undefined);
    try {
      await client.retireBot(botId);
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
        title="Create or revise AI bot"
        subtitle="A revision keeps the selected bot and ranking identity."
        status={writeState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-submissions"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <TextField label="Competition scope" value={scopeID} onChange={setScopeID} placeholder="reversi-v1-regular" required />
          <TextField
            label="Existing bot ID"
            value={botID}
            onChange={setBotID}
            placeholder="leave empty for a new bot"
          />
          <TextField label="Bot name" value={botName} onChange={setBotName} placeholder="required for a new bot" />
          <TextField label="Uploaded AI artifact ID" value={artifactID} onChange={setArtifactID} placeholder="SHA-256 digest from bundle upload" required />
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
            Save bot revision
          </button>
        </form>
      </Panel>

      <Panel
        title="Your bots"
        subtitle="Active and retired bots remain visible with stable identities."
        status={listState}
        error={listError}
        hint={hintFor(listError)}
        testId="operator-panel-submissions"
      >
        {items.length === 0 ? (
          <p className="text-sm text-black/60">Enter a competition scope to list bots.</p>
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <article
                key={item.botId}
                className="rounded-3xl border border-black/10 bg-paper p-4"
                data-testid={`bot-row-${item.botId}`}
              >
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">{item.botName}</p>
                    <p className="mt-1 text-xs text-black/60">{item.botId}</p>
                  </div>
                  <div className="text-xs text-black/60">
                    <span>{item.lifecycleState}</span>
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                  <span>scope: {item.scopeId}</span>
                  <span>active revision: {item.activeSubmissionId || "none"}</span>
                </div>
                {item.lifecycleState === "active" ? (
                  <button className="mt-3 rounded-full border border-black/20 px-3 py-1 text-xs font-semibold" type="button" onClick={() => void handleRetire(item.botId)}>
                    Retire bot
                  </button>
                ) : null}
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
