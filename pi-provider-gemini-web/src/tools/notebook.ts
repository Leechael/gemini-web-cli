import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { truncateHead, type ToolDefinition } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

import { GeminiWebNotebookClient } from "../notebook.ts";

interface NotebookToolConfig {
  getBaseUrl(): string | undefined;
}

function clientFor(config: NotebookToolConfig, signal?: AbortSignal): GeminiWebNotebookClient {
  const baseUrl = config.getBaseUrl();
  if (!baseUrl) {
    throw new Error("Gemini Web is not configured");
  }
  return new GeminiWebNotebookClient(baseUrl, fetch, signal);
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

export function buildNotebookTools(config: NotebookToolConfig): ToolDefinition[] {
  return [
    {
      name: "gemini_notebook_create",
      label: "Gemini Notebook Create",
      description: "Create a Gemini notebook and return its notebooks/<uuid> resource name.",
      promptSnippet: "Create a Gemini notebook",
      parameters: Type.Object({
        title: Type.String({ description: "Notebook title" }),
      }),
      async execute(_toolCallId: string, params: { title: string }, signal?: AbortSignal) {
        return resultContent(await clientFor(config, signal).create(params.title));
      },
    },
    {
      name: "gemini_notebook_get",
      label: "Gemini Notebook Get",
      description: "Get a notebook's title, emoji, and source list.",
      promptSnippet: "Inspect a Gemini notebook",
      parameters: Type.Object({
        id: Type.String({ description: "Notebook id, with or without the notebooks/ prefix" }),
      }),
      async execute(_toolCallId: string, params: { id: string }, signal?: AbortSignal) {
        return resultContent(await clientFor(config, signal).get(params.id));
      },
    },
    {
      name: "gemini_notebook_list_chats",
      label: "Gemini Notebook List Chats",
      description: "List the chats belonging to a notebook, newest first.",
      promptSnippet: "List chats inside a Gemini notebook",
      parameters: Type.Object({
        id: Type.String({ description: "Notebook id, with or without the notebooks/ prefix" }),
      }),
      async execute(_toolCallId: string, params: { id: string }, signal?: AbortSignal) {
        return resultContent(await clientFor(config, signal).listChats(params.id));
      },
    },
    {
      name: "gemini_notebook_add_file_source",
      label: "Gemini Notebook Add File Source",
      description:
        "Upload a local file (on the machine running the gemini-web-cli server) and attach it to a notebook as a source.",
      promptSnippet: "Attach a local file to a Gemini notebook",
      parameters: Type.Object({
        id: Type.String({ description: "Notebook id, with or without the notebooks/ prefix" }),
        path: Type.String({ description: "Local file path to upload" }),
      }),
      async execute(
        _toolCallId: string,
        params: { id: string; path: string },
        signal?: AbortSignal,
      ) {
        return resultContent(await clientFor(config, signal).addFileSource(params.id, params.path));
      },
    },
    {
      name: "gemini_notebook_add_url_source",
      label: "Gemini Notebook Add URL Source",
      description: "Attach a web URL to a notebook as a source.",
      promptSnippet: "Attach a web URL to a Gemini notebook",
      parameters: Type.Object({
        id: Type.String({ description: "Notebook id, with or without the notebooks/ prefix" }),
        url: Type.String({ description: "http(s) URL to attach" }),
      }),
      async execute(
        _toolCallId: string,
        params: { id: string; url: string },
        signal?: AbortSignal,
      ) {
        return resultContent(await clientFor(config, signal).addUrlSource(params.id, params.url));
      },
    },
    {
      name: "gemini_notebook_remove_source",
      label: "Gemini Notebook Remove Source",
      description: "Remove a source from a notebook.",
      promptSnippet: "Remove a source from a Gemini notebook",
      parameters: Type.Object({
        source: Type.String({
          description: "Full source resource name: notebooks/<uuid>/sources/<sid>",
        }),
      }),
      async execute(_toolCallId: string, params: { source: string }, signal?: AbortSignal) {
        return resultContent(await clientFor(config, signal).removeSource(params.source));
      },
    },
  ];
}
