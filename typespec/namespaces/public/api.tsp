import "@typespec/http";
import "@typespec/rest";

import "../shared.tsp";

using Http;
using Rest;
using AiArena.Shared;

namespace AiArena.Public;

alias PublicStateAvailability = "available" | "state_unavailable";
alias PublicReplayAvailability = "available" | "replay_unavailable";

model PublicParticipant {
  // Entries retain the submitted player sequence; positional meaning belongs to the game.
  player_id: string;
  display_name: string;
  ai_submission_id: string;
}

model PublicMatch {
  match_id: string;
  selected_run_id: string;
  game: GameMetadata;
  lifecycle_state: LifecycleState;
  // The selected run's participant sequence is identical in list, detail, and state views.
  participants?: PublicParticipant[];
  completed_at?: utcDateTime;
}

model PublicMatchListResponse {
  pagination: PublicMatchPagination;
  available_ruleset_versions: string[];
  items: PublicMatch[];
}

model PublicMatchPagination {
  page: int32;
  limit: int32;
  total: int32;
  total_pages: int32;
}

alias PublicMatchSort = "completed_at";
alias PublicMatchSortOrder = "asc" | "desc";

model PublicReplayMetadata {
  availability: PublicReplayAvailability;
  format?: string;
  version?: string;
  size_bytes?: int64;
  digest?: string;
}

model PublicMatchDetail extends PublicMatch {
  replay: PublicReplayMetadata;
}

model PublicStateResponse extends PublicMatch {
  availability: PublicStateAvailability;
  state_version?: int64;
  turn?: int32;
  public_state?: unknown;
  retry_after_ms: int32;
}

model PublicReplayResponse {
  availability: PublicReplayAvailability;
  format?: string;
  version?: string;
  payload?: unknown;
}

@route("/api/v1-alpha/public/matches")
@get
op listMatches(
  @query game_id?: string,
  @query @minValue(1) game_version_major?: int32,
  @query ruleset_version?: string,
  @query @minValue(1) page?: int32,
  @query @minValue(1) @maxValue(100) limit?: int32,
  @query sort?: PublicMatchSort,
  @query sort_order?: PublicMatchSortOrder,
): PublicMatchListResponse;

@route("/api/v1-alpha/public/matches/{match_id}")
@get
op getMatch(@path match_id: string): PublicMatchDetail;

@route("/api/v1-alpha/public/matches/{match_id}/state")
@get
op getLatestState(@path match_id: string): PublicStateResponse;

@route("/api/v1-alpha/public/matches/{match_id}/replay")
@get
op getReplay(@path match_id: string): PublicReplayResponse;
