import assert from "node:assert/strict";
import test from "node:test";

import { GeminiWebResearchClient } from "../src/research.ts";

test("creates, checks, and retrieves deep research through the configured server", async () => {
  const requests: Array<{ url: string; method: string; body?: string }> = [];
  const client = new GeminiWebResearchClient("http://gemini.internal:8080", async (input, init) => {
    requests.push({
      url: String(input),
      method: init?.method ?? "GET",
      body: typeof init?.body === "string" ? init.body : undefined,
    });
    const url = String(input);
    if (url.endsWith("/v1/research")) {
      return Response.json(
        { id: "c_research", chat_id: "c_research", title: "Topic" },
        { status: 201 },
      );
    }
    if (url.endsWith("/status")) {
      return Response.json({ id: "c_research", chat_id: "c_research", state: "running" });
    }
    return Response.json({
      id: "c_research",
      chat_id: "c_research",
      text: "Report",
      sources: [{ url: "https://example.com", title: "Example" }],
    });
  });

  assert.equal((await client.create("Investigate", "gemini-3.5-flash")).id, "c_research");
  assert.equal((await client.status("c_research")).state, "running");
  assert.equal((await client.result("c_research")).text, "Report");
  assert.deepEqual(requests, [
    {
      url: "http://gemini.internal:8080/v1/research",
      method: "POST",
      body: '{"prompt":"Investigate","model":"gemini-3.5-flash"}',
    },
    {
      url: "http://gemini.internal:8080/v1/research/c_research/status",
      method: "GET",
      body: undefined,
    },
    {
      url: "http://gemini.internal:8080/v1/research/c_research/result",
      method: "GET",
      body: undefined,
    },
  ]);
});

test("surfaces server errors without exposing an unbounded response", async () => {
  const client = new GeminiWebResearchClient(
    "http://gemini.internal:8080",
    async () => new Response("x".repeat(1000), { status: 500 }),
  );

  await assert.rejects(client.status("c_failed"), (error: unknown) => {
    assert.ok(error instanceof Error);
    assert.match(error.message, /HTTP 500/);
    assert.ok(error.message.length < 700);
    return true;
  });
});
