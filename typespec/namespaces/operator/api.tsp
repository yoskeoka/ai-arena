import "@typespec/http";
import "@typespec/rest";

import "../shared.tsp";

using Http;
using Rest;
using AiArena.Shared;

namespace AiArena.Operator;

@route("/api/v1/game-registrations")
@get
op listGameRegistrations(): GameRegistrationListResponse;

@route("/api/v1/game-registrations")
@post
op createGameRegistration(@body body: GameRegistrationRequest): GameRegistration;

@route("/api/v1/game-bundles")
@post
op uploadGameBundle(@multipartBody body: { bundle: HttpPart<File>; }): {
  @statusCode statusCode: 201;
  @body body: GameBundleAdmission;
};

@route("/api/v1/ai-submissions")
@get
op listAISubmissions(): AISubmissionListResponse;

@route("/api/v1/ai-submissions")
@post
op createAISubmission(@body body: AISubmissionRequest): AISubmission;

@route("/api/v1/bots")
@post
op createOrReviseBot(@body body: BotRevisionRequest): BotRevisionResponse;

@route("/api/v1/bots")
@get
op listBots(@query scope_id: string, @query include_retired?: boolean): AIBotListResponse;

@route("/api/v1/bots/{bot_id}/retire")
@post
op retireBot(@path bot_id: string): AIBot;

@route("/api/v1/ai-bundles")
@post
op uploadAIBundle(@multipartBody body: { bundle: HttpPart<File>; game_registration_id: HttpPart<string>; display_name?: HttpPart<string>; }): {
  @statusCode statusCode: 201;
  @body body: AISubmission;
};

@route("/api/v1/match-requests")
@get
op listMatchRequests(): MatchRequestListResponse;

@route("/api/v1/match-requests")
@post
op createMatchRequest(@body body: MatchRequestCreateRequest): MatchRequest;

@route("/api/v1/signup-invites")
@post
op createSignupInvite(@body body: SignupInviteRequest): SignupInviteResponse;

@route("/api/v1/rankings")
@get
op getRanking(
  @query game_id: string,
  @query game_version: string,
  @query ruleset_version: string,
): StoredRankingSnapshot;

@route("/api/v1/preset-matches")
@post
op enqueuePreset(@body body: PresetMatchRequest): ResultListItem;

@route("/api/v1/runs/{run_id}/cancel")
@post
op cancelRun(@path run_id: string): ResultListItem;

@route("/api/v1/runs/{run_id}/retry")
@post
op retryRun(@path run_id: string): ResultListItem;

@route("/api/v1/runs/{run_id}/rerun")
@post
op rerunRun(@path run_id: string): ResultListItem;

@route("/api/v1/runs/{run_id}/promote")
@post
op promoteRun(@path run_id: string): ResultListItem;

@route("/api/v1/matches/active")
@get
op listActiveMatches(): RunListResponse;

@route("/api/v1/matches/completed")
@get
op listCompletedMatches(): RunListResponse;

@route("/api/v1/runs/{run_id}")
@get
op getRun(@path run_id: string): MatchDetailResponse;
