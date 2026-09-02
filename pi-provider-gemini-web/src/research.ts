export interface ResearchCreateResult {
  id: string;
  chat_id: string;
  title?: string;
  eta_text?: string;
  steps?: string[];
}

export interface ResearchStatusResult {
  id: string;
  chat_id: string;
  state: "done" | "running" | "pending_confirm" | "not_research" | "empty";
}

export interface ResearchResult {
  id: string;
  chat_id: string;
  text: string;
  sources?: Array<{ url: string; title: string }>;
}

export class GeminiWebResearchClient {
  private readonly baseUrl: string;
  private readonly fetchFn: typeof fetch;
  private readonly signal?: AbortSignal;

  constructor(baseUrl: string, fetchFn: typeof fetch = fetch, signal?: AbortSignal) {
    this.baseUrl = baseUrl;
    this.fetchFn = fetchFn;
    this.signal = signal;
  }

  create(prompt: string, model?: string): Promise<ResearchCreateResult> {
    return this.request("/v1/research", {
      method: "POST",
      headers: { "content-type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ prompt, ...(model ? { model } : {}) }),
    });
  }

  status(id: string): Promise<ResearchStatusResult> {
    return this.request(`/v1/research/${encodeURIComponent(id)}/status`);
  }

  result(id: string): Promise<ResearchResult> {
    return this.request(`/v1/research/${encodeURIComponent(id)}/result`);
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await this.fetchFn(`${this.baseUrl}${path}`, {
      ...init,
      signal: this.signal,
    });
    if (!response.ok) {
      const body = (await response.text()).slice(0, 500).trim();
      throw new Error(
        `Gemini Web request failed with HTTP ${response.status}${body ? `: ${body}` : ""}`,
      );
    }
    return (await response.json()) as T;
  }
}
