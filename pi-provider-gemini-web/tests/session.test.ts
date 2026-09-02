import assert from "node:assert/strict";
import test from "node:test";

import { applyChatContinuity, extractChatId } from "../src/session.ts";

test("injects the latest Gemini Web chat id into the next request", () => {
  const payload: Record<string, unknown> = { model: "gemini-3.5-flash", stream: true };
  const context = {
    messages: [
      {
        role: "assistant",
        provider: "gemini-web",
        responseId: "chatcmpl-c_123",
      },
      { role: "user", content: "continue" },
    ],
  };

  applyChatContinuity(payload, context, false);
  assert.equal(payload.chat_id, "c_123");
});

test("uses only assistant response ids produced by this provider", () => {
  assert.equal(
    extractChatId({
      messages: [
        { role: "assistant", provider: "gemini-web", responseId: "chatcmpl-c_old" },
        { role: "assistant", provider: "openai", responseId: "chatcmpl-wrong" },
        { role: "user", content: "continue" },
      ],
    }),
    "c_old",
  );
});

test("starts a new remote chat for the first request after a fork", () => {
  const payload: Record<string, unknown> = { model: "gemini-3.5-flash" };
  applyChatContinuity(
    payload,
    {
      messages: [
        { role: "assistant", provider: "gemini-web", responseId: "chatcmpl-c_parent" },
        { role: "user", content: "branch" },
      ],
    },
    true,
  );

  assert.equal(Object.hasOwn(payload, "chat_id"), false);
});
