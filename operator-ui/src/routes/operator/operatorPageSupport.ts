export type LoadState = "idle" | "loading" | "ready" | "error";
export type EnqueueState = "idle" | "submitting" | "success" | "error";

export function defaultBaseUrl() {
  const envValue = import.meta.env.VITE_OPERATOR_API_BASE_URL;
  if (typeof envValue === "string" && envValue.trim() !== "") {
    return normalizeBaseUrl(envValue);
  }
  if (typeof window !== "undefined" && window.location.hostname === "localhost") {
    return "";
  }
  return "";
}

export function normalizeBaseUrl(baseUrl: string) {
  const trimmed = baseUrl.trim();
  if (trimmed === "") {
    return "";
  }
  if (typeof window !== "undefined" && window.location.hostname === "localhost") {
    if (trimmed === "http://127.0.0.1:10000" || trimmed === "http://localhost:10000") {
      return "";
    }
  }
  return trimmed;
}

export function messageOf(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}

export function hintFor(error?: string) {
  if (!error) {
    return undefined;
  }
  const normalized = error.toLowerCase();
  if (normalized.includes("failed to fetch") || normalized.includes("err_connection_refused")) {
    return "For local dev, start arena-service first and verify `curl http://127.0.0.1:10000/healthz` returns 200. Leaving the base URL blank uses the local Vite proxy.";
  }
  return undefined;
}
