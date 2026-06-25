import { FormEvent, useEffect, useMemo, useState } from "react";

import { AISubmission, OperatorApiClient } from "../../api";
import { Panel } from "../../shared/ui/Panel";
import { hintFor, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

type SubmissionsPageProps = {
  baseUrl: string;
};

export function SubmissionsPage({ baseUrl }: SubmissionsPageProps) {
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);
  const [items, setItems] = useState<AISubmission[]>([]);
  const [listState, setListState] = useState<LoadState>("loading");
  const [listError, setListError] = useState<string>();
  const [writeState, setWriteState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [submissionID, setSubmissionID] = useState("");
  const [gameRegistrationID, setGameRegistrationID] = useState("");
  const [artifactRef, setArtifactRef] = useState("");
  const [displayName, setDisplayName] = useState("");

  const load = async () => {
    setListState((current) => (current === "ready" ? current : "loading"));
    try {
      const response = await client.listAISubmissions();
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
      await client.createAISubmission({
        ai_submission_id: submissionID.trim() || undefined,
        game_registration_id: gameRegistrationID.trim(),
        artifact_ref: artifactRef.trim(),
        display_name: displayName.trim() || undefined,
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
        title="Register AI"
        subtitle="Admit one AI artifact for one game registration."
        status={writeState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-submissions"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <TextField label="AI Submission ID" value={submissionID} onChange={setSubmissionID} placeholder="optional stable id" />
          <TextField
            label="Game Registration ID"
            value={gameRegistrationID}
            onChange={setGameRegistrationID}
            placeholder="echo-count-v2"
            required
          />
          <TextField label="Artifact Ref" value={artifactRef} onChange={setArtifactRef} placeholder="/abs/path/to/ai" required />
          <TextField label="Display Name" value={displayName} onChange={setDisplayName} placeholder="Echo Bot 01" />
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90" type="submit">
            Create AI submission
          </button>
        </form>
      </Panel>

      <Panel
        title="Admitted AIs"
        subtitle="Ready AI identities for manual requests and preset materialization."
        status={listState}
        error={listError}
        hint={hintFor(listError)}
        testId="operator-panel-submissions"
      >
        {items.length === 0 ? (
          <p className="text-sm text-black/60">No admitted AI submissions yet.</p>
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <article
                key={item.ai_submission_id}
                className="rounded-3xl border border-black/10 bg-paper p-4"
                data-testid={`submission-row-${item.ai_submission_id}`}
              >
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">{item.display_name}</p>
                    <p className="mt-1 text-xs text-black/60">{item.ai_submission_id}</p>
                  </div>
                  <div className="text-xs text-black/60">
                    <span>{item.validation_state}</span>
                  </div>
                </div>
                <p className="mt-2 text-sm text-black/70">
                  {item.game.game_id}@{item.game.game_version} / {item.game.ruleset_version}
                </p>
                <p className="mt-2 break-all text-sm text-black/65">{item.artifact_ref}</p>
                <div className="mt-3 flex flex-wrap gap-4 text-xs text-black/60">
                  <span>game registration: {item.game_registration_id}</span>
                  <span>runtime: {item.runtime_kind}</span>
                  <span>ai id: {item.ai_id}</span>
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
