import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface OperatorClientContext extends Client {}
export interface OperatorClientOptions extends ClientOptions {
  endpoint?: string;
}
export function createOperatorClientContext(
  endpoint: string,
  options?: OperatorClientOptions,
): OperatorClientContext {
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
