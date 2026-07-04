import "@typespec/http";
import "@typespec/rest";

import "../shared.tsp";

using Http;
using Rest;
using AiArena.Shared;

namespace AiArena.Operator;

@route("/auth/session")
@get
op session(): SessionStatusResponse;

@route("/auth/logout")
@post
op logout(): void;
