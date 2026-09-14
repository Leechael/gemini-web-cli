import assert from "node:assert/strict";
import test from "node:test";

import { GeminiWebNotebookClient } from "../src/notebook.ts";

test("covers the notebook lifecycle through the configured server", async () => {
  const requests: Array<{ url: string; method: string; body?: string }> = [];
  const client = new GeminiWebNotebookClient("http://gemini.internal:8080", async (input, init) => {
    const url = String(input);
    requests.push({
      url,
      method: init?.method ?? "GET",
      body: typeof init?.body === "string" ? init.body : undefined,
    });
    if (url.endsWith("/v1/notebooks")) {
      return Response.json({ resource: "notebooks/nb-1", title: "My Project" });
    }
    if (url.endsWith("/chats")) {
      return Response.json({ chats: [{ cid: "c_1", title: "First chat" }] });
    }
    if (url.endsWith("/sources") && init?.method === "POST") {
      return Response.json({
        resource: "notebooks/nb-1",
        title: "My Project",
        sources: [
          {
            resource: "notebooks/nb-1/sources/src-1",
            file_name: "notes.md",
            mime_type: "text/markdown",
          },
        ],
        source_count: 1,
      });
    }
    if (url.includes("/sources/") && init?.method === "DELETE") {
      return Response.json({ removed: "notebooks/nb-1/sources/src-1" });
    }
    return Response.json({
      resource: "notebooks/nb-1",
      title: "My Project",
      sources: [],
      source_count: 0,
    });
  });

  assert.equal((await client.create("My Project")).resource, "notebooks/nb-1");
  assert.equal((await client.get("nb-1")).source_count, 0);
  assert.deepEqual(await client.listChats("nb-1"), [{ cid: "c_1", title: "First chat" }]);
  assert.equal((await client.addFileSource("nb-1", "/tmp/notes.md")).source_count, 1);
  assert.equal((await client.addUrlSource("nb-1", "https://example.com")).source_count, 1);
  assert.deepEqual(await client.removeSource("notebooks/nb-1/sources/src-1"), {
    removed: "notebooks/nb-1/sources/src-1",
  });

  assert.deepEqual(requests, [
    {
      url: "http://gemini.internal:8080/v1/notebooks",
      method: "POST",
      body: '{"title":"My Project"}',
    },
    { url: "http://gemini.internal:8080/v1/notebooks/nb-1", method: "GET", body: undefined },
    {
      url: "http://gemini.internal:8080/v1/notebooks/nb-1/chats",
      method: "GET",
      body: undefined,
    },
    {
      url: "http://gemini.internal:8080/v1/notebooks/nb-1/sources",
      method: "POST",
      body: '{"path":"/tmp/notes.md"}',
    },
    {
      url: "http://gemini.internal:8080/v1/notebooks/nb-1/sources",
      method: "POST",
      body: '{"url":"https://example.com"}',
    },
    {
      url: "http://gemini.internal:8080/v1/notebooks/nb-1/sources/src-1",
      method: "DELETE",
      body: undefined,
    },
  ]);
});

test("rejects invalid source resource names before hitting the server", async () => {
  const client = new GeminiWebNotebookClient("http://gemini.internal:8080", async () => {
    throw new Error("fetch should not be called");
  });

  await assert.rejects(client.removeSource("src-1"), /invalid source resource name/);
});

test("surfaces server errors without exposing an unbounded response", async () => {
  const client = new GeminiWebNotebookClient(
    "http://gemini.internal:8080",
    async () => new Response("x".repeat(1000), { status: 500 }),
  );

  await assert.rejects(client.get("nb-1"), (error: unknown) => {
    assert.ok(error instanceof Error);
    assert.match(error.message, /HTTP 500/);
    assert.ok(error.message.length < 700);
    return true;
  });
});
