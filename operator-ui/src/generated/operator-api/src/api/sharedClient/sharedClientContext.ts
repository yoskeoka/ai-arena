import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface SharedClientContext extends Client {

}export interface SharedClientOptions extends ClientOptions {
  endpoint?: string;
}export function createSharedClientContext(
  endpoint: string,
  options?: SharedClientOptions,
): SharedClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "https://{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
