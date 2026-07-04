import { parse } from "uri-template";
import type { OperatorClientContext } from "./operatorClientContext.js";
import { createRestError } from "../helpers/error.js";
import type { OperationOptions } from "../helpers/interfaces.js";
import {
  jsonAiSubmissionRequestToTransportTransform,
  jsonAiSubmissionToApplicationTransform,
  jsonGameRegistrationRequestToTransportTransform,
  jsonGameRegistrationToApplicationTransform,
  jsonHealthResponseToApplicationTransform,
  jsonListResponseToApplicationTransform,
  jsonListResponseToApplicationTransform_2,
  jsonListResponseToApplicationTransform_3,
  jsonListResponseToApplicationTransform_4,
  jsonMatchDetailResponseToApplicationTransform,
  jsonMatchRequestCreateRequestToTransportTransform,
  jsonMatchRequestToApplicationTransform,
  jsonPresetMatchRequestToTransportTransform,
  jsonResultListItemToApplicationTransform,
  jsonSessionStatusResponseToApplicationTransform,
  jsonSignupInviteRequestToTransportTransform,
  jsonSignupInviteResponseToApplicationTransform,
  jsonStoredRankingSnapshotToApplicationTransform,
} from "../models/internal/serializers.js";
import type {
  AiSubmission,
  AiSubmissionRequest,
  GameRegistration,
  GameRegistrationRequest,
  HealthResponse,
  ListResponse,
  ListResponse_2,
  ListResponse_3,
  ListResponse_4,
  MatchDetailResponse,
  MatchRequest,
  MatchRequestCreateRequest,
  PresetMatchRequest,
  ResultListItem,
  SessionStatusResponse,
  SignupInviteRequest,
  SignupInviteResponse,
  StoredRankingSnapshot,
} from "../models/models.js";

