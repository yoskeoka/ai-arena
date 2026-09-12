import {
  type AiArenaClientContext,
  type AiArenaClientOptions,
  createAiArenaClientContext,
} from "./api/aiArenaClientContext.js";
import {
  createOperatorClientContext,
  type OperatorClientContext,
  type OperatorClientOptions,
} from "./api/operatorClient/operatorClientContext.js";
import {
  cancelRun,
  type CancelRunOptions,
  createAiSubmission,
  type CreateAiSubmissionOptions,
  createGameRegistration,
  type CreateGameRegistrationOptions,
  createMatchRequest,
  type CreateMatchRequestOptions,
  createOrReviseBot,
  type CreateOrReviseBotOptions,
  createSignupInvite,
  type CreateSignupInviteOptions,
  getRanking,
  type GetRankingOptions,
  getRun,
  type GetRunOptions,
  healthz,
  type HealthzOptions,
  listActiveMatches,
  type ListActiveMatchesOptions,
  listAiSubmissions,
  type ListAiSubmissionsOptions,
  listBots,
  type ListBotsOptions,
  listCompletedMatches,
  type ListCompletedMatchesOptions,
  listEligibleBots,
  type ListEligibleBotsOptions,
  listGameRegistrations,
  type ListGameRegistrationsOptions,
  listMatchRequests,
  type ListMatchRequestsOptions,
  logout,
  type LogoutOptions,
  promoteRun,
  type PromoteRunOptions,
  rerunRun,
  type RerunRunOptions,
  retireBot,
  type RetireBotOptions,
  retryRun,
  type RetryRunOptions,
  session,
  type SessionOptions,
  uploadAiBundle,
  type UploadAiBundleOptions,
  uploadGameBundle,
  type UploadGameBundleOptions,
  version,
  type VersionOptions,
} from "./api/operatorClient/operatorClientOperations.js";
import {
  createPublicClientContext,
  type PublicClientContext,
  type PublicClientOptions,
} from "./api/publicClient/publicClientContext.js";
import {
  getLatestState,
  type GetLatestStateOptions,
  getMatch,
  type GetMatchOptions,
  getReplay,
  type GetReplayOptions,
  listMatches,
  type ListMatchesOptions,
} from "./api/publicClient/publicClientOperations.js";
import {
  createSharedClientContext,
  type SharedClientContext,
  type SharedClientOptions,
} from "./api/sharedClient/sharedClientContext.js";
import type {
  AiSubmissionRequest,
  BotRevisionRequest,
  File,
  GameRegistrationRequest,
  MatchRequestCreateRequest,
  SignupInviteRequest,
} from "./models/models.js";

