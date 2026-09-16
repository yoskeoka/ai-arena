import assert from "node:assert/strict";
import test from "node:test";

import { normalizeEmptyClientContexts } from "./normalize-empty-client-context.mjs";

const fixture = `import {
  type AiArenaClientContext,
  type AiArenaClientOptions,
  createAiArenaClientContext,
} from "./api/aiArenaClientContext.js";
import {
  createOperatorClientContext,
  type OperatorClientContext,
  type OperatorClientOptions,
} from "./api/operatorClient/operatorClientContext.js";
import {
  createSharedClientContext,
  type SharedClientContext,
  type SharedClientOptions,
} from "./api/sharedClient/sharedClientContext.js";
export class AiArenaClient {
  #context: AiArenaClientContext
  sharedClient: SharedClient;
  constructor(endpoint: string, options?: AiArenaClientOptions) {
    this.#context = createAiArenaClientContext(endpoint, options);
    this.sharedClient = new SharedClient(endpoint, options);
  }
}
export class OperatorClient {
  #context: OperatorClientContext
  constructor(endpoint: string, options?: OperatorClientOptions) {
    this.#context = createOperatorClientContext(endpoint, options);
  }
}
export class PublicClient {}
export class SharedClient {
  #context: SharedClientContext
  constructor(endpoint: string, options?: SharedClientOptions) {
    this.#context = createSharedClientContext(endpoint, options);

  }
}
`;

test("normalizes only the known empty client contexts", () => {
  const normalized = normalizeEmptyClientContexts(fixture);
  assert.doesNotMatch(normalized, /AiArenaClientContext|SharedClientContext|createAiArenaClientContext/);
  assert.match(normalized, /AiArenaClientOptions/);
  assert.match(normalized, /SharedClientOptions/);
  assert.match(normalized, /createOperatorClientContext/);
  assert.match(normalized, /constructor\(_endpoint: string, _options\?: SharedClientOptions\) \{\}/);
});

for (const [name, invalid] of [
  ["missing initializer", fixture.replace("    this.#context = createSharedClientContext(endpoint, options);\n", "")],
  ["duplicate initializer", fixture.replace("    this.#context = createSharedClientContext(endpoint, options);", "    this.#context = createSharedClientContext(endpoint, options);\n    this.#context = createSharedClientContext(endpoint, options);")],
  ["AiArenaClient operation context use", fixture.replace("    this.sharedClient = new SharedClient(endpoint, options);", "    this.sharedClient = new SharedClient(endpoint, options);\n    return this.#context;")],
  ["already-fixed output", normalizeEmptyClientContexts(fixture)],
]) {
  test(`fails closed for ${name}`, () => {
    assert.throws(() => normalizeEmptyClientContexts(invalid), /normalize-empty-client-context/);
  });
}
