import "@typespec/http";
import "@typespec/rest";

import "../shared.tsp";

using Http;
using Rest;
using AiArena.Shared;

namespace AiArena.Operator;

@route("/healthz")
@get
op healthz(): HealthResponse;
