import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { truncateHead, type ToolDefinition } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

import { GeminiWebResearchClient } from "../research.ts";

interface ResearchToolConfig {
  getBaseUrl(): string | undefined;
}

function clientFor(config: ResearchToolConfig, signal?: AbortSignal): GeminiWebResearchClient {
  const baseUrl = config.getBaseUrl();
  if (!baseUrl) {
    throw new Error("Gemini Web is not configured");
  }
  return new GeminiWebResearchClient(baseUrl, fetch, signal);
}

function resultContent(value: unknown) {
  const text = JSON.stringify(value, null, 2);
  const truncated = truncateHead(text);
  if (!truncated.truncated) {
    return { content: [{ type: "text" as const, text }], details: value };
  }

  const dir = mkdtempSync(join(tmpdir(), "pi-gemini-web-"));
  const path = join(dir, "result.json");
  writeFileSync(path, `${text}\n`, "utf8");
  return {
    content: [
      {
        type: "text" as const,
        text: `${truncated.content}\n\n[Output truncated. Full output saved to: ${path}]`,
      },
    ],
    details: { value, fullOutputPath: path },
  };
}

export function buildResearchTools(config: ResearchToolConfig): ToolDefinition[] {
  return [
    {
      name: "gemini_research_create",
      label: "Gemini Research Create",
      description: "Submit a Gemini Deep Research task and return its research id.",
      promptSnippet: "Submit a Gemini Deep Research task",
      promptGuidelines: [
        "Use gemini_research_create when the user explicitly asks for deep research, then poll gemini_research_status and fetch gemini_research_result when done.",
      ],
      parameters: Type.Object({
        prompt: Type.String({ description: "Research topic or prompt" }),
        model: Type.Optional(Type.String({ description: "Gemini model name override" })),
      }),
      async execute(
        _toolCallId: string,
        params: { prompt: string; model?: string },
        signal?: AbortSignal,
      ) {
        return resultContent(await clientFor(config, signal).create(params.prompt, params.model));
      },
    },
    {
      name: "gemini_research_status",
      label: "Gemini Research Status",
      description: "Check the state of a Gemini Deep Research task.",
      promptSnippet: "Check a Gemini Deep Research task",
      parameters: Type.Object({
        id: Type.String({ description: "Research id returned by gemini_research_create" }),
      }),
      async execute(_toolCallId: string, params: { id: string }, signal?: AbortSignal) {
        return resultContent(await clientFor(config, signal).status(params.id));
      },
    },
    {
      name: "gemini_research_result",
      label: "Gemini Research Result",
      description: "Fetch a completed Gemini Deep Research report and its sources.",
      promptSnippet: "Fetch a completed Gemini Deep Research report",
      parameters: Type.Object({
        id: Type.String({ description: "Completed research id" }),
      }),
      async execute(_toolCallId: string, params: { id: string }, signal?: AbortSignal) {
        return resultContent(await clientFor(config, signal).result(params.id));
      },
    },
  ];
}
