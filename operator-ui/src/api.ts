export type LifecycleState =
  | "queued"
  | "leased"
  | "running"
  | "persisting"
  | "completed"
  | "failed"
  | "canceled";

export type ResultListItem = {
  submission_id: string;
  match_id: string;
  game_id: string;
  game_version: string;
  ruleset_version: string;
  lifecycle_state: LifecycleState;
  worker_id?: string;
  terminal_status?: string;
  error?: string;
  turn?: number;
  placements?: Placement[];
  result_summary_path?: string;
};

export type Placement = {
  player_id: string;
  rank: number;
  score?: number;
};

export type SubmittedPlayer = {
  player_id: string;
  artifact_ref: string;
};

export type VerificationSummary = {
  issues: string[];
};

export type ReplayInputs = {
  record_path?: string;
  snapshot_path?: string;
  history_path?: string;
  exported_snapshot_path?: string;
  verification?: VerificationSummary;
};

export type ResultSummary = {
  status: string;
  turn: number;
  error?: string;
  placements?: Placement[];
};

export type ArtifactAccessMetadata = {
  locator: string;
  download_url?: string;
  issuer?: string;
  status?: string;
  expires_at?: string;
};

export type MatchDetailResponse = ResultListItem & {
  players: SubmittedPlayer[];
  output_dir: string;
  match_dir?: string;
  record_path?: string;
  player_stderr_paths?: Record<string, string>;
  result_summary?: ResultSummary;
  replay_inputs?: ReplayInputs;
  artifact_access?: Record<string, ArtifactAccessMetadata>;
};

type ListResponse = {
  items: ResultListItem[];
};

type EnqueuePayload = {
  preset_id: string;
};

export class OperatorApiClient {
  constructor(private readonly baseUrl: string) {}

  async listActive(signal?: AbortSignal): Promise<ResultListItem[]> {
    const response = await fetch(this.url("/api/v1/matches/active"), { signal });
    return this.decodeList(response);
  }

  async listCompleted(signal?: AbortSignal): Promise<ResultListItem[]> {
    const response = await fetch(this.url("/api/v1/matches/completed"), { signal });
    return this.decodeList(response);
  }

  async getMatchDetail(submissionId: string, signal?: AbortSignal): Promise<MatchDetailResponse> {
    const response = await fetch(this.url(`/api/v1/matches/${encodeURIComponent(submissionId)}`), { signal });
    return this.decodeJSON<MatchDetailResponse>(response);
  }

  async enqueuePreset(presetId: string): Promise<ResultListItem> {
    const response = await fetch(this.url("/api/v1/preset-matches"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ preset_id: presetId } satisfies EnqueuePayload),
    });
    return this.decodeJSON<ResultListItem>(response);
  }

  private async decodeList(response: Response): Promise<ResultListItem[]> {
    const payload = await this.decodeJSON<ListResponse>(response);
    return payload.items;
  }

  private async decodeJSON<T>(response: Response): Promise<T> {
    const payload = (await response.json()) as T | { error?: string };
    if (!response.ok) {
      const message =
        typeof payload === "object" && payload !== null && "error" in payload && typeof payload.error === "string"
          ? payload.error
          : `request failed with status ${response.status}`;
      throw new Error(message);
    }
    return payload as T;
  }

  private url(pathname: string): string {
    const base = this.baseUrl.endsWith("/") ? this.baseUrl.slice(0, -1) : this.baseUrl;
    return `${base}${pathname}`;
  }
}
