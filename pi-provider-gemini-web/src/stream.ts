import {
  type Api,
  type AssistantMessage,
  type AssistantMessageEventStream,
  type Context,
  type Model,
  type SimpleStreamOptions,
  createAssistantMessageEventStream,
  streamSimpleOpenAICompletions,
} from "@earendil-works/pi-ai";

import { applyChatContinuity, type ChatContext } from "./session.ts";

export interface GeminiWebStreamState {
  baseUrl?: string;
  freshSessionIds: Set<string>;
}

export function buildGeminiWebPayloadHook(
  context: ChatContext,
  startFresh: boolean,
  outerHook?: SimpleStreamOptions["onPayload"],
): NonNullable<SimpleStreamOptions["onPayload"]> {
  return async (payload, model) => {
    let nextPayload: unknown = payload;
    if (typeof nextPayload === "object" && nextPayload !== null && !Array.isArray(nextPayload)) {
      applyChatContinuity(nextPayload as Record<string, unknown>, context, startFresh);
    }
    if (outerHook) {
      const result = await outerHook(nextPayload, model);
      if (result !== undefined) nextPayload = result;
    }
    return nextPayload;
  };
}

function unconfiguredStream(model: Model<Api>): AssistantMessageEventStream {
  const stream = createAssistantMessageEventStream();
  const error: AssistantMessage = {
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
    stopReason: "error",
    errorMessage: "Gemini Web is not configured for this project",
    timestamp: Date.now(),
  };
  queueMicrotask(() => {
    stream.push({ type: "error", reason: "error", error });
    stream.end();
  });
  return stream;
}

export function createGeminiWebStream(state: GeminiWebStreamState) {
  return (
    model: Model<Api>,
    context: Context,
    options?: SimpleStreamOptions,
  ): AssistantMessageEventStream => {
    if (!state.baseUrl) return unconfiguredStream(model);

    const sessionId = options?.sessionId;
    const startFresh = sessionId ? state.freshSessionIds.delete(sessionId) : false;
    const runtimeModel = {
      ...model,
      api: "openai-completions",
      baseUrl: `${state.baseUrl}/v1`,
    } as Model<"openai-completions">;

    return streamSimpleOpenAICompletions(runtimeModel, context, {
      ...options,
      apiKey: options?.apiKey || "gemini-web-internal",
      onPayload: buildGeminiWebPayloadHook(context as ChatContext, startFresh, options?.onPayload),
    });
  };
}
