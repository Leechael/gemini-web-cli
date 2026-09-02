import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { GeminiWeb } from "../index.ts";

function modelResponse() {
  return Response.json({
    object: "list",
    data: [{ id: "gemini-3.5-flash", object: "model", owned_by: "google" }],
  });
}

function harness() {
  const providers: Array<{ name: string; config: Record<string, unknown> }> = [];
  const tools: string[] = [];
  const handlers = new Map<string, (event: any, ctx: any) => Promise<void>>();
  const api = {
    registerProvider(name: string, config: Record<string, unknown>) {
      providers.push({ name, config });
    },
    unregisterProvider() {},
    registerTool(tool: { name: string }) {
      tools.push(tool.name);
    },
    on(name: string, handler: (event: any, ctx: any) => Promise<void>) {
      handlers.set(name, handler);
    },
  } as unknown as ExtensionAPI;
  return { api, providers, tools, handlers };
}

test("registers discovered models and research tools from global configuration", async () => {
  const agentDir = mkdtempSync(join(tmpdir(), "gemini-web-extension-"));
  mkdirSync(join(agentDir, "extensions"), { recursive: true });
  writeFileSync(
    join(agentDir, "extensions", "pi-provider-gemini-web.json"),
    '{"baseUrl":"http://global.internal:8080"}\n',
  );
  const h = harness();

  await GeminiWeb({ agentDir, fetch: async () => modelResponse() })(h.api);

  assert.equal(h.providers[0].name, "gemini-web");
  assert.equal(h.providers[0].config.baseUrl, "http://global.internal:8080/v1");
  assert.equal(h.providers[0].config.authHeader, undefined);
  assert.deepEqual(h.tools, [
    "gemini_research_create",
    "gemini_research_status",
    "gemini_research_result",
  ]);
});

test("malformed global configuration does not disable tools or session recovery", async () => {
  const agentDir = mkdtempSync(join(tmpdir(), "gemini-web-extension-"));
  mkdirSync(join(agentDir, "extensions"), { recursive: true });
  writeFileSync(join(agentDir, "extensions", "pi-provider-gemini-web.json"), "not json");
  const h = harness();
  const originalError = console.error;
  console.error = () => {};
  try {
    await GeminiWeb({ agentDir, fetch: async () => modelResponse() })(h.api);
  } finally {
    console.error = originalError;
  }

  assert.deepEqual(h.tools, [
    "gemini_research_create",
    "gemini_research_status",
    "gemini_research_result",
  ]);
  assert.equal(h.handlers.has("session_start"), true);
});

test("trusted project-only configuration registers during session start", async () => {
  const root = mkdtempSync(join(tmpdir(), "gemini-web-extension-"));
  const agentDir = join(root, "agent");
  const cwd = join(root, "project");
  mkdirSync(join(cwd, ".pi"), { recursive: true });
  writeFileSync(
    join(cwd, ".pi", "pi-provider-gemini-web.json"),
    '{"baseUrl":"http://project.internal:8080"}\n',
  );
  const h = harness();
  await GeminiWeb({ agentDir, cwd, fetch: async () => modelResponse() })(h.api);

  assert.equal(h.providers.length, 1);
  assert.deepEqual(
    (h.providers[0].config.models as Array<{ id: string }>).map((model) => model.id),
    ["gemini-3.5-flash", "gemini-3.1-pro", "gemini-3.1-flash-lite"],
  );
  await h.handlers.get("session_start")?.(
    { reason: "startup" },
    {
      cwd,
      isProjectTrusted: () => true,
      sessionManager: { getSessionId: () => "session-1" },
      ui: { notify() {} },
    },
  );

  assert.equal(h.providers.at(-1)?.config.baseUrl, "http://project.internal:8080/v1");
});
