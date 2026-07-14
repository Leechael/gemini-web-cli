export const PROVIDER_ID = "gemini-web";
export const API_ID = "gemini-web-chat-completions";

export interface GeminiWebModelDefinition {
  id: string;
  name: string;
  reasoning: boolean;
  input: ["text"];
  cost: { input: number; output: number; cacheRead: number; cacheWrite: number };
  contextWindow: number;
  maxTokens: number;
  compat: { supportsDeveloperRole: false };
}

interface ModelsResponse {
  data?: Array<{ id?: unknown }>;
}

function buildModel(id: string): GeminiWebModelDefinition {
  return {
    id,
    name: id,
    reasoning: true,
    input: ["text"],
    cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
    contextWindow: 128000,
    maxTokens: 16384,
    compat: { supportsDeveloperRole: false },
  };
}

export const FALLBACK_MODELS: GeminiWebModelDefinition[] = [
  "gemini-3.5-flash",
  "gemini-3.1-pro",
  "gemini-3.1-flash-lite",
].map(buildModel);

export async function discoverGeminiWebModels(
  baseUrl: string,
  fetchFn: typeof fetch = fetch,
): Promise<GeminiWebModelDefinition[]> {
  const response = await fetchFn(`${baseUrl}/v1/models`, {
    headers: { Accept: "application/json" },
  });
  if (!response.ok) {
    const body = (await response.text()).slice(0, 500).trim();
    throw new Error(
      `Gemini Web model discovery failed with HTTP ${response.status}${body ? `: ${body}` : ""}`,
    );
  }

  const payload = (await response.json()) as ModelsResponse;
  const ids = (payload.data ?? [])
    .map((model) => model.id)
    .filter((id): id is string => typeof id === "string" && id.length > 0);
  if (ids.length === 0) {
    throw new Error("Gemini Web model discovery returned no models");
  }

  return ids.map(buildModel);
}
