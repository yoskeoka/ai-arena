/**
 * A sequence of textual characters.
 */
export type String = string;
export interface HealthResponse {
  status: string;
}
export interface SessionStatusResponse {
  authMode: "disabled" | "enabled";
  authenticated: boolean;
  principal?: AuthPrincipal;
}
/**
 * Boolean with `true` and `false` values.
 */
export type Boolean = boolean;
export interface AuthPrincipal {
  accountId: string;
  provider: string;
  providerLogin: string;
  providerEmail?: string;
  roles: Array<string>;
}
export interface ListResponse {
  items: Array<GameRegistration>;
}
export interface GameRegistration {
  registrationId: string;
  game: GameMetadata;
  buildMode: string;
  builderId: string;
  supportedRulesets: Array<string>;
  source?: "manual" | "preset";
  sourceId?: string;
}
export interface GameMetadata {
  gameId: string;
  gameVersion: string;
  rulesetVersion: string;
}
export interface GameRegistrationRequest {
  registrationId?: string;
  game: GameMetadata;
}
export interface ListResponse_2 {
  items: Array<AiSubmission>;
}
export interface AiSubmission {
  aiSubmissionId: string;
  gameRegistrationId: string;
  game: GameMetadata;
  artifactRef: string;
  displayName: string;
  runtimeKind: string;
  aiId: string;
  validationState: "ready";
  source?: "manual" | "preset";
  sourceId?: string;
}
export interface AiSubmissionRequest {
  aiSubmissionId?: string;
  gameRegistrationId: string;
  artifactRef: string;
  displayName?: string;
}
export interface ListResponse_3 {
  items: Array<MatchRequest>;
}
export interface MatchRequest {
  requestId: string;
  gameRegistrationId: string;
  game: GameMetadata;
  participants: Array<MatchRequestParticipant>;
  outputDir: string;
  source?: "manual" | "preset";
  sourceId?: string;
  matchId: string;
  latestRunId: string;
  officialRunId?: string;
  lifecycleState: "queued" | "leased" | "running" | "persisting" | "completed" | "failed" | "canceled";
}
export interface MatchRequestParticipant {
  playerId: string;
  aiSubmissionId: string;
}
export interface MatchRequestCreateRequest {
  requestId?: string;
  gameRegistrationId: string;
  participants: Array<MatchRequestParticipant>;
  outputDir: string;
  matchId?: string;
}
export interface SignupInviteRequest {
  role: "participant" | "developer" | "operator";
  ttl?: string;
}
export interface SignupInviteResponse {
  inviteToken: string;
  role: "participant" | "developer" | "operator";
  expiresAt: string;
  inviteUrl: string;
}
export interface StoredRankingSnapshot {
  locator: string;
  snapshot: RankingSnapshot;
}
export interface RankingSnapshot {
  scope: RankingScope;
  appliedRunIds?: Array<string>;
  appliedMatchIds?: Array<string>;
  lastAppliedRunId?: string;
  lastAppliedMatchId?: string;
  completedMatches: number;
  entries?: Array<RankingEntry>;
}
export interface RankingScope {
  gameId: string;
  gameVersion: string;
  rulesetVersion: string;
}
/**
 * A 32-bit integer. (`-2,147,483,648` to `2,147,483,647`)
 */
export type Int32 = number;
/**
 * A 64-bit integer. (`-9,223,372,036,854,775,808` to `9,223,372,036,854,775,807`)
 */
export type Int64 = bigint;
/**
 * A whole number. This represent any `integer` value possible.
 * It is commonly represented as `BigInteger` in some languages.
 */
export type Integer = number;
/**
 * A numeric type
 */
export type Numeric = number;
export interface RankingEntry {
  competitorRef: string;
  lastPlayerId: string;
  matchesPlayed: number;
  firstPlaces: number;
  placementCounts?: Record<string, number>;
  lastRunId: string;
  lastMatchId: string;
  lastStatus: string;
}
export interface PresetMatchRequest {
  presetId: string;
  runId?: string;
  matchId?: string;
  outputDir?: string;
}
export interface ResultListItem {
  runId: string;
  matchId: string;
  attemptCount: number;
  official: boolean;
  gameId: string;
  gameVersion: string;
  rulesetVersion: string;
  lifecycleState: "queued" | "leased" | "running" | "persisting" | "completed" | "failed" | "canceled";
  workerId?: string;
  terminalStatus?: string;
  error?: string;
  turn?: number;
  placements?: Array<Placement>;
  resultSummaryPath?: string;
}
export interface Placement {
  playerId: string;
  place: number;
}
export interface MatchDetailResponse extends ResultListItem {
  players: Array<SubmittedPlayer>;
  outputDir: string;
  matchDir?: string;
  recordPath?: string;
  playerStderrPaths?: Record<string, string>;
  resultSummary?: ResultSummary;
  replayInputs?: ReplayInputs;
  artifactAccess?: Record<string, ArtifactAccessMetadata>;
}
export interface SubmittedPlayer {
  playerId: string;
  artifactRef: string;
}
export interface ResultSummary {
  matchId: string;
  gameId: string;
  gameVersion: string;
  rulesetVersion: string;
  status: string;
  turn: number;
  placements?: Array<Placement>;
  artifactPaths: PathRefs;
  error?: string;
}
export interface PathRefs {
  record: string;
  structuredLog: string;
  snapshot: string;
  exportedSnapshot: string;
  history: string;
}
export interface ReplayInputs {
  recordPath?: string;
  snapshotPath?: string;
  historyPath?: string;
  exportedSnapshotPath?: string;
  verification?: VerificationSummary;
}
export interface VerificationSummary {
  issues: Array<string>;
}
export interface ArtifactAccessMetadata {
  locator: string;
  downloadUrl?: string;
  issuer?: string;
  status?: string;
  expiresAt?: string;
}
export interface ListResponse_4 {
  items: Array<ResultListItem>;
}