export class AiArenaClient {
  #context: AiArenaClientContext
  sharedClient: SharedClient;
  operatorClient: OperatorClient;
  publicClient: PublicClient
  constructor(endpoint: string, options?: AiArenaClientOptions) {
    this.#context = createAiArenaClientContext(endpoint, options);
    this.sharedClient = new SharedClient(endpoint, options);;this
      .operatorClient = new OperatorClient(endpoint, options);;this
      .publicClient = new PublicClient(endpoint, options);
  }
}
export class PublicClient {
  #context: PublicClientContext
  constructor(endpoint: string, options?: PublicClientOptions) {
    this.#context = createPublicClientContext(endpoint, options);

  }
  async listMatches(options?: ListMatchesOptions) {
    return listMatches(this.#context, options);
  };
  async getMatch(matchId: string, options?: GetMatchOptions) {
    return getMatch(this.#context, matchId, options);
  };
  async getLatestState(matchId: string, options?: GetLatestStateOptions) {
    return getLatestState(this.#context, matchId, options);
  };
  async getReplay(matchId: string, options?: GetReplayOptions) {
    return getReplay(this.#context, matchId, options);
  }
}
export class OperatorClient {
  #context: OperatorClientContext
  constructor(endpoint: string, options?: OperatorClientOptions) {
    this.#context = createOperatorClientContext(endpoint, options);

  }
  async healthz(options?: HealthzOptions) {
    return healthz(this.#context, options);
  };
  async version(options?: VersionOptions) {
    return version(this.#context, options);
  };
  async session(options?: SessionOptions) {
    return session(this.#context, options);
  };
  async logout(options?: LogoutOptions) {
    return logout(this.#context, options);
  };
  async listGameRegistrations(options?: ListGameRegistrationsOptions) {
    return listGameRegistrations(this.#context, options);
  };
  async createGameRegistration(
    body: GameRegistrationRequest,
    options?: CreateGameRegistrationOptions,
  ) {
    return createGameRegistration(this.#context, body, options);
  };
  async uploadGameBundle(
    body: {
        bundle: File;
      },
    options?: UploadGameBundleOptions,
  ) {
    return uploadGameBundle(this.#context, body, options);
  };
  async listAiSubmissions(options?: ListAiSubmissionsOptions) {
    return listAiSubmissions(this.#context, options);
  };
  async createAiSubmission(
    body: AiSubmissionRequest,
    options?: CreateAiSubmissionOptions,
  ) {
    return createAiSubmission(this.#context, body, options);
  };
  async createOrReviseBot(
    body: BotRevisionRequest,
    options?: CreateOrReviseBotOptions,
  ) {
    return createOrReviseBot(this.#context, body, options);
  };
  async listBots(scopeId: string, options?: ListBotsOptions) {
    return listBots(this.#context, scopeId, options);
  };
  async listEligibleBots(scopeId: string, options?: ListEligibleBotsOptions) {
    return listEligibleBots(this.#context, scopeId, options);
  };
  async retireBot(botId: string, options?: RetireBotOptions) {
    return retireBot(this.#context, botId, options);
  };
  async uploadAiBundle(
    body: {
        bundle: File;
        gameRegistrationId: string;
        displayName?: string;
      },
    options?: UploadAiBundleOptions,
  ) {
    return uploadAiBundle(this.#context, body, options);
  };
  async listMatchRequests(options?: ListMatchRequestsOptions) {
    return listMatchRequests(this.#context, options);
  };
  async createMatchRequest(
    body: MatchRequestCreateRequest,
    options?: CreateMatchRequestOptions,
  ) {
    return createMatchRequest(this.#context, body, options);
  };
  async createSignupInvite(
    body: SignupInviteRequest,
    options?: CreateSignupInviteOptions,
  ) {
    return createSignupInvite(this.#context, body, options);
  };
  async getRanking(
    gameId: string,
    gameVersion: string,
    rulesetVersion: string,
    options?: GetRankingOptions,
  ) {
    return getRanking(
      this.#context,
      gameId,
      gameVersion,
      rulesetVersion,
      options
    );
  };
  async cancelRun(runId: string, options?: CancelRunOptions) {
    return cancelRun(this.#context, runId, options);
  };
  async retryRun(runId: string, options?: RetryRunOptions) {
    return retryRun(this.#context, runId, options);
  };
  async rerunRun(runId: string, options?: RerunRunOptions) {
    return rerunRun(this.#context, runId, options);
  };
  async promoteRun(runId: string, options?: PromoteRunOptions) {
    return promoteRun(this.#context, runId, options);
  };
  async listActiveMatches(options?: ListActiveMatchesOptions) {
    return listActiveMatches(this.#context, options);
  };
  async listCompletedMatches(options?: ListCompletedMatchesOptions) {
    return listCompletedMatches(this.#context, options);
  };
  async getRun(runId: string, options?: GetRunOptions) {
    return getRun(this.#context, runId, options);
  }
}
export class SharedClient {
  #context: SharedClientContext
  constructor(endpoint: string, options?: SharedClientOptions) {
    this.#context = createSharedClientContext(endpoint, options);

  }
}
