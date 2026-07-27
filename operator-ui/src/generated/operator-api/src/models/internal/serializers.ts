import type {
  AiSubmission,
  AiSubmissionListResponse,
  AiSubmissionRequest,
  ArtifactAccessMetadata,
  AuthPrincipal,
  File,
  GameMetadata,
  GameRegistration,
  GameRegistrationListResponse,
  GameRegistrationRequest,
  HealthResponse,
  MatchDetailResponse,
  MatchRequest,
  MatchRequestCreateRequest,
  MatchRequestListResponse,
  MatchRequestParticipant,
  PathRefs,
  Placement,
  PresetMatchRequest,
  RankingEntry,
  RankingScope,
  RankingSnapshot,
  ReplayInputs,
  ResultListItem,
  ResultSummary,
  RunListResponse,
  SessionStatusResponse,
  SignupInviteRequest,
  SignupInviteResponse,
  StoredRankingSnapshot,
  SubmittedPlayer,
  VerificationSummary,
} from "../models.js";

export function decodeBase64(value: string): Uint8Array | undefined {
  if(!value) {
    return value as any;
  }
  // Normalize Base64URL to Base64
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/')
    .padEnd(value.length + (4 - (value.length % 4)) % 4, '=');

  return new Uint8Array(Buffer.from(base64, 'base64'));
}export function encodeUint8Array(
  value: Uint8Array | undefined | null,
  encoding: BufferEncoding,
): string | undefined {
  if (!value) {
    return value as any;
  }
  return Buffer.from(value).toString(encoding);
}export function dateDeserializer(date?: string | null): Date {
  if (!date) {
    return date as any;
  }

  return new Date(date);
}export function dateRfc7231Deserializer(date?: string | null): Date {
  if (!date) {
    return date as any;
  }

  return new Date(date);
}export function dateRfc3339Serializer(date?: Date | null): string {
  if (!date) {
    return date as any
  }

  return date.toISOString();
}export function dateRfc7231Serializer(date?: Date | null): string {
  if (!date) {
    return date as any;
  }

  return date.toUTCString();
}export function dateUnixTimestampSerializer(date?: Date | null): number {
  if (!date) {
    return date as any;
  }

  return Math.floor(date.getTime() / 1000);
}export function dateUnixTimestampDeserializer(date?: number | null): Date {
  if (!date) {
    return date as any;
  }

  return new Date(date * 1000);
}export function createGameRegistrationPayloadToTransport(
  payload: GameRegistrationRequest,
) {
  return jsonGameRegistrationRequestToTransportTransform(payload)!;
}export function createAiSubmissionPayloadToTransport(
  payload: AiSubmissionRequest,
) {
  return jsonAiSubmissionRequestToTransportTransform(payload)!;
}export function createMatchRequestPayloadToTransport(
  payload: MatchRequestCreateRequest,
) {
  return jsonMatchRequestCreateRequestToTransportTransform(payload)!;
}export function createSignupInvitePayloadToTransport(
  payload: SignupInviteRequest,
) {
  return jsonSignupInviteRequestToTransportTransform(payload)!;
}export function enqueuePresetPayloadToTransport(payload: PresetMatchRequest) {
  return jsonPresetMatchRequestToTransportTransform(payload)!;
}export function jsonHealthResponseToTransportTransform(
  input_?: HealthResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: input_.status
  }!;
}export function jsonHealthResponseToApplicationTransform(
  input_?: any,
): HealthResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: input_.status
  }!;
}export function jsonSessionStatusResponseToTransportTransform(
  input_?: SessionStatusResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    auth_mode: input_.authMode,authenticated: input_.authenticated,principal: jsonAuthPrincipalToTransportTransform(input_.principal)
  }!;
}export function jsonSessionStatusResponseToApplicationTransform(
  input_?: any,
): SessionStatusResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    authMode: input_.auth_mode,authenticated: input_.authenticated,principal: jsonAuthPrincipalToApplicationTransform(input_.principal)
  }!;
}export function jsonAuthPrincipalToTransportTransform(
  input_?: AuthPrincipal | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    account_id: input_.accountId,provider: input_.provider,provider_login: input_.providerLogin,provider_email: input_.providerEmail,roles: jsonArrayStringToTransportTransform(input_.roles)
  }!;
}export function jsonAuthPrincipalToApplicationTransform(
  input_?: any,
): AuthPrincipal {
  if(!input_) {
    return input_ as any;
  }
    return {
    accountId: input_.account_id,provider: input_.provider,providerLogin: input_.provider_login,providerEmail: input_.provider_email,roles: jsonArrayStringToApplicationTransform(input_.roles)
  }!;
}export function jsonArrayStringToTransportTransform(
  items_?: Array<string> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayStringToApplicationTransform(
  items_?: any,
): Array<string> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonGameRegistrationListResponseToTransportTransform(
  input_?: GameRegistrationListResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayGameRegistrationToTransportTransform(input_.items)
  }!;
}export function jsonGameRegistrationListResponseToApplicationTransform(
  input_?: any,
): GameRegistrationListResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayGameRegistrationToApplicationTransform(input_.items)
  }!;
}export function jsonArrayGameRegistrationToTransportTransform(
  items_?: Array<GameRegistration> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonGameRegistrationToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayGameRegistrationToApplicationTransform(
  items_?: any,
): Array<GameRegistration> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonGameRegistrationToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonGameRegistrationToTransportTransform(
  input_?: GameRegistration | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    registration_id: input_.registrationId,game: jsonGameMetadataToTransportTransform(input_.game),build_mode: input_.buildMode,builder_id: input_.builderId,supported_rulesets: jsonArrayStringToTransportTransform(input_.supportedRulesets),source: input_.source,source_id: input_.sourceId
  }!;
}export function jsonGameRegistrationToApplicationTransform(
  input_?: any,
): GameRegistration {
  if(!input_) {
    return input_ as any;
  }
    return {
    registrationId: input_.registration_id,game: jsonGameMetadataToApplicationTransform(input_.game),buildMode: input_.build_mode,builderId: input_.builder_id,supportedRulesets: jsonArrayStringToApplicationTransform(input_.supported_rulesets),source: input_.source,sourceId: input_.source_id
  }!;
}export function jsonGameMetadataToTransportTransform(
  input_?: GameMetadata | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    game_id: input_.gameId,game_version: input_.gameVersion,ruleset_version: input_.rulesetVersion
  }!;
}export function jsonGameMetadataToApplicationTransform(
  input_?: any,
): GameMetadata {
  if(!input_) {
    return input_ as any;
  }
    return {
    gameId: input_.game_id,gameVersion: input_.game_version,rulesetVersion: input_.ruleset_version
  }!;
}export function jsonGameRegistrationRequestToTransportTransform(
  input_?: GameRegistrationRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    registration_id: input_.registrationId,game: jsonGameMetadataToTransportTransform(input_.game)
  }!;
}export function jsonGameRegistrationRequestToApplicationTransform(
  input_?: any,
): GameRegistrationRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    registrationId: input_.registration_id,game: jsonGameMetadataToApplicationTransform(input_.game)
  }!;
}export function jsonFileToTransportTransform(input_?: File | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    contentType: input_.contentType,filename: input_.filename,contents: input_.contents
  }!;
}export function jsonFileToApplicationTransform(input_?: any): File {
  if(!input_) {
    return input_ as any;
  }
    return {
    contentType: input_.contentType,filename: input_.filename,contents: input_.contents
  }!;
}export function jsonAiSubmissionListResponseToTransportTransform(
  input_?: AiSubmissionListResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayAiSubmissionToTransportTransform(input_.items)
  }!;
}export function jsonAiSubmissionListResponseToApplicationTransform(
  input_?: any,
): AiSubmissionListResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayAiSubmissionToApplicationTransform(input_.items)
  }!;
}export function jsonArrayAiSubmissionToTransportTransform(
  items_?: Array<AiSubmission> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonAiSubmissionToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayAiSubmissionToApplicationTransform(
  items_?: any,
): Array<AiSubmission> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonAiSubmissionToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonAiSubmissionToTransportTransform(
  input_?: AiSubmission | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    ai_submission_id: input_.aiSubmissionId,game_registration_id: input_.gameRegistrationId,game: jsonGameMetadataToTransportTransform(input_.game),artifact_ref: input_.artifactRef,display_name: input_.displayName,runtime_kind: input_.runtimeKind,ai_id: input_.aiId,validation_state: input_.validationState,source: input_.source,source_id: input_.sourceId
  }!;
}export function jsonAiSubmissionToApplicationTransform(
  input_?: any,
): AiSubmission {
  if(!input_) {
    return input_ as any;
  }
    return {
    aiSubmissionId: input_.ai_submission_id,gameRegistrationId: input_.game_registration_id,game: jsonGameMetadataToApplicationTransform(input_.game),artifactRef: input_.artifact_ref,displayName: input_.display_name,runtimeKind: input_.runtime_kind,aiId: input_.ai_id,validationState: input_.validation_state,source: input_.source,sourceId: input_.source_id
  }!;
}export function jsonAiSubmissionRequestToTransportTransform(
  input_?: AiSubmissionRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    ai_submission_id: input_.aiSubmissionId,game_registration_id: input_.gameRegistrationId,artifact_ref: input_.artifactRef,display_name: input_.displayName
  }!;
}export function jsonAiSubmissionRequestToApplicationTransform(
  input_?: any,
): AiSubmissionRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    aiSubmissionId: input_.ai_submission_id,gameRegistrationId: input_.game_registration_id,artifactRef: input_.artifact_ref,displayName: input_.display_name
  }!;
}export function jsonMatchRequestListResponseToTransportTransform(
  input_?: MatchRequestListResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayMatchRequestToTransportTransform(input_.items)
  }!;
}export function jsonMatchRequestListResponseToApplicationTransform(
  input_?: any,
): MatchRequestListResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayMatchRequestToApplicationTransform(input_.items)
  }!;
}export function jsonArrayMatchRequestToTransportTransform(
  items_?: Array<MatchRequest> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMatchRequestToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayMatchRequestToApplicationTransform(
  items_?: any,
): Array<MatchRequest> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMatchRequestToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonMatchRequestToTransportTransform(
  input_?: MatchRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    request_id: input_.requestId,game_registration_id: input_.gameRegistrationId,game: jsonGameMetadataToTransportTransform(input_.game),participants: jsonArrayMatchRequestParticipantToTransportTransform(input_.participants),output_dir: input_.outputDir,source: input_.source,source_id: input_.sourceId,match_id: input_.matchId,latest_run_id: input_.latestRunId,official_run_id: input_.officialRunId,lifecycle_state: input_.lifecycleState
  }!;
}export function jsonMatchRequestToApplicationTransform(
  input_?: any,
): MatchRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    requestId: input_.request_id,gameRegistrationId: input_.game_registration_id,game: jsonGameMetadataToApplicationTransform(input_.game),participants: jsonArrayMatchRequestParticipantToApplicationTransform(input_.participants),outputDir: input_.output_dir,source: input_.source,sourceId: input_.source_id,matchId: input_.match_id,latestRunId: input_.latest_run_id,officialRunId: input_.official_run_id,lifecycleState: input_.lifecycle_state
  }!;
}export function jsonArrayMatchRequestParticipantToTransportTransform(
  items_?: Array<MatchRequestParticipant> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMatchRequestParticipantToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayMatchRequestParticipantToApplicationTransform(
  items_?: any,
): Array<MatchRequestParticipant> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMatchRequestParticipantToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonMatchRequestParticipantToTransportTransform(
  input_?: MatchRequestParticipant | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    player_id: input_.playerId,ai_submission_id: input_.aiSubmissionId
  }!;
}export function jsonMatchRequestParticipantToApplicationTransform(
  input_?: any,
): MatchRequestParticipant {
  if(!input_) {
    return input_ as any;
  }
    return {
    playerId: input_.player_id,aiSubmissionId: input_.ai_submission_id
  }!;
}export function jsonMatchRequestCreateRequestToTransportTransform(
  input_?: MatchRequestCreateRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    request_id: input_.requestId,game_registration_id: input_.gameRegistrationId,participants: jsonArrayMatchRequestParticipantToTransportTransform(input_.participants),output_dir: input_.outputDir,match_id: input_.matchId
  }!;
}export function jsonMatchRequestCreateRequestToApplicationTransform(
  input_?: any,
): MatchRequestCreateRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    requestId: input_.request_id,gameRegistrationId: input_.game_registration_id,participants: jsonArrayMatchRequestParticipantToApplicationTransform(input_.participants),outputDir: input_.output_dir,matchId: input_.match_id
  }!;
}export function jsonSignupInviteRequestToTransportTransform(
  input_?: SignupInviteRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    role: input_.role,ttl: input_.ttl
  }!;
}export function jsonSignupInviteRequestToApplicationTransform(
  input_?: any,
): SignupInviteRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    role: input_.role,ttl: input_.ttl
  }!;
}export function jsonSignupInviteResponseToTransportTransform(
  input_?: SignupInviteResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    invite_token: input_.inviteToken,role: input_.role,expires_at: input_.expiresAt,invite_url: input_.inviteUrl
  }!;
}export function jsonSignupInviteResponseToApplicationTransform(
  input_?: any,
): SignupInviteResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    inviteToken: input_.invite_token,role: input_.role,expiresAt: input_.expires_at,inviteUrl: input_.invite_url
  }!;
}export function jsonStoredRankingSnapshotToTransportTransform(
  input_?: StoredRankingSnapshot | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    locator: input_.locator,snapshot: jsonRankingSnapshotToTransportTransform(input_.snapshot)
  }!;
}export function jsonStoredRankingSnapshotToApplicationTransform(
  input_?: any,
): StoredRankingSnapshot {
  if(!input_) {
    return input_ as any;
  }
    return {
    locator: input_.locator,snapshot: jsonRankingSnapshotToApplicationTransform(input_.snapshot)
  }!;
}export function jsonRankingSnapshotToTransportTransform(
  input_?: RankingSnapshot | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    scope: jsonRankingScopeToTransportTransform(input_.scope),applied_run_ids: jsonArrayStringToTransportTransform(input_.appliedRunIds),applied_match_ids: jsonArrayStringToTransportTransform(input_.appliedMatchIds),last_applied_run_id: input_.lastAppliedRunId,last_applied_match_id: input_.lastAppliedMatchId,completed_matches: input_.completedMatches,entries: jsonArrayRankingEntryToTransportTransform(input_.entries)
  }!;
}export function jsonRankingSnapshotToApplicationTransform(
  input_?: any,
): RankingSnapshot {
  if(!input_) {
    return input_ as any;
  }
    return {
    scope: jsonRankingScopeToApplicationTransform(input_.scope),appliedRunIds: jsonArrayStringToApplicationTransform(input_.applied_run_ids),appliedMatchIds: jsonArrayStringToApplicationTransform(input_.applied_match_ids),lastAppliedRunId: input_.last_applied_run_id,lastAppliedMatchId: input_.last_applied_match_id,completedMatches: input_.completed_matches,entries: jsonArrayRankingEntryToApplicationTransform(input_.entries)
  }!;
}export function jsonRankingScopeToTransportTransform(
  input_?: RankingScope | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    game_id: input_.gameId,game_version: input_.gameVersion,ruleset_version: input_.rulesetVersion
  }!;
}export function jsonRankingScopeToApplicationTransform(
  input_?: any,
): RankingScope {
  if(!input_) {
    return input_ as any;
  }
    return {
    gameId: input_.game_id,gameVersion: input_.game_version,rulesetVersion: input_.ruleset_version
  }!;
}export function jsonArrayRankingEntryToTransportTransform(
  items_?: Array<RankingEntry> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonRankingEntryToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayRankingEntryToApplicationTransform(
  items_?: any,
): Array<RankingEntry> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonRankingEntryToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonRankingEntryToTransportTransform(
  input_?: RankingEntry | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    competitor_ref: input_.competitorRef,last_player_id: input_.lastPlayerId,matches_played: input_.matchesPlayed,first_places: input_.firstPlaces,placement_counts: jsonRecordInt32ToTransportTransform(input_.placementCounts),last_run_id: input_.lastRunId,last_match_id: input_.lastMatchId,last_status: input_.lastStatus
  }!;
}export function jsonRankingEntryToApplicationTransform(
  input_?: any,
): RankingEntry {
  if(!input_) {
    return input_ as any;
  }
    return {
    competitorRef: input_.competitor_ref,lastPlayerId: input_.last_player_id,matchesPlayed: input_.matches_played,firstPlaces: input_.first_places,placementCounts: jsonRecordInt32ToApplicationTransform(input_.placement_counts),lastRunId: input_.last_run_id,lastMatchId: input_.last_match_id,lastStatus: input_.last_status
  }!;
}export function jsonRecordInt32ToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordInt32ToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonPresetMatchRequestToTransportTransform(
  input_?: PresetMatchRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    preset_id: input_.presetId,run_id: input_.runId,match_id: input_.matchId,output_dir: input_.outputDir
  }!;
}export function jsonPresetMatchRequestToApplicationTransform(
  input_?: any,
): PresetMatchRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    presetId: input_.preset_id,runId: input_.run_id,matchId: input_.match_id,outputDir: input_.output_dir
  }!;
}export function jsonResultListItemToTransportTransform(
  input_?: ResultListItem | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    run_id: input_.runId,match_id: input_.matchId,attempt_count: input_.attemptCount,official: input_.official,game_id: input_.gameId,game_version: input_.gameVersion,ruleset_version: input_.rulesetVersion,lifecycle_state: input_.lifecycleState,worker_id: input_.workerId,terminal_status: input_.terminalStatus,error: input_.error,turn: input_.turn,placements: jsonArrayPlacementToTransportTransform(input_.placements),result_summary_path: input_.resultSummaryPath
  }!;
}export function jsonResultListItemToApplicationTransform(
  input_?: any,
): ResultListItem {
  if(!input_) {
    return input_ as any;
  }
    return {
    runId: input_.run_id,matchId: input_.match_id,attemptCount: input_.attempt_count,official: input_.official,gameId: input_.game_id,gameVersion: input_.game_version,rulesetVersion: input_.ruleset_version,lifecycleState: input_.lifecycle_state,workerId: input_.worker_id,terminalStatus: input_.terminal_status,error: input_.error,turn: input_.turn,placements: jsonArrayPlacementToApplicationTransform(input_.placements),resultSummaryPath: input_.result_summary_path
  }!;
}export function jsonArrayPlacementToTransportTransform(
  items_?: Array<Placement> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlacementToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayPlacementToApplicationTransform(
  items_?: any,
): Array<Placement> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlacementToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPlacementToTransportTransform(
  input_?: Placement | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    player_id: input_.playerId,place: input_.place
  }!;
}export function jsonPlacementToApplicationTransform(input_?: any): Placement {
  if(!input_) {
    return input_ as any;
  }
    return {
    playerId: input_.player_id,place: input_.place
  }!;
}export function jsonMatchDetailResponseToTransportTransform(
  input_?: MatchDetailResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    players: jsonArraySubmittedPlayerToTransportTransform(input_.players),output_dir: input_.outputDir,match_dir: input_.matchDir,record_path: input_.recordPath,player_stderr_paths: jsonRecordStringToTransportTransform(input_.playerStderrPaths),result_summary: jsonResultSummaryToTransportTransform(input_.resultSummary),replay_inputs: jsonReplayInputsToTransportTransform(input_.replayInputs),artifact_access: jsonRecordArtifactAccessMetadataToTransportTransform(input_.artifactAccess),run_id: input_.runId,match_id: input_.matchId,attempt_count: input_.attemptCount,official: input_.official,game_id: input_.gameId,game_version: input_.gameVersion,ruleset_version: input_.rulesetVersion,lifecycle_state: input_.lifecycleState,worker_id: input_.workerId,terminal_status: input_.terminalStatus,error: input_.error,turn: input_.turn,placements: jsonArrayPlacementToTransportTransform(input_.placements),result_summary_path: input_.resultSummaryPath
  }!;
}export function jsonMatchDetailResponseToApplicationTransform(
  input_?: any,
): MatchDetailResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    players: jsonArraySubmittedPlayerToApplicationTransform(input_.players),outputDir: input_.output_dir,matchDir: input_.match_dir,recordPath: input_.record_path,playerStderrPaths: jsonRecordStringToApplicationTransform(input_.player_stderr_paths),resultSummary: jsonResultSummaryToApplicationTransform(input_.result_summary),replayInputs: jsonReplayInputsToApplicationTransform(input_.replay_inputs),artifactAccess: jsonRecordArtifactAccessMetadataToApplicationTransform(input_.artifact_access),runId: input_.run_id,matchId: input_.match_id,attemptCount: input_.attempt_count,official: input_.official,gameId: input_.game_id,gameVersion: input_.game_version,rulesetVersion: input_.ruleset_version,lifecycleState: input_.lifecycle_state,workerId: input_.worker_id,terminalStatus: input_.terminal_status,error: input_.error,turn: input_.turn,placements: jsonArrayPlacementToApplicationTransform(input_.placements),resultSummaryPath: input_.result_summary_path
  }!;
}export function jsonArraySubmittedPlayerToTransportTransform(
  items_?: Array<SubmittedPlayer> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonSubmittedPlayerToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArraySubmittedPlayerToApplicationTransform(
  items_?: any,
): Array<SubmittedPlayer> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonSubmittedPlayerToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonSubmittedPlayerToTransportTransform(
  input_?: SubmittedPlayer | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    player_id: input_.playerId,artifact_ref: input_.artifactRef
  }!;
}export function jsonSubmittedPlayerToApplicationTransform(
  input_?: any,
): SubmittedPlayer {
  if(!input_) {
    return input_ as any;
  }
    return {
    playerId: input_.player_id,artifactRef: input_.artifact_ref
  }!;
}export function jsonRecordStringToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordStringToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonResultSummaryToTransportTransform(
  input_?: ResultSummary | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    match_id: input_.matchId,game_id: input_.gameId,game_version: input_.gameVersion,ruleset_version: input_.rulesetVersion,status: input_.status,turn: input_.turn,placements: jsonArrayPlacementToTransportTransform(input_.placements),artifact_paths: jsonPathRefsToTransportTransform(input_.artifactPaths),error: input_.error
  }!;
}export function jsonResultSummaryToApplicationTransform(
  input_?: any,
): ResultSummary {
  if(!input_) {
    return input_ as any;
  }
    return {
    matchId: input_.match_id,gameId: input_.game_id,gameVersion: input_.game_version,rulesetVersion: input_.ruleset_version,status: input_.status,turn: input_.turn,placements: jsonArrayPlacementToApplicationTransform(input_.placements),artifactPaths: jsonPathRefsToApplicationTransform(input_.artifact_paths),error: input_.error
  }!;
}export function jsonPathRefsToTransportTransform(
  input_?: PathRefs | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    record: input_.record,structured_log: input_.structuredLog,snapshot: input_.snapshot,exported_snapshot: input_.exportedSnapshot,history: input_.history
  }!;
}export function jsonPathRefsToApplicationTransform(input_?: any): PathRefs {
  if(!input_) {
    return input_ as any;
  }
    return {
    record: input_.record,structuredLog: input_.structured_log,snapshot: input_.snapshot,exportedSnapshot: input_.exported_snapshot,history: input_.history
  }!;
}export function jsonReplayInputsToTransportTransform(
  input_?: ReplayInputs | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    record_path: input_.recordPath,snapshot_path: input_.snapshotPath,history_path: input_.historyPath,exported_snapshot_path: input_.exportedSnapshotPath,verification: jsonVerificationSummaryToTransportTransform(input_.verification)
  }!;
}export function jsonReplayInputsToApplicationTransform(
  input_?: any,
): ReplayInputs {
  if(!input_) {
    return input_ as any;
  }
    return {
    recordPath: input_.record_path,snapshotPath: input_.snapshot_path,historyPath: input_.history_path,exportedSnapshotPath: input_.exported_snapshot_path,verification: jsonVerificationSummaryToApplicationTransform(input_.verification)
  }!;
}export function jsonVerificationSummaryToTransportTransform(
  input_?: VerificationSummary | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    issues: jsonArrayStringToTransportTransform(input_.issues)
  }!;
}export function jsonVerificationSummaryToApplicationTransform(
  input_?: any,
): VerificationSummary {
  if(!input_) {
    return input_ as any;
  }
    return {
    issues: jsonArrayStringToApplicationTransform(input_.issues)
  }!;
}export function jsonRecordArtifactAccessMetadataToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonArtifactAccessMetadataToTransportTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordArtifactAccessMetadataToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonArtifactAccessMetadataToApplicationTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonArtifactAccessMetadataToTransportTransform(
  input_?: ArtifactAccessMetadata | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    locator: input_.locator,download_url: input_.downloadUrl,issuer: input_.issuer,status: input_.status,expires_at: input_.expiresAt
  }!;
}export function jsonArtifactAccessMetadataToApplicationTransform(
  input_?: any,
): ArtifactAccessMetadata {
  if(!input_) {
    return input_ as any;
  }
    return {
    locator: input_.locator,downloadUrl: input_.download_url,issuer: input_.issuer,status: input_.status,expiresAt: input_.expires_at
  }!;
}export function jsonRunListResponseToTransportTransform(
  input_?: RunListResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayResultListItemToTransportTransform(input_.items)
  }!;
}export function jsonRunListResponseToApplicationTransform(
  input_?: any,
): RunListResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    items: jsonArrayResultListItemToApplicationTransform(input_.items)
  }!;
}export function jsonArrayResultListItemToTransportTransform(
  items_?: Array<ResultListItem> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonResultListItemToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayResultListItemToApplicationTransform(
  items_?: any,
): Array<ResultListItem> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonResultListItemToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}
