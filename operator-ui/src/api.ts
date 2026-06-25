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

export type GameRegistration = {
  registration_id: string;
  game: {
    game_id: string;
    game_version: string;
    ruleset_version: string;
  };
  build_mode: string;
  builder_id: string;
  supported_rulesets: string[];
  source?: string;
  source_id?: string;
};

export type AISubmission = {
  ai_submission_id: string;
  game_registration_id: string;
  game: {
    game_id: string;
    game_version: string;
    ruleset_version: string;
  };
  artifact_ref: string;
  display_name: string;
  runtime_kind: string;
  ai_id: string;
  validation_state: string;
  source?: string;
  source_id?: string;
};

export type MatchRequestParticipant = {
  player_id: string;
  ai_submission_id: string;
};

export type MatchRequest = {
  request_id: string;
  game_registration_id: string;
  game: {
    game_id: string;
    game_version: string;
    ruleset_version: string;
  };
  participants: MatchRequestParticipant[];
  output_dir: string;
  source?: string;
  source_id?: string;
  match_id: string;
  latest_run_id: string;
  official_run_id?: string;
  lifecycle_state: LifecycleState;
};

export type RankingScope = {
  game_id: string;
  game_version: string;
  ruleset_version: string;
};

export type RankingEntry = {
  competitor_ref: string;
  last_player_id: string;
  matches_played: number;
  first_places: number;
  placement_counts?: Record<string, number>;
  last_run_id: string;
  last_match_id: string;
  last_status: string;
};

export type RankingSnapshot = {
  scope: RankingScope;
  applied_run_ids?: string[];
  applied_match_ids?: string[];
  last_applied_run_id?: string;
  last_applied_match_id?: string;
  completed_matches: number;
  entries?: RankingEntry[];
};

export type StoredRankingSnapshot = {
  locator: string;
  snapshot: RankingSnapshot;
};

type ListResponse = {
  items: ResultListItem[];
};

type GenericListResponse<T> = {
  items: T[];
};

type EnqueuePayload = {
  preset_id: string;
};

type GameRegistrationPayload = {
  registration_id?: string;
  game: {
    game_id: string;
    game_version: string;
    ruleset_version: string;
  };
};

type AISubmissionPayload = {
  ai_submission_id?: string;
  game_registration_id: string;
  artifact_ref: string;
  display_name?: string;
};

type MatchRequestPayload = {
  request_id?: string;
  game_registration_id: string;
  participants: MatchRequestParticipant[];
  output_dir: string;
  match_id?: string;
};

export class OperatorApiClient {
  constructor(private readonly baseUrl: string) {}

  async session(signal?: AbortSignal): Promise<SessionStatusResponse> {
    const response = await fetch(this.url("/auth/session"), {
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

  async listGames(signal?: AbortSignal): Promise<GameRegistration[]> {
    const response = await fetch(this.url("/api/v1/game-registrations"), {
      signal,
      credentials: "include",
    });
    return this.decodeTypedList<GameRegistration>(response);
  }

  async createGame(payload: GameRegistrationPayload): Promise<GameRegistration> {
    const response = await fetch(this.url("/api/v1/game-registrations"), {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });
    return this.decodeJSON<GameRegistration>(response);
  }

  async listAISubmissions(signal?: AbortSignal): Promise<AISubmission[]> {
    const response = await fetch(this.url("/api/v1/ai-submissions"), {
      signal,
      credentials: "include",
    });
    return this.decodeTypedList<AISubmission>(response);
  }

  async createAISubmission(payload: AISubmissionPayload): Promise<AISubmission> {
    const response = await fetch(this.url("/api/v1/ai-submissions"), {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });
    return this.decodeJSON<AISubmission>(response);
  }

  async listMatchRequests(signal?: AbortSignal): Promise<MatchRequest[]> {
    const response = await fetch(this.url("/api/v1/match-requests"), {
      signal,
      credentials: "include",
    });
    return this.decodeTypedList<MatchRequest>(response);
  }

  async createMatchRequest(payload: MatchRequestPayload): Promise<MatchRequest> {
    const response = await fetch(this.url("/api/v1/match-requests"), {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });
    return this.decodeJSON<MatchRequest>(response);
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

  async getRanking(scope: RankingScope, signal?: AbortSignal): Promise<StoredRankingSnapshot> {
    const params = new URLSearchParams(scope);
    const response = await fetch(this.url(`/api/v1/rankings?${params.toString()}`), {
      signal,
      credentials: "include",
    });
    return this.decodeJSON<StoredRankingSnapshot>(response);
  }

  async cancelRun(runId: string): Promise<ResultListItem> {
    return this.postRunAction(runId, "cancel");
  }

  async retryRun(runId: string): Promise<ResultListItem> {
    return this.postRunAction(runId, "retry");
  }

  async rerunRun(runId: string): Promise<ResultListItem> {
    return this.postRunAction(runId, "rerun");
  }

  async promoteRun(runId: string): Promise<ResultListItem> {
    return this.postRunAction(runId, "promote");
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

  private async decodeTypedList<T>(response: Response): Promise<T[]> {
    const payload = await this.decodeJSON<GenericListResponse<T>>(response);
    if (!isListResponse(payload)) {
      throw new Error("operator API returned an unexpected list payload");
    }
    return payload.items as T[];
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

  private async postRunAction(runId: string, action: "cancel" | "retry" | "rerun" | "promote"): Promise<ResultListItem> {
    const response = await fetch(this.url(`/api/v1/runs/${encodeURIComponent(runId)}/${action}`), {
      method: "POST",
      credentials: "include",
    });
    return this.decodeJSON<ResultListItem>(response);
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
