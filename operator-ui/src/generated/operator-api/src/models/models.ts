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
export interface GameRegistrationListResponse {
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
/**
 * A file in an HTTP request, response, or multipart payload.
 *
 * Files have a special meaning that the HTTP library understands. When the body of an HTTP request, response,
 * or multipart payload is _effectively_ an instance of `TypeSpec.Http.File` or any type that extends it, the
 * operation is treated as a file upload or download.
 *
 * When using file bodies, the fields of the file model are defined to come from particular locations by default:
 *
 * - `contentType`: The `Content-Type` header of the request, response, or multipart payload (CANNOT be overridden or changed).
 * - `contents`: The body of the request, response, or multipart payload (CANNOT be overridden or changed).
 * - `filename`: The `filename` parameter value of the `Content-Disposition` header of the response or multipart payload
 * (MAY be overridden or changed).
 *
 * A File may be used as a normal structured JSON object in a request or response, if the request specifies an explicit
 * `Content-Type` header. In this case, the entire File model is serialized as if it were any other model. In a JSON payload,
 * it will have a structure like:
 *
 * ```
 * {
 *   "contentType": <string?>,
 *   "filename": <string?>,
 *   "contents": <string, base64>
 * }
 * ```
 *
 * The `contentType` _within_ the file defines what media types the data inside the file can be, but if the specification
 * defines a `Content-Type` for the payload as HTTP metadata, that `Content-Type` metadata defines _how the file is
 * serialized_. See the examples below for more information.
 *
 * NOTE: The `filename` and `contentType` fields are optional. Furthermore, the default location of `filename`
 * (`Content-Disposition: <disposition>; filename=<filename>`) is only valid in HTTP responses and multipart payloads. If
 * you wish to send the `filename` in a request, you must use HTTP metadata decorators to describe the location of the
 * `filename` field. You can combine the metadata decorators with `@visibility` to control when the `filename` location
 * is overridden, as shown in the examples below.
 */
export interface File {
  /**
   * The allowed media (MIME) types of the file contents.
   *
   * In file bodies, this value comes from the `Content-Type` header of the request or response. In JSON bodies,
   * this value is serialized as a field in the response.
   *
   * NOTE: this is not _necessarily_ the same as the `Content-Type` header of the request or response, but
   * it will be for file bodies. It may be different if the file is serialized as a JSON object. It always refers to the
   * _contents_ of the file, and not necessarily the way the file itself is transmitted or serialized.
   */
  contentType?: string;
  /**
   * The name of the file, if any.
   *
   * In file bodies, this value comes from the `filename` parameter of the `Content-Disposition` header of the response
   * or multipart payload. In JSON bodies, this value is serialized as a field in the response.
   *
   * NOTE: By default, `filename` cannot be sent in request payloads and can only be sent in responses and multipart
   * payloads, as the `Content-Disposition` header is not valid in requests. If you want to send the `filename` in a request,
   * you must extend the `File` model and override the `filename` property with a different location defined by HTTP metadata
   * decorators.
   */
  filename?: string;
  /**
   * The contents of the file.
   *
   * In file bodies, this value comes from the body of the request, response, or multipart payload. In JSON bodies,
   * this value is serialized as a field in the response.
   */
  contents: Uint8Array;
}
/**
 * Represent a byte array
 */
export type Bytes = Uint8Array;
export interface GameBundleAdmission {
  gameId: string;
  gameVersion: string;
  artifactId: string;
  buildMode: string;
  builderId: string;
  supportedRulesets: Array<string>;
}
export interface AiSubmissionListResponse {
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
export interface MatchRequestListResponse {
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
  lifecycleState:
    | "queued"
    | "leased"
    | "running"
    | "persisting"
    | "completed"
    | "failed"
    | "canceled";
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
  lifecycleState:
    | "queued"
    | "leased"
    | "running"
    | "persisting"
    | "completed"
    | "failed"
    | "canceled";
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
export interface RunListResponse {
  items: Array<ResultListItem>;
}
