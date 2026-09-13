import "@typespec/http";
import "@typespec/rest";

import "../shared.tsp";

using Http;
using Rest;
using AiArena.Shared;

namespace AiArena.Public;

alias PublicStateAvailability = "available" | "state_unavailable";
alias PublicReplayAvailability = "available" | "replay_unavailable";

model PublicMatch {
  match_id: string;
  selected_run_id: string;
  game: GameMetadata;
  lifecycle_state: LifecycleState;
}

model PublicMatchListResponse {
  items: PublicMatch[];
}

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
op listMatches(): PublicMatchListResponse;

@route("/api/v1-alpha/public/matches/{match_id}")
@get
op getMatch(@path match_id: string): PublicMatchDetail;

@route("/api/v1-alpha/public/matches/{match_id}/state")
@get
op getLatestState(@path match_id: string): PublicStateResponse;

@route("/api/v1-alpha/public/matches/{match_id}/replay")
@get
op getReplay(@path match_id: string): PublicReplayResponse;
