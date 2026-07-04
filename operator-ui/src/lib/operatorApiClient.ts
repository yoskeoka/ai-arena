import { getClient, type Client, type PathUncheckedResponse, type PipelinePolicy, RestError } from "@typespec/ts-http-runtime";

import {
  jsonAiSubmissionListResponseToApplicationTransform,
  jsonAiSubmissionRequestToTransportTransform,
  jsonAiSubmissionToApplicationTransform,
  jsonGameRegistrationListResponseToApplicationTransform,
  jsonGameRegistrationRequestToTransportTransform,
  jsonGameRegistrationToApplicationTransform,
  jsonMatchDetailResponseToApplicationTransform,
  jsonMatchRequestCreateRequestToTransportTransform,
  jsonMatchRequestListResponseToApplicationTransform,
  jsonMatchRequestToApplicationTransform,
  jsonPresetMatchRequestToTransportTransform,
  jsonResultListItemToApplicationTransform,
  jsonRunListResponseToApplicationTransform,
  jsonSessionStatusResponseToApplicationTransform,
  jsonSignupInviteRequestToTransportTransform,
  jsonSignupInviteResponseToApplicationTransform,
  jsonStoredRankingSnapshotToApplicationTransform,
} from "../generated/operator-api/src/models/internal/serializers.js";
import type {
  AiSubmission,
  AiSubmissionRequest,
  AuthPrincipal,
  GameRegistration,
  GameRegistrationRequest,
  MatchDetailResponse,
  MatchRequest,
  MatchRequestCreateRequest,
  MatchRequestParticipant,
  PresetMatchRequest,
  RankingScope,
  ResultListItem,
  SessionStatusResponse,
  SignupInviteRequest,
  SignupInviteResponse,
  StoredRankingSnapshot,
} from "../generated/operator-api/src/models/models.js";

export type {
  AiSubmission,
  AiSubmissionRequest,
  AuthPrincipal,
  GameRegistration,
  GameRegistrationRequest,
  MatchDetailResponse,
  MatchRequest,
  MatchRequestCreateRequest,
  MatchRequestParticipant,
  RankingScope,
  ResultListItem,
  SessionStatusResponse,
  SignupInviteRequest,
  SignupInviteResponse,
  StoredRankingSnapshot,
};

const credentialsPolicy: PipelinePolicy = {
  name: "operator-ui-with-credentials",
  async sendRequest(request, next) {
    request.withCredentials = true;
    return next(request);
  },
};

export class OperatorApiClient {
  readonly #baseUrl: string;
  readonly #client: Client;

