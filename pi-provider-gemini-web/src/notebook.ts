export interface NotebookSource {
  resource: string;
  file_name: string;
  mime_type: string;
  uploaded_at_unix?: number;
}

export interface Notebook {
  resource: string;
  title: string;
  emoji?: string;
  sources: NotebookSource[];
  source_count: number;
  created_unix?: number;
  updated_unix?: number;
}

export interface NotebookChat {
  cid: string;
  title: string;
  updated?: string;
}

export class GeminiWebNotebookClient {
  private readonly baseUrl: string;
  private readonly fetchFn: typeof fetch;
  private readonly signal?: AbortSignal;

  constructor(baseUrl: string, fetchFn: typeof fetch = fetch, signal?: AbortSignal) {
    this.baseUrl = baseUrl;
    this.fetchFn = fetchFn;
    this.signal = signal;
  }

  create(title: string): Promise<{ resource: string; title: string }> {
    return this.request("/v1/notebooks", {
      method: "POST",
      headers: { "content-type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ title }),
    });
  }

  get(id: string): Promise<Notebook> {
    return this.request(`/v1/notebooks/${encodeURIComponent(id)}`);
  }

  async listChats(id: string): Promise<NotebookChat[]> {
    const result = await this.request<{ chats: NotebookChat[] }>(
      `/v1/notebooks/${encodeURIComponent(id)}/chats`,
    );
    return result.chats;
  }

  addFileSource(id: string, path: string): Promise<Notebook> {
    return this.request(`/v1/notebooks/${encodeURIComponent(id)}/sources`, {
      method: "POST",
      headers: { "content-type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ path }),
    });
  }

  addUrlSource(id: string, url: string): Promise<Notebook> {
    return this.request(`/v1/notebooks/${encodeURIComponent(id)}/sources`, {
      method: "POST",
      headers: { "content-type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ url }),
    });
  }

  async removeSource(source: string): Promise<{ removed: string }> {
    const match = /^notebooks\/([^/]+)\/sources\/([^/]+)$/.exec(source);
    if (!match) {
      throw new Error(`invalid source resource name: ${source}`);
    }
    return this.request(
      `/v1/notebooks/${encodeURIComponent(match[1])}/sources/${encodeURIComponent(match[2])}`,
      { method: "DELETE" },
    );
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