export interface HealthzOptions extends OperationOptions {}
export async function healthz(
  client: OperatorClientContext,
  options?: HealthzOptions,
): Promise<HealthResponse> {
  const path = parse("/healthz").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonHealthResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface SessionOptions extends OperationOptions {}
export async function session(
  client: OperatorClientContext,
  options?: SessionOptions,
): Promise<SessionStatusResponse> {
  const path = parse("/auth/session").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSessionStatusResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface LogoutOptions extends OperationOptions {}
export async function logout(
  client: OperatorClientContext,
  options?: LogoutOptions,
): Promise<void> {
  const path = parse("/auth/logout").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 204 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface ListGameRegistrationsOptions extends OperationOptions {}
export async function listGameRegistrations(
  client: OperatorClientContext,
  options?: ListGameRegistrationsOptions,
): Promise<ListResponse> {
  const path = parse("/api/v1/game-registrations").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonListResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface CreateGameRegistrationOptions extends OperationOptions {}
export async function createGameRegistration(
  client: OperatorClientContext,
  body: GameRegistrationRequest,
  options?: CreateGameRegistrationOptions,
): Promise<GameRegistration> {
  const path = parse("/api/v1/game-registrations").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonGameRegistrationRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonGameRegistrationToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface ListAiSubmissionsOptions extends OperationOptions {}
export async function listAiSubmissions(
  client: OperatorClientContext,
  options?: ListAiSubmissionsOptions,
): Promise<ListResponse_2> {
  const path = parse("/api/v1/ai-submissions").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonListResponseToApplicationTransform_2(response.body)!;
  }
  throw createRestError(response);
}
;
export interface CreateAiSubmissionOptions extends OperationOptions {}
export async function createAiSubmission(
  client: OperatorClientContext,
  body: AiSubmissionRequest,
  options?: CreateAiSubmissionOptions,
): Promise<AiSubmission> {
  const path = parse("/api/v1/ai-submissions").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonAiSubmissionRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAiSubmissionToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface ListMatchRequestsOptions extends OperationOptions {}
export async function listMatchRequests(
  client: OperatorClientContext,
  options?: ListMatchRequestsOptions,
): Promise<ListResponse_3> {
  const path = parse("/api/v1/match-requests").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonListResponseToApplicationTransform_3(response.body)!;
  }
  throw createRestError(response);
}
;
export interface CreateMatchRequestOptions extends OperationOptions {}
export async function createMatchRequest(
  client: OperatorClientContext,
  body: MatchRequestCreateRequest,
  options?: CreateMatchRequestOptions,
): Promise<MatchRequest> {
  const path = parse("/api/v1/match-requests").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonMatchRequestCreateRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonMatchRequestToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface CreateSignupInviteOptions extends OperationOptions {}
export async function createSignupInvite(
  client: OperatorClientContext,
  body: SignupInviteRequest,
  options?: CreateSignupInviteOptions,
): Promise<SignupInviteResponse> {
  const path = parse("/api/v1/signup-invites").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonSignupInviteRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSignupInviteResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface GetRankingOptions extends OperationOptions {}
export async function getRanking(
  client: OperatorClientContext,
  gameId: string,
  gameVersion: string,
  rulesetVersion: string,
  options?: GetRankingOptions,
): Promise<StoredRankingSnapshot> {
  const path = parse("/api/v1/rankings{?game_id,game_version,ruleset_version}").expand({
    game_id: gameId,
    game_version: gameVersion,
    ruleset_version: rulesetVersion
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonStoredRankingSnapshotToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface EnqueuePresetOptions extends OperationOptions {}
export async function enqueuePreset(
  client: OperatorClientContext,
  body: PresetMatchRequest,
  options?: EnqueuePresetOptions,
): Promise<ResultListItem> {
  const path = parse("/api/v1/preset-matches").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonPresetMatchRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonResultListItemToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface CancelRunOptions extends OperationOptions {}
export async function cancelRun(
  client: OperatorClientContext,
  runId: string,
  options?: CancelRunOptions,
): Promise<ResultListItem> {
  const path = parse("/api/v1/runs/{run_id}/cancel").expand({
    run_id: runId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonResultListItemToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface RetryRunOptions extends OperationOptions {}
export async function retryRun(
  client: OperatorClientContext,
  runId: string,
  options?: RetryRunOptions,
): Promise<ResultListItem> {
  const path = parse("/api/v1/runs/{run_id}/retry").expand({
    run_id: runId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonResultListItemToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface RerunRunOptions extends OperationOptions {}
export async function rerunRun(
  client: OperatorClientContext,
  runId: string,
  options?: RerunRunOptions,
): Promise<ResultListItem> {
  const path = parse("/api/v1/runs/{run_id}/rerun").expand({
    run_id: runId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonResultListItemToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface PromoteRunOptions extends OperationOptions {}
export async function promoteRun(
  client: OperatorClientContext,
  runId: string,
  options?: PromoteRunOptions,
): Promise<ResultListItem> {
  const path = parse("/api/v1/runs/{run_id}/promote").expand({
    run_id: runId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonResultListItemToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface ListActiveMatchesOptions extends OperationOptions {}
export async function listActiveMatches(
  client: OperatorClientContext,
  options?: ListActiveMatchesOptions,
): Promise<ListResponse_4> {
  const path = parse("/api/v1/matches/active").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonListResponseToApplicationTransform_4(response.body)!;
  }
  throw createRestError(response);
}
;
export interface ListCompletedMatchesOptions extends OperationOptions {}
export async function listCompletedMatches(
  client: OperatorClientContext,
  options?: ListCompletedMatchesOptions,
): Promise<ListResponse_4> {
  const path = parse("/api/v1/matches/completed").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonListResponseToApplicationTransform_4(response.body)!;
  }
  throw createRestError(response);
}
;
export interface GetRunOptions extends OperationOptions {}
export async function getRun(
  client: OperatorClientContext,
  runId: string,
  options?: GetRunOptions,
): Promise<MatchDetailResponse> {
  const path = parse("/api/v1/runs/{run_id}").expand({
    run_id: runId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonMatchDetailResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
