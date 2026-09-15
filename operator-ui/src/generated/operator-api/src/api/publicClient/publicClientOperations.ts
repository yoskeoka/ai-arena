import { parse } from "uri-template";
import type { PublicClientContext } from "./publicClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonPublicMatchDetailToApplicationTransform,
  jsonPublicMatchListResponseToApplicationTransform,
  jsonPublicReplayResponseToApplicationTransform,
  jsonPublicStateResponseToApplicationTransform,
} from "../../models/internal/serializers.js";
import type {
  PublicMatchDetail,
  PublicMatchListResponse,
  PublicReplayResponse,
  PublicStateResponse,
} from "../../models/models.js";

export interface ListMatchesOptions extends OperationOptions {
  gameId?: string
  gameVersionMajor?: number
  rulesetVersion?: string
  page?: number
  limit?: number
  sort?: "completed_at"
  sortOrder?: "asc" | "desc"
}
export async function listMatches(
  client: PublicClientContext,
  options?: ListMatchesOptions,
): Promise<PublicMatchListResponse> {
  const path = parse("/api/v1-alpha/public/matches{?game_id,game_version_major,ruleset_version,page,limit,sort,sort_order}").expand({
    ...(options?.gameId && {game_id: options.gameId}),
    ...(options?.gameVersionMajor && {game_version_major: options.gameVersionMajor}),
    ...(options?.rulesetVersion && {ruleset_version: options.rulesetVersion}),
    ...(options?.page && {page: options.page}),
    ...(options?.limit && {limit: options.limit}),
    sort: options?.sort ?? "completed_at",
    ...(options?.sortOrder && {sort_order: options.sortOrder})
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPublicMatchListResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface GetMatchOptions extends OperationOptions {}
export async function getMatch(
  client: PublicClientContext,
  matchId: string,
  options?: GetMatchOptions,
): Promise<PublicMatchDetail> {
  const path = parse("/api/v1-alpha/public/matches/{match_id}").expand({
    match_id: matchId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPublicMatchDetailToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface GetLatestStateOptions extends OperationOptions {}
export async function getLatestState(
  client: PublicClientContext,
  matchId: string,
  options?: GetLatestStateOptions,
): Promise<PublicStateResponse> {
  const path = parse("/api/v1-alpha/public/matches/{match_id}/state").expand({
    match_id: matchId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPublicStateResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
export interface GetReplayOptions extends OperationOptions {}
export async function getReplay(
  client: PublicClientContext,
  matchId: string,
  options?: GetReplayOptions,
): Promise<PublicReplayResponse> {
  const path = parse("/api/v1-alpha/public/matches/{match_id}/replay").expand({
    match_id: matchId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPublicReplayResponseToApplicationTransform(response.body)!;
  }
  throw createRestError(response);
}
;
