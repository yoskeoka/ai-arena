export type LifecycleState =
  | "queued"
  | "leased"
  | "running"
  | "persisting"
  | "completed"
  | "failed"
  | "canceled";

export type ResultListItem = {
  run_id: string;
  match_id: string;
  attempt_count: number;
  official: boolean;
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

export type AuthPrincipal = {
  account_id: string;
  provider: string;
  provider_login: string;
  provider_email?: string;
  roles: string[];
};

export type SessionStatusResponse = {
  auth_mode: "disabled" | "enabled";
  authenticated: boolean;
  principal?: AuthPrincipal;
};

type ListResponse = {
  items: ResultListItem[];
};

type EnqueuePayload = {
  preset_id: string;
};

export class OperatorApiClient {
  constructor(private readonly baseUrl: string) {}

  async session(signal?: AbortSignal): Promise<SessionStatusResponse> {
    const response = await fetch(this.url("/api/v1/session"), {
      signal,
      credentials: "include",
    });
    return this.decodeJSON<SessionStatusResponse>(response);
  }

  async listActive(signal?: AbortSignal): Promise<ResultListItem[]> {
    const response = await fetch(this.url("/api/v1/matches/active"), {
      signal,
      credentials: "include",
    });
    return this.decodeList(response);
  }

  async listCompleted(signal?: AbortSignal): Promise<ResultListItem[]> {
    const response = await fetch(this.url("/api/v1/matches/completed"), {
      signal,
      credentials: "include",
    });
    return this.decodeList(response);
  }

  async getMatchDetail(runId: string, signal?: AbortSignal): Promise<MatchDetailResponse> {
    const response = await fetch(this.url(`/api/v1/runs/${encodeURIComponent(runId)}`), {
      signal,
      credentials: "include",
    });
    return this.decodeJSON<MatchDetailResponse>(response);
  }

  async enqueuePreset(presetId: string): Promise<ResultListItem> {
    const response = await fetch(this.url("/api/v1/preset-matches"), {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ preset_id: presetId } satisfies EnqueuePayload),
    });
    return this.decodeJSON<ResultListItem>(response);
  }

  async logout(): Promise<void> {
    const response = await fetch(this.url("/auth/logout"), {
      method: "POST",
      credentials: "include",
    });
    if (!response.ok) {
      throw new Error(`request failed with status ${response.status}`);
    }
  }

  githubLoginURL(returnTo: string, inviteToken?: string): string {
    const params = new URLSearchParams({ return_to: returnTo });
    if (inviteToken && inviteToken.trim() !== "") {
      params.set("invite_token", inviteToken.trim());
    }
    return `${this.url("/auth/github/login")}?${params.toString()}`;
  }

  private async decodeList(response: Response): Promise<ResultListItem[]> {
    const payload = await this.decodeJSON<ListResponse>(response);
    if (!isListResponse(payload)) {
      throw new Error("operator API returned an unexpected list payload");
    }
    return payload.items;
  }

  private async decodeJSON<T>(response: Response): Promise<T> {
    const payload = await decodeResponseBody<T>(response);
    if (!response.ok) {
      const message =
        typeof payload === "object" && payload !== null && "error" in payload && typeof payload.error === "string"
          ? payload.error
          : typeof payload === "string" && payload.trim() !== ""
            ? payload
            : `request failed with status ${response.status}`;
      throw new Error(message);
    }
    return payload as T;
  }

  private url(pathname: string): string {
    const trimmed = this.baseUrl.trim();
    if (trimmed === "") {
      return pathname;
    }
    const base = trimmed.endsWith("/") ? trimmed.slice(0, -1) : trimmed;
    return `${base}${pathname}`;
  }
}

function isListResponse(payload: unknown): payload is ListResponse {
  return typeof payload === "object" && payload !== null && "items" in payload && Array.isArray(payload.items);
}

async function decodeResponseBody<T>(response: Response): Promise<T | { error?: string } | string> {
  const body = await response.text();
  if (body.trim() === "") {
    return "";
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return JSON.parse(body) as T | { error?: string };
  }

  try {
    return JSON.parse(body) as T | { error?: string };
  } catch {
    return body;
  }
}
