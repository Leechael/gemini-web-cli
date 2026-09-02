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

type ProviderStream = (
  model: Model<"openai-completions">,
  context: Context,
  options?: SimpleStreamOptions,
) => AssistantMessageEventStream;

function applyContinuityToPayload(
  payload: unknown,
  context: ChatContext,
  startFresh: boolean,
): void {
  if (typeof payload === "object" && payload !== null && !Array.isArray(payload)) {
    applyChatContinuity(payload as Record<string, unknown>, context, startFresh);
  }
}

export function buildGeminiWebPayloadHook(
  context: ChatContext,
  startFresh: boolean,
  outerHook?: SimpleStreamOptions["onPayload"],
): NonNullable<SimpleStreamOptions["onPayload"]> {
  return async (payload, model) => {
    let nextPayload: unknown = payload;
    applyContinuityToPayload(nextPayload, context, startFresh);
    if (outerHook) {
      const result = await outerHook(nextPayload, model);
      if (result !== undefined) nextPayload = result;
    }
    applyContinuityToPayload(nextPayload, context, startFresh);
    return nextPayload;
  };
}

function errorMessage(model: Model<Api>, message: string): AssistantMessage {
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
    stopReason: "error",
    errorMessage: message,
    timestamp: Date.now(),
  };
}

function unconfiguredStream(model: Model<Api>): AssistantMessageEventStream {
  const stream = createAssistantMessageEventStream();
  queueMicrotask(() => {
    stream.push({
      type: "error",
      reason: "error",
      error: errorMessage(model, "Gemini Web is not configured for this project"),
    });
  });
  return stream;
}

function trackSuccessfulFork(
  upstream: AssistantMessageEventStream,
  model: Model<Api>,
  onDone: () => void,
): AssistantMessageEventStream {
  const stream = createAssistantMessageEventStream();
  void (async () => {
    try {
      for await (const event of upstream) {
        if (event.type === "done") onDone();
        stream.push(event);
      }
    } catch (error) {
      stream.push({
        type: "error",
        reason: "error",
        error: errorMessage(model, error instanceof Error ? error.message : String(error)),
      });
    }
  })();
  return stream;
}

export function createGeminiWebStream(
  state: GeminiWebStreamState,
  providerStream: ProviderStream = streamSimpleOpenAICompletions,
) {
  return (
    model: Model<Api>,
    context: Context,
    options?: SimpleStreamOptions,
  ): AssistantMessageEventStream => {
    if (!state.baseUrl) return unconfiguredStream(model);

    const sessionId = options?.sessionId;
    const startFresh = sessionId ? state.freshSessionIds.has(sessionId) : false;
    const runtimeModel = {
      ...model,
      api: "openai-completions",
      baseUrl: `${state.baseUrl}/v1`,
    } as Model<"openai-completions">;

    const upstream = providerStream(runtimeModel, context, {
      ...options,
      apiKey: "gemini-web-internal",
      onPayload: buildGeminiWebPayloadHook(context as ChatContext, startFresh, options?.onPayload),
    });
    if (!startFresh || !sessionId) return upstream;

    return trackSuccessfulFork(upstream, model, () => {
      state.freshSessionIds.delete(sessionId);
    });
  };
}
