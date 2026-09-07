import { FormEvent, useEffect, useMemo, useState } from "react";

import { GameBundleAdmission, GameRegistration, OperatorApiClient } from "../../lib/operatorApiClient";
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
  const [uploadState, setUploadState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [activationState, setActivationState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [writeError, setWriteError] = useState<string>();
  const [selectedFile, setSelectedFile] = useState<File>();
  const [admission, setAdmission] = useState<GameBundleAdmission>();
  const [rulesetVersion, setRulesetVersion] = useState("");

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

  const handleUpload = async () => {
    if (!selectedFile) {
      setUploadState("error");
      setWriteError("Choose a game bundle ZIP before uploading.");
      return;
    }
    setUploadState("submitting");
    setWriteError(undefined);
    setAdmission(undefined);
    setRulesetVersion("");
    try {
      const response = await client.uploadGameBundle(selectedFile);
      setAdmission(response);
      setRulesetVersion(response.supportedRulesets[0] ?? "");
      setUploadState("success");
    } catch (error) {
      setUploadState("error");
      setWriteError(messageOf(error));
    }
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!admission || rulesetVersion === "") {
      return;
    }
    setActivationState("submitting");
    setWriteError(undefined);
    try {
      await client.createGameRegistration({
        artifactId: admission.artifactId,
        rulesetVersion,
      });
      setActivationState("success");
      setSelectedFile(undefined);
      setAdmission(undefined);
      setRulesetVersion("");
      await load();
    } catch (error) {
      setActivationState("error");
      setWriteError(messageOf(error));
    }
  };

  return (
    <section className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <Panel
        title="Activate uploaded game"
        subtitle="Select an admitted game bundle and one ruleset for a stable competition scope."
        status={activationState === "submitting" || uploadState === "submitting" ? "submitting" : activationState === "error" ? "error" : uploadState}
        error={writeError}
        hint={hintFor(writeError)}
        testId="operator-form-games"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <label className="flex flex-col gap-2 text-sm">
            <span className="font-medium text-black/70">Game bundle ZIP</span>
            <input
              aria-label="Game bundle ZIP"
              className="rounded-2xl border border-black/15 bg-white px-4 py-3 shadow-sm"
              type="file"
              accept=".zip,application/zip"
              onChange={(event) => {
                setSelectedFile(event.target.files?.[0]);
                setAdmission(undefined);
                setRulesetVersion("");
                setUploadState("idle");
                setActivationState("idle");
                setWriteError(undefined);
              }}
            />
          </label>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50" type="button" onClick={() => void handleUpload()} disabled={!selectedFile || uploadState === "submitting" || activationState === "submitting"}>
            {uploadState === "submitting" ? "Uploading game bundle…" : "Upload game bundle"}
          </button>
          {admission ? (
            <div className="space-y-3 rounded-2xl border border-black/10 bg-white p-4" data-testid="game-bundle-admission">
              <p><strong>Game ID:</strong> <span data-testid="admitted-game-id">{admission.gameId}</span></p>
              <p><strong>Game Version:</strong> <span data-testid="admitted-game-version">{admission.gameVersion}</span></p>
              <p><strong>Artifact digest:</strong> <span data-testid="admitted-artifact-id">{admission.artifactId}</span></p>
              <label className="flex flex-col gap-2 text-sm">
                <span className="font-medium text-black/70">Ruleset Version</span>
                <select value={rulesetVersion} onChange={(event) => setRulesetVersion(event.target.value)} required>
                  {admission.supportedRulesets.map((ruleset) => <option key={ruleset} value={ruleset}>{ruleset}</option>)}
                </select>
              </label>
              <button className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-paper transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50" type="submit" disabled={activationState === "submitting" || rulesetVersion === ""}>
                {activationState === "submitting" ? "Activating game…" : "Activate game"}
              </button>
            </div>
          ) : null}
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
                  <span>rulesets: {item.supportedRulesets?.join(", ") || "n/a"}</span>
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
