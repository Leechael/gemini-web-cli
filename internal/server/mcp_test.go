package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

func mustTestClient(t *testing.T) *client.Client {
	t.Helper()
	c, err := client.New(client.Config{Secure1PSID: "dummy", Secure1PSIDTS: "dummy"})
	if err != nil {
		t.Fatalf("creating test client: %v", err)
	}
	return c
}

func TestMCPRouteRegisteredWithoutAuth(t *testing.T) {
	s := &Server{mux: http.NewServeMux(), apiKey: "secret"}
	s.registerRoutes()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			Capabilities    struct {
				Tools     json.RawMessage `json:"tools"`
				Resources json.RawMessage `json:"resources"`
				Prompts   json.RawMessage `json:"prompts"`
			} `json:"capabilities"`
			ServerInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	if resp.Result.ProtocolVersion != "2025-06-18" {
		t.Fatalf("protocolVersion = %q, want 2025-06-18", resp.Result.ProtocolVersion)
	}
	if len(resp.Result.Capabilities.Tools) == 0 {
		t.Fatal("initialize result missing tools capability; clients expect it advertised")
	}
	if len(resp.Result.Capabilities.Resources) == 0 {
		t.Fatal("initialize result missing resources capability; clients probing resources/list get -32601 without it")
	}
	if len(resp.Result.Capabilities.Prompts) == 0 {
		t.Fatal("initialize result missing prompts capability; clients probing prompts/list get -32601 without it")
	}
	if resp.Result.ServerInfo.Name != "gemini-web-cli" {
		t.Fatalf("serverInfo.name = %q, want gemini-web-cli", resp.Result.ServerInfo.Name)
	}
	if resp.Result.ServerInfo.Version != "dev" {
		t.Fatalf("serverInfo.version = %q, want dev", resp.Result.ServerInfo.Version)
	}
}

func TestMCPResourcesAndPromptsListEmpty(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.registerRoutes()

	for _, method := range []string{"resources/list", "prompts/list"} {
		body := `{"jsonrpc":"2.0","id":99,"method":"` + method + `","params":{}}`
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2025-06-18")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d; body = %s", method, w.Code, http.StatusOK, w.Body.String())
		}
		var resp struct {
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s: unmarshal response: %v; body=%s", method, err, w.Body.String())
		}
		if resp.Error != nil {
			t.Fatalf("%s: got JSON-RPC error code=%d msg=%q; clients expect an empty result, not -32601", method, resp.Error.Code, resp.Error.Message)
		}
	}
}

func TestMCPLoggingSummarizeArgs(t *testing.T) {
	long := strings.Repeat("x", maxLoggedArgLen+50)
	got := summarizeMCPArgs(map[string]any{"prompt": long, "n": 3})
	if !strings.Contains(got, "\"prompt\":\""+strings.Repeat("x", maxLoggedArgLen)+"...\"") {
		t.Fatalf("long arg not truncated to %d chars + ...: %s", maxLoggedArgLen, got)
	}
	if strings.Contains(got, strings.Repeat("x", maxLoggedArgLen+1)) {
		t.Fatalf("arg exceeds cap of %d chars: %s", maxLoggedArgLen, got)
	}
	if !strings.Contains(got, `"n":3`) {
		t.Fatalf("short arg missing from summary: %s", got)
	}

	if got := summarizeMCPArgs(nil); got != "{}" {
		t.Fatalf("nil args = %q, want {}", got)
	}
}

func TestMCPLoggingStatusWriter(t *testing.T) {
	s := &statusWriter{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	s.WriteHeader(http.StatusTeapot)
	if s.status != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", s.status, http.StatusTeapot)
	}
	// second WriteHeader must not overwrite
	s.WriteHeader(http.StatusBadRequest)
	if s.status != http.StatusTeapot {
		t.Fatalf("status overwritten to %d", s.status)
	}
}

func TestMCPLoggingMethodFromRequest(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	if got := mcpMethodFromRequest(req); got != "tools/call" {
		t.Fatalf("method = %q, want tools/call", got)
	}
	// body must still be readable for downstream handler
	rest, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("re-read body: %v", err)
	}
	if !strings.Contains(string(rest), "tools/call") {
		t.Fatalf("body not re-readable after peek: %s", string(rest))
	}

	req2 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("not json"))
	if got := mcpMethodFromRequest(req2); got != "?" {
		t.Fatalf("method = %q, want ?", got)
	}
}

func TestMCPToolsList(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.registerRoutes()

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}

	want := map[string]bool{
		"gemini_ask":             false,
		"gemini_list_models":     false,
		"gemini_research_create": false,
		"gemini_research_list":   false,
		"gemini_research_reply":  false,
		"gemini_research_status": false,
		"gemini_research_result": false,
	}
	for _, tool := range resp.Result.Tools {
		if _, ok := want[tool.Name]; !ok {
			t.Fatalf("unexpected tool %q", tool.Name)
		}
		want[tool.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("missing tool %q", name)
		}
	}
}

func TestResolveMCPModel(t *testing.T) {
	c := mustTestClient(t)

	s := &Server{client: c, mcpDefaultModel: "gemini-3.5-flash"}
	if m := s.resolveMCPModel("unspecified"); m == nil || m.Name != "unspecified" {
		t.Fatalf("override should take precedence, got %v", m)
	}
	if m := s.resolveMCPModel(""); m == nil || m.Name != "gemini-3.5-flash" {
		t.Fatalf("default model not applied, got %v", m)
	}

	s2 := &Server{client: c}
	if m := s2.resolveMCPModel(""); m == nil || m.Name != "unspecified" {
		t.Fatalf("fallback to unspecified failed, got %v", m)
	}
}

func TestMCPResearchCreateRequiresPrompt(t *testing.T) {
	s := &Server{}
	result, err := s.handleMCPResearchCreate(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected tool-level error result for missing prompt")
	}
}

func TestMCPListModelsEmptyWhenNotFetched(t *testing.T) {
	s := &Server{client: mustTestClient(t)}
	result, err := s.handleMCPListModels(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected error result: %v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}
	text := result.Content[0].(mcp.TextContent).Text
	var parsed struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal list_models result: %v", err)
	}
	if len(parsed.Models) != 0 {
		t.Fatalf("models = %d, want 0 when not fetched", len(parsed.Models))
	}
}

func TestMCPResearchStatusRequiresID(t *testing.T) {
	s := &Server{}
	result, err := s.handleMCPResearchStatus(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected tool-level error result for missing id")
	}
}

func TestMCPResearchResultRequiresID(t *testing.T) {
	s := &Server{}
	result, err := s.handleMCPResearchResult(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected tool-level error result for missing id")
	}
}

func TestMCPResearchReplyRequiresID(t *testing.T) {
	s := &Server{}
	result, err := s.handleMCPResearchReply(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"prompt": "refine"}},
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected tool-level error result for missing id")
	}
}

func TestMCPResearchReplyRequiresPrompt(t *testing.T) {
	s := &Server{}
	result, err := s.handleMCPResearchReply(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"id": "c_abc123"}},
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected tool-level error result for missing prompt")
	}
}
