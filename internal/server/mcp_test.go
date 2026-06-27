package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
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
	secretPrompt := "secret token: sk-test-value"
	got := summarizeMCPArgs(map[string]any{
		"prompt": secretPrompt,
		"n":      3,
		"items":  []any{"a", "b"},
		"meta":   map[string]any{"x": 1},
	})
	if strings.Contains(got, secretPrompt) || strings.Contains(got, "sk-test-value") {
		t.Fatalf("string arg value leaked into log summary: %s", got)
	}
	if !strings.Contains(got, fmt.Sprintf(`"prompt":"string(len=%d)"`, len(secretPrompt))) {
		t.Fatalf("string arg length missing from summary: %s", got)
	}
	if !strings.Contains(got, `"n":3`) {
		t.Fatalf("numeric arg missing from summary: %s", got)
	}
	if !strings.Contains(got, `"items":"array(len=2)"`) {
		t.Fatalf("array summary missing: %s", got)
	}
	if !strings.Contains(got, `"meta":"object(keys=1)"`) {
		t.Fatalf("object summary missing: %s", got)
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

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (r *flushRecorder) Flush() {
	r.flushed = true
}

func TestMCPLoggingStatusWriterPreservesFlush(t *testing.T) {
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	s := &statusWriter{ResponseWriter: recorder, status: http.StatusOK}
	s.Flush()
	if !recorder.flushed {
		t.Fatal("Flush was not forwarded to underlying ResponseWriter")
	}
}

func TestMCPLoggingMethodFromRequest(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	got, tooLarge := mcpMethodFromRequest(req)
	if tooLarge {
		t.Fatal("method extraction unexpectedly marked request too large")
	}
	if got != "tools/call" {
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

	// POST with unparseable body
	req2 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("not json"))
	got, tooLarge = mcpMethodFromRequest(req2)
	if tooLarge {
		t.Fatal("invalid JSON request unexpectedly marked request too large")
	}
	if got != "POST?" {
		t.Fatalf("method = %q, want POST?", got)
	}

	// JSON-RPC batch: extract first method
	batch := `[{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}},{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{}}]`
	req3 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(batch))
	got, tooLarge = mcpMethodFromRequest(req3)
	if tooLarge {
		t.Fatal("batch request unexpectedly marked request too large")
	}
	if got != "tools/list" {
		t.Fatalf("batch method = %q, want tools/list", got)
	}
	rest3, _ := io.ReadAll(req3.Body)
	if !strings.Contains(string(rest3), "tools/call") {
		t.Fatalf("batch body not re-readable: %s", string(rest3))
	}

	// Non-POST (GET SSE probe, DELETE session terminate) has empty body: log
	// the HTTP method instead of a meaningless "?".
	for _, m := range []string{http.MethodGet, http.MethodDelete, http.MethodOptions} {
		r := httptest.NewRequest(m, "/mcp", nil)
		got, tooLarge = mcpMethodFromRequest(r)
		if tooLarge {
			t.Fatalf("%s unexpectedly marked request too large", m)
		}
		if got != m {
			t.Fatalf("%s method = %q, want %q", m, got, m)
		}
	}
}

func TestMCPLoggingMiddlewareRejectsLargeBody(t *testing.T) {
	largeBody := strings.Repeat("x", maxRequestBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(largeBody))
	req.ContentLength = int64(len(largeBody))
	w := httptest.NewRecorder()

	mcpLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("downstream handler should not run for oversized body")
	})).ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestMCPClientIP(t *testing.T) {
	cases := []struct {
		name   string
		remote string
		xff    string
		want   string
	}{
		{name: "remote with port", remote: "192.168.1.5:54321", want: "192.168.1.5"},
		{name: "remote no port", remote: "10.0.0.1", want: "10.0.0.1"},
		{name: "ipv6 remote", remote: "[::1]:1234", want: "::1"},
		{name: "xff ignored", remote: "127.0.0.1:1", xff: "203.0.113.7", want: "127.0.0.1"},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := clientIP(r); got != c.want {
			t.Fatalf("%s: clientIP = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestBannerHostSpecificBind(t *testing.T) {
	got := bannerHost("127.0.0.1:8080")
	if len(got) != 1 || got[0] != "127.0.0.1" {
		t.Fatalf("bannerHost(127.0.0.1:8080) = %v, want [127.0.0.1]", got)
	}
	got = bannerHost("localhost:9000")
	if len(got) != 1 || got[0] != "localhost" {
		t.Fatalf("bannerHost(localhost:9000) = %v, want [localhost]", got)
	}
}

func TestBannerHostWildcardReturnsLAN(t *testing.T) {
	got := bannerHost("0.0.0.0:8080")
	if len(got) == 0 {
		t.Skip("no non-loopback IPv4 interfaces on this host")
	}
	for _, h := range got {
		ip := net.ParseIP(h)
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			t.Fatalf("bannerHost(0.0.0.0) returned non-LAN addr %q", h)
		}
		if ip.To4() == nil {
			t.Fatalf("bannerHost(0.0.0.0) returned non-IPv4 %q", h)
		}
	}
}

func TestPrintBannerWritesToStderr(t *testing.T) {
	// Capture stderr: redirect os.Stderr through a pipe. log.Printf also writes
	// to stderr by default, but printBanner uses fmt.Fprintf(os.Stderr, ...)
	// directly, so this verifies the banner stream without depending on log.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	printBanner("127.0.0.1:8080", StateInfo{})
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(out), "gemini-web-cli server running on") {
		t.Fatalf("banner not written to stderr; got %q", string(out))
	}
	if !strings.Contains(string(out), "http://127.0.0.1:8080/mcp") {
		t.Fatalf("banner missing /mcp URL; got %q", string(out))
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
	if m, err := s.resolveMCPModel("unspecified"); err != nil || m == nil || m.Name != "unspecified" {
		t.Fatalf("override should take precedence, got model=%v err=%v", m, err)
	}
	if m, err := s.resolveMCPModel(""); err != nil || m == nil || m.Name != "gemini-3.5-flash" {
		t.Fatalf("default model not applied, got model=%v err=%v", m, err)
	}
	if _, err := s.resolveMCPModel("missing-model-name"); err == nil {
		t.Fatal("explicit missing model should return an error")
	}

	s2 := &Server{client: c}
	if m, err := s2.resolveMCPModel(""); err != nil || m == nil || m.Name != "unspecified" {
		t.Fatalf("fallback to unspecified failed, got model=%v err=%v", m, err)
	}

	s3 := &Server{client: c, mcpDefaultModel: "missing-default-model"}
	if _, err := s3.resolveMCPModel(""); err == nil {
		t.Fatal("missing default model should return an error")
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

func TestMCPResearchListRejectsTooLargeCount(t *testing.T) {
	s := &Server{}
	result, err := s.handleMCPResearchList(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"count": maxMCPResearchListCount + 1}},
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected tool-level error result for too-large count")
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
