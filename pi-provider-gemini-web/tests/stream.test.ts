import assert from "node:assert/strict";
import test from "node:test";

import {
  type Api,
  type AssistantMessage,
  type AssistantMessageEventStream,
  type Model,
  type SimpleStreamOptions,
  createAssistantMessageEventStream,
} from "@earendil-works/pi-ai";
import { buildGeminiWebPayloadHook, createGeminiWebStream } from "../src/stream.ts";

function message(model: Model<Api>, stopReason: "stop" | "error"): AssistantMessage {
  return {
    role: "assistant",
    content: [],
    api: model.api,
    provider: model.provider,
    model: model.id,
    usage: {
      input: 0,
      output: 0,
      cacheRead: 0,
      cacheWrite: 0,
      totalTokens: 0,
      cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0 },
    },
    stopReason,
    timestamp: Date.now(),
  };
}

function terminalStream(model: Model<Api>, succeeds: boolean): AssistantMessageEventStream {
  const stream = createAssistantMessageEventStream();
  queueMicrotask(() => {
    const output = message(model, succeeds ? "stop" : "error");
    if (succeeds) {
      stream.push({ type: "done", reason: "stop", message: output });
    } else {
      output.errorMessage = "failed";
      stream.push({ type: "error", reason: "error", error: output });
    }
    stream.end();
  });
  return stream;
}

async function consume(stream: AssistantMessageEventStream): Promise<void> {
  for await (const _event of stream) {
    // Consume all events so terminal state updates run.
  }
}

const model = {
  id: "gemini-3.5-flash",
  provider: "gemini-web",
  api: "gemini-web-chat-completions",
  baseUrl: "http://placeholder.invalid/v1",
} as Model<Api>;

const context = {
  messages: [
    { role: "assistant", provider: "gemini-web", responseId: "chatcmpl-c_42" },
    { role: "user", content: "next", timestamp: 1 },
  ],
};

test("payload hook preserves chat continuity after an outer payload replacement", async () => {
  const seen: unknown[] = [];
  const hook = buildGeminiWebPayloadHook(context, false, async (payload) => {
    seen.push(structuredClone(payload));
    return { model: "gemini-3.5-flash", temperature: 0 };
  });

  const result = await hook({ model: "gemini-3.5-flash" }, model);
  assert.deepEqual(seen, [{ model: "gemini-3.5-flash", chat_id: "c_42" }]);
  assert.deepEqual(result, {
    model: "gemini-3.5-flash",
    chat_id: "c_42",
    temperature: 0,
  });
});

test("keeps a fork fresh after failure and clears it only after success", async () => {
  const state = { baseUrl: "http://gemini.internal:8080", freshSessionIds: new Set(["fork-1"]) };
  const apiKeys: Array<string | undefined> = [];
  const failedProvider = (
    providerModel: Model<Api>,
    _context: unknown,
    options?: SimpleStreamOptions,
  ) => {
    apiKeys.push(options?.apiKey);
    return terminalStream(providerModel, false);
  };

  await consume(
    createGeminiWebStream(state, failedProvider)(model, context as never, {
      sessionId: "fork-1",
      apiKey: "must-not-leak",
    }),
  );
  assert.equal(state.freshSessionIds.has("fork-1"), true);
  assert.deepEqual(apiKeys, ["gemini-web-internal"]);

  const successfulProvider = (providerModel: Model<Api>) => terminalStream(providerModel, true);
  await consume(
    createGeminiWebStream(state, successfulProvider)(model, context as never, {
      sessionId: "fork-1",
    }),
  );
  assert.equal(state.freshSessionIds.has("fork-1"), false);
});
