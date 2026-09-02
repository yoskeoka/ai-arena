namespace AiArena.Shared;

alias AuthMode = "disabled" | "enabled";
alias LifecycleState = "queued" | "leased" | "running" | "persisting" | "completed" | "failed" | "canceled";
alias RegistrationSource = "manual" | "preset";
alias RunKind = "initial" | "retry" | "rerun";
alias SignupInviteRole = "participant" | "developer" | "operator";
alias ValidationState = "ready";

model GameMetadata {
  game_id: string;
  game_version: string;
  ruleset_version: string;
}

model Placement {
  player_id: string;
  place: int32;
}

model PathRefs {
  record: string;
  structured_log: string;
  snapshot: string;
  exported_snapshot: string;
  history: string;
}

model ResultSummary {
  match_id: string;
  game_id: string;
  game_version: string;
  ruleset_version: string;
  status: string;
  turn: int32;
  placements?: Placement[];
  artifact_paths: PathRefs;
  error?: string;
}

model VerificationSummary {
  issues: string[];
}

model ReplayInputs {
  record_path?: string;
  snapshot_path?: string;
  history_path?: string;
  exported_snapshot_path?: string;
  verification?: VerificationSummary;
}

model SubmittedPlayer {
  player_id: string;
  artifact_ref: string;
}

model ArtifactAccessMetadata {
  locator: string;
  download_url?: string;
  issuer?: string;
  status?: string;
  expires_at?: string;
}

model ResultListItem {
  run_id: string;
  match_id: string;
  attempt_count: int32;
  official: boolean;
  game_id: string;
  game_version: string;
  ruleset_version: string;
  lifecycle_state: LifecycleState;
  worker_id?: string;
  terminal_status?: string;
  error?: string;
  turn?: int32;
  placements?: Placement[];
  result_summary_path?: string;
}

model MatchDetailResponse extends ResultListItem {
  players: SubmittedPlayer[];
  output_dir: string;
  match_dir?: string;
  record_path?: string;
  player_stderr_paths?: Record<string>;
  result_summary?: ResultSummary;
  replay_inputs?: ReplayInputs;
  artifact_access?: Record<ArtifactAccessMetadata>;
}

model AuthPrincipal {
  account_id: string;
  provider: string;
  provider_login: string;
  provider_email?: string;
  roles: string[];
}

model SessionStatusResponse {
  auth_mode: AuthMode;
  authenticated: boolean;
  principal?: AuthPrincipal;
}

model HealthResponse {
  status: string;
}

model SignupInviteRequest {
  role: SignupInviteRole;
  ttl?: string;
}

model SignupInviteResponse {
  invite_token: string;
  role: SignupInviteRole;
  expires_at: string;
  invite_url: string;
}

model GameRegistrationRequest {
  registration_id?: string;
  game?: GameMetadata;
  artifact_id?: string;
  ruleset_version?: string;
}

model GameRegistration {
  registration_id: string;
  game: GameMetadata;
  artifact_id?: string;
  player_count?: int32;
  max_active_bots_per_owner?: int32;
  build_mode: string;
  builder_id: string;
  supported_rulesets: string[];
  source?: RegistrationSource;
  source_id?: string;
}

model GameBundleAdmission {
  game_id: string;
  game_version: string;
  artifact_id: string;
  build_mode: string;
  builder_id: string;
  supported_rulesets: string[];
}

model AISubmissionRequest {
  ai_submission_id?: string;
  game_registration_id: string;
  artifact_ref: string;
  display_name?: string;
}

model BotRevisionRequest {
  scope_id: string;
  bot_id?: string;
  bot_name?: string;
  artifact_id: string;
}

model AIBot {
  bot_id: string;
  scope_id: string;
  bot_name: string;
  lifecycle_state: "active" | "retired";
  active_submission_id?: string;
}

model AIBotListResponse {
  items: AIBot[];
}

model EligibleBot {
  bot_id: string;
  scope_id: string;
  bot_name: string;
  active_submission_id: string;
}

model EligibleBotListResponse {
  items: EligibleBot[];
}

model AISubmissionRevision {
  ai_submission_id: string;
  bot_id: string;
  artifact_id: string;
  validation_state: ValidationState;
  created_at: string;
}

model BotRevisionResponse {
  bot: AIBot;
  revision: AISubmissionRevision;
}

model AISubmission {
  ai_submission_id: string;
  game_registration_id: string;
  game: GameMetadata;
  artifact_ref: string;
  artifact_id?: string;
  display_name: string;
  runtime_kind: string;
  ai_id: string;
  validation_state: ValidationState;
  source?: RegistrationSource;
  source_id?: string;
}

model MatchRequestParticipant {
  player_id: string;
  bot_id: string;
  ai_submission_id: string;
}

model MatchRequestCreateRequest {
  scope_id: string;
  bot_ids: string[];
}

model MatchRequest {
  request_id: string;
  game_registration_id: string;
  game: GameMetadata;
  participants: MatchRequestParticipant[];
  output_dir: string;
  source?: RegistrationSource;
  source_id?: string;
  match_id: string;
  latest_run_id: string;
  official_run_id?: string;
  lifecycle_state: LifecycleState;
}

model PresetMatchRequest {
  preset_id: string;
  run_id?: string;
  match_id?: string;
  output_dir?: string;
}

model RankingScope {
  game_id: string;
  game_version: string;
  ruleset_version: string;
}

model RankingEntry {
  competitor_ref: string;
  last_player_id: string;
  matches_played: int32;
  first_places: int32;
  placement_counts?: Record<int32>;
  last_run_id: string;
  last_match_id: string;
  last_status: string;
}

model RankingSnapshot {
  scope: RankingScope;
  applied_run_ids?: string[];
  applied_match_ids?: string[];
  last_applied_run_id?: string;
  last_applied_match_id?: string;
  completed_matches: int32;
  entries?: RankingEntry[];
}

model StoredRankingSnapshot {
  locator: string;
  snapshot: RankingSnapshot;
}

model GameRegistrationListResponse {
  items: GameRegistration[];
}

model AISubmissionListResponse {
  items: AISubmission[];
}

model MatchRequestListResponse {
  items: MatchRequest[];
}

model RunListResponse {
  items: ResultListItem[];
}
