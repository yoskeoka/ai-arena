import {
  createOperatorClientContext,
  type OperatorClientContext,
  type OperatorClientOptions,
} from "./api/operatorClientContext.js";
import {
  cancelRun,
  type CancelRunOptions,
  createAiSubmission,
  type CreateAiSubmissionOptions,
  createGameRegistration,
  type CreateGameRegistrationOptions,
  createMatchRequest,
  type CreateMatchRequestOptions,
  createSignupInvite,
  type CreateSignupInviteOptions,
  enqueuePreset,
  type EnqueuePresetOptions,
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
  listCompletedMatches,
  type ListCompletedMatchesOptions,
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
  retryRun,
  type RetryRunOptions,
  session,
  type SessionOptions,
  uploadAiBundle,
  type UploadAiBundleOptions,
  uploadGameBundle,
  type UploadGameBundleOptions,
} from "./api/operatorClientOperations.js";
import type {
  AiSubmissionRequest,
  File,
  GameRegistrationRequest,
  MatchRequestCreateRequest,
  PresetMatchRequest,
  SignupInviteRequest,
} from "./models/models.js";

export class OperatorClient {
  #context: OperatorClientContext
  constructor(endpoint: string, options?: OperatorClientOptions) {
    this.#context = createOperatorClientContext(endpoint, options);

  }
  async healthz(options?: HealthzOptions) {
    return healthz(this.#context, options);
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
  async enqueuePreset(
    body: PresetMatchRequest,
    options?: EnqueuePresetOptions,
  ) {
    return enqueuePreset(this.#context, body, options);
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
