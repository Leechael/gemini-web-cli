import assert from "node:assert/strict";
import test from "node:test";

import { discoverGeminiWebModels } from "../src/models.ts";

test("discovers Pi model definitions from the configured server", async () => {
  const requests: string[] = [];
  const models = await discoverGeminiWebModels("http://gemini.internal:8080", async (input) => {
    requests.push(String(input));
    return new Response(
      JSON.stringify({
        object: "list",
        data: [
          { id: "gemini-3.5-flash", object: "model", owned_by: "google" },
          { id: "gemini-3.1-pro", object: "model", owned_by: "google" },
        ],
      }),
      { status: 200, headers: { "content-type": "application/json" } },
    );
  });

  assert.deepEqual(requests, ["http://gemini.internal:8080/v1/models"]);
  assert.deepEqual(
    models.map(({ id, name, reasoning, input }) => ({ id, name, reasoning, input })),
    [
      {
        id: "gemini-3.5-flash",
        name: "gemini-3.5-flash",
        reasoning: true,
        input: ["text"],
      },
      {
        id: "gemini-3.1-pro",
        name: "gemini-3.1-pro",
        reasoning: true,
        input: ["text"],
      },
    ],
  );
});

test("rejects failed and empty model discovery responses", async () => {
  await assert.rejects(
    discoverGeminiWebModels(
      "http://gemini.internal:8080",
      async () => new Response("unavailable", { status: 503 }),
    ),
    /503.*unavailable/,
  );
  await assert.rejects(
    discoverGeminiWebModels(
      "http://gemini.internal:8080",
      async () => new Response(JSON.stringify({ object: "list", data: [] })),
    ),
    /returned no models/,
  );
});
