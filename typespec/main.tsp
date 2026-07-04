import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./namespaces/shared.tsp";
import "./namespaces/operator/health.tsp";
import "./namespaces/operator/auth.tsp";
import "./namespaces/operator/api.tsp";
import "./namespaces/public/api.tsp";

using Http;
using Rest;

@service(#{ title: "AI Arena Operator API" })
@server("https://arena-service.example.com", "Operator API origin")
namespace AiArena;
