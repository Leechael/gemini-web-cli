import assert from "node:assert/strict";
import test from "node:test";

import { buildGeminiWebPayloadHook } from "../src/stream.ts";

test("payload hook injects chat continuity before calling the outer hook", async () => {
  const seen: unknown[] = [];
  const hook = buildGeminiWebPayloadHook(
    {
      messages: [
        { role: "assistant", provider: "gemini-web", responseId: "chatcmpl-c_42" },
        { role: "user", content: "next" },
      ],
    },
    false,
    async (payload) => {
      seen.push(structuredClone(payload));
      return { ...(payload as object), temperature: 0 };
    },
  );

  const result = await hook({ model: "gemini-3.5-flash" }, {} as never);
  assert.deepEqual(seen, [{ model: "gemini-3.5-flash", chat_id: "c_42" }]);
  assert.deepEqual(result, {
    model: "gemini-3.5-flash",
    chat_id: "c_42",
    temperature: 0,
  });
});
