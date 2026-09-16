#!/usr/bin/env node

import { readFile, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const aiArenaContextImport = `import {
  type AiArenaClientContext,
  type AiArenaClientOptions,
  createAiArenaClientContext,
} from "./api/aiArenaClientContext.js";
`;
const sharedContextImport = `import {
  createSharedClientContext,
  type SharedClientContext,
  type SharedClientOptions,
} from "./api/sharedClient/sharedClientContext.js";
`;

function fail(message) {
  throw new Error(`normalize-empty-client-context: ${message}`);
}

function replaceOnce(source, target, replacement, description) {
  const first = source.indexOf(target);
  if (first === -1) {
    fail(`expected exactly one ${description}, found none`);
  }
  if (source.indexOf(target, first + target.length) !== -1) {
    fail(`expected exactly one ${description}, found multiple`);
  }
  return `${source.slice(0, first)}${replacement}${source.slice(first + target.length)}`;
}

function assertOnce(source, target, description) {
  const first = source.indexOf(target);
  if (first === -1 || source.indexOf(target, first + target.length) !== -1) {
    fail(`expected exactly one ${description}`);
  }
}

export function normalizeEmptyClientContexts(source) {
  assertOnce(source, "export class AiArenaClient {", "AiArenaClient class");
  assertOnce(source, "export class SharedClient {", "SharedClient class");
  assertOnce(source, "export class OperatorClient {", "OperatorClient class");
  assertOnce(source, "export class PublicClient {", "PublicClient class");

  let normalized = replaceOnce(
    source,
    aiArenaContextImport,
    `import type { AiArenaClientOptions } from "./api/aiArenaClientContext.js";\n`,
    "AiArenaClient context import",
  );
  normalized = replaceOnce(
    normalized,
    "export class AiArenaClient {\n  #context: AiArenaClientContext\n",
    "export class AiArenaClient {\n",
    "AiArenaClient context field",
  );
  normalized = replaceOnce(
    normalized,
    "    this.#context = createAiArenaClientContext(endpoint, options);\n",
    "",
    "AiArenaClient context initializer",
  );
  normalized = replaceOnce(normalized, sharedContextImport, "", "SharedClient context import");
  normalized = replaceOnce(
    normalized,
    `export class SharedClient {
  #context: SharedClientContext
  constructor(endpoint: string, options?: SharedClientOptions) {
    this.#context = createSharedClientContext(endpoint, options);

  }
}
`,
    `export class SharedClient {
  constructor(_endpoint: string, _options?: AiArenaClientOptions) {}
}
`,
    "empty SharedClient class body",
  );

  for (const token of [
    "AiArenaClientContext",
    "createAiArenaClientContext",
    "SharedClientContext",
    "SharedClientOptions",
    "createSharedClientContext",
    "this.#context = createAiArenaClientContext",
    "this.#context = createSharedClientContext",
  ]) {
    if (normalized.includes(token)) {
      fail(`target token '${token}' remained after normalization`);
    }
  }
  return normalized;
}

export async function normalizeFile(path) {
  const source = await readFile(path, "utf8");
  const normalized = normalizeEmptyClientContexts(source);
  await writeFile(path, normalized);
}

const invokedPath = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invokedPath) {
  const [path] = process.argv.slice(2);
  if (!path || process.argv.length !== 3) {
    fail("usage: normalize-empty-client-context.mjs <aiArenaClient.ts>");
  }
  normalizeFile(path).catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
