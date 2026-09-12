import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface PublicClientContext extends Client {

}export interface PublicClientOptions extends ClientOptions {
  endpoint?: string;
}export function createPublicClientContext(
  endpoint: string,
  options?: PublicClientOptions,
): PublicClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "https://{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