  constructor(baseUrl: string) {
    this.#baseUrl = baseUrl.trim();
    this.#client = getClient(this.endpoint(), {
      allowInsecureConnection: true,
      additionalPolicies: [{ policy: credentialsPolicy, position: "perCall" }],
    });
  }

  async session(signal?: AbortSignal): Promise<SessionStatusResponse> {
    const response = await this.get("/auth/session", signal);
    return jsonSessionStatusResponseToApplicationTransform(response.body)!;
  }

  async logout(signal?: AbortSignal): Promise<void> {
    await this.post("/auth/logout", undefined, [204], signal);
  }

  async listGameRegistrations(signal?: AbortSignal): Promise<GameRegistration[]> {
    const response = await this.get("/api/v1/game-registrations", signal);
    return jsonGameRegistrationListResponseToApplicationTransform(response.body)!.items;
  }

  async createGameRegistration(body: GameRegistrationRequest, signal?: AbortSignal): Promise<GameRegistration> {
    const response = await this.post(
      "/api/v1/game-registrations",
      jsonGameRegistrationRequestToTransportTransform(body),
      [200, 201],
      signal,
    );
    return jsonGameRegistrationToApplicationTransform(response.body)!;
  }

  async listAiSubmissions(signal?: AbortSignal): Promise<AiSubmission[]> {
    const response = await this.get("/api/v1/ai-submissions", signal);
    return jsonAiSubmissionListResponseToApplicationTransform(response.body)!.items;
  }

  async createAiSubmission(body: AiSubmissionRequest, signal?: AbortSignal): Promise<AiSubmission> {
    const response = await this.post(
      "/api/v1/ai-submissions",
      jsonAiSubmissionRequestToTransportTransform(body),
      [200, 201],
      signal,
    );
    return jsonAiSubmissionToApplicationTransform(response.body)!;
  }

  async listMatchRequests(signal?: AbortSignal): Promise<MatchRequest[]> {
    const response = await this.get("/api/v1/match-requests", signal);
    return jsonMatchRequestListResponseToApplicationTransform(response.body)!.items;
  }

  async createMatchRequest(body: MatchRequestCreateRequest, signal?: AbortSignal): Promise<MatchRequest> {
    const response = await this.post(
      "/api/v1/match-requests",
      jsonMatchRequestCreateRequestToTransportTransform(body),
      [200, 201],
      signal,
    );
    return jsonMatchRequestToApplicationTransform(response.body)!;
  }

  async createSignupInvite(body: SignupInviteRequest, signal?: AbortSignal): Promise<SignupInviteResponse> {
    const response = await this.post(
      "/api/v1/signup-invites",
      jsonSignupInviteRequestToTransportTransform(body),
      [200, 201],
      signal,
    );
    return jsonSignupInviteResponseToApplicationTransform(response.body)!;
  }

  async listActiveMatches(signal?: AbortSignal): Promise<ResultListItem[]> {
    const response = await this.get("/api/v1/matches/active", signal);
    return jsonRunListResponseToApplicationTransform(response.body)!.items;
  }

  async listCompletedMatches(signal?: AbortSignal): Promise<ResultListItem[]> {
    const response = await this.get("/api/v1/matches/completed", signal);
    return jsonRunListResponseToApplicationTransform(response.body)!.items;
  }

  async getRun(runId: string, signal?: AbortSignal): Promise<MatchDetailResponse> {
    const response = await this.get(`/api/v1/runs/${encodeURIComponent(runId)}`, signal);
    return jsonMatchDetailResponseToApplicationTransform(response.body)!;
  }

  async getRanking(scope: RankingScope, signal?: AbortSignal): Promise<StoredRankingSnapshot> {
    const params = new URLSearchParams({
      game_id: scope.gameId,
      game_version: scope.gameVersion,
      ruleset_version: scope.rulesetVersion,
    });
    const response = await this.get(`/api/v1/rankings?${params.toString()}`, signal);
    return jsonStoredRankingSnapshotToApplicationTransform(response.body)!;
  }

  async enqueuePreset(presetId: string, signal?: AbortSignal): Promise<ResultListItem> {
    const body: PresetMatchRequest = { presetId };
    const response = await this.post(
      "/api/v1/preset-matches",
      jsonPresetMatchRequestToTransportTransform(body),
      [200, 201, 202],
      signal,
    );
    return jsonResultListItemToApplicationTransform(response.body)!;
  }

  async cancelRun(runId: string, signal?: AbortSignal): Promise<ResultListItem> {
    const response = await this.post(`/api/v1/runs/${encodeURIComponent(runId)}/cancel`, undefined, [200, 202], signal);
    return jsonResultListItemToApplicationTransform(response.body)!;
  }

  async retryRun(runId: string, signal?: AbortSignal): Promise<ResultListItem> {
    const response = await this.post(`/api/v1/runs/${encodeURIComponent(runId)}/retry`, undefined, [200, 201, 202], signal);
    return jsonResultListItemToApplicationTransform(response.body)!;
  }

  async rerunRun(runId: string, signal?: AbortSignal): Promise<ResultListItem> {
    const response = await this.post(`/api/v1/runs/${encodeURIComponent(runId)}/rerun`, undefined, [200, 201, 202], signal);
    return jsonResultListItemToApplicationTransform(response.body)!;
  }

  async promoteRun(runId: string, signal?: AbortSignal): Promise<ResultListItem> {
    const response = await this.post(`/api/v1/runs/${encodeURIComponent(runId)}/promote`, undefined, [200, 201, 202], signal);
    return jsonResultListItemToApplicationTransform(response.body)!;
  }

  githubLoginURL(returnTo: string, inviteToken?: string): string {
    const params = new URLSearchParams({ return_to: returnTo });
    if (inviteToken && inviteToken.trim() !== "") {
      params.set("invite_token", inviteToken.trim());
    }
    return `${this.url("/auth/github/login")}?${params.toString()}`;
  }

  private endpoint() {
    if (this.#baseUrl !== "") {
      return this.#baseUrl;
    }
    if (typeof window !== "undefined") {
      return window.location.origin;
    }
    return "http://localhost:4173";
  }

  private url(pathname: string) {
    if (this.#baseUrl === "") {
      return pathname;
    }
    const base = this.#baseUrl.endsWith("/") ? this.#baseUrl.slice(0, -1) : this.#baseUrl;
    return `${base}${pathname}`;
  }

  private async get(path: string, signal?: AbortSignal) {
    return this.request(() => this.#client.pathUnchecked(path).get({ abortSignal: signal }), [200]);
  }

  private async post(path: string, body: unknown, okStatuses: number[], signal?: AbortSignal) {
    return this.request(() => this.#client.pathUnchecked(path).post({ body, abortSignal: signal }), okStatuses);
  }

  private async request(run: () => PromiseLike<PathUncheckedResponse>, okStatuses: number[]) {
    try {
      const response = await run();
      const status = Number(response.status);
      if (okStatuses.includes(status)) {
        return response;
      }
      throw new Error(responseErrorMessage(response));
    } catch (error) {
      throw normalizeOperatorError(error);
    }
  }
}

function normalizeOperatorError(error: unknown) {
  if (error instanceof DOMException && error.name === "AbortError") {
    return error;
  }
  if (error instanceof RestError) {
    return new Error(error.message);
  }
  if (error instanceof Error) {
    return error;
  }
  return new Error("unknown error");
}

function responseErrorMessage(response: PathUncheckedResponse) {
  const body = response.body;
  if (typeof body === "string" && body.trim() !== "") {
    return body;
  }
  if (typeof body === "object" && body !== null) {
    if ("error" in body && typeof body.error === "string" && body.error.trim() !== "") {
      return body.error;
    }
    if (
      "error" in body &&
      typeof body.error === "object" &&
      body.error !== null &&
      "message" in body.error &&
      typeof body.error.message === "string" &&
      body.error.message.trim() !== ""
    ) {
      return body.error.message;
    }
  }
  return `request failed with status ${response.status}`;
}
