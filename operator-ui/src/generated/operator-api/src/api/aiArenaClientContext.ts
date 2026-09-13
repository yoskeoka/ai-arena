import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface AiArenaClientContext extends Client {}
export interface AiArenaClientOptions extends ClientOptions {
  endpoint?: string;
}
export function createAiArenaClientContext(
  endpoint: string,
  options?: AiArenaClientOptions,
): AiArenaClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint,
  };
  const resolvedEndpoint = "https://{endpoint}".replace(
    /{([^}]+)}/g,
    (_, key) =>
      key in params
        ? String(params[key])
        : (() => {
            throw new Error(`Missing parameter: ${key}`);
          })(),
  );
  return getClient(resolvedEndpoint, {
    ...options,
  });
}
