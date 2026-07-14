import { PROVIDER_ID } from "./models.ts";

interface ContextMessage {
  role?: unknown;
  provider?: unknown;
  responseId?: unknown;
  content?: unknown;
}

export interface ChatContext {
  messages: ContextMessage[];
}

export function extractChatId(context: ChatContext): string | undefined {
  for (let index = context.messages.length - 1; index >= 0; index--) {
    const message = context.messages[index];
    if (
      message.role !== "assistant" ||
      message.provider !== PROVIDER_ID ||
      typeof message.responseId !== "string" ||
      message.responseId.length === 0
    ) {
      continue;
    }
    return message.responseId.startsWith("chatcmpl-")
      ? message.responseId.slice("chatcmpl-".length)
      : message.responseId;
  }
  return undefined;
}

export function applyChatContinuity(
  payload: Record<string, unknown>,
  context: ChatContext,
  startFresh: boolean,
): void {
  if (startFresh) {
    delete payload.chat_id;
    return;
  }
  const chatId = extractChatId(context);
  if (chatId) payload.chat_id = chatId;
}
