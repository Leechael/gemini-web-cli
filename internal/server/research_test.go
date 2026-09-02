package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

func TestResearchCreateRejectsExplicitModelWithoutStreamRequest(t *testing.T) {
	requests := 0
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "unexpected outbound request", http.StatusInternalServerError)
	}))
	defer proxyServer.Close()

	c, err := client.New(client.Config{Proxy: proxyServer.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	s := &Server{client: c, mux: http.NewServeMux()}
	s.registerRoutes()
	req := httptest.NewRequest(http.MethodPost, "/v1/research", strings.NewReader(`{"prompt":"model rejection smoke","model":"gemini-3.1-pro"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "deep research only supports model auto/unspecified") {
		t.Fatalf("body = %s", w.Body.String())
	}
	if requests != 0 {
		t.Fatalf("outbound requests = %d, want 0", requests)
	}
}

func TestMCPResearchCreateRejectsExplicitModelWithoutStreamRequest(t *testing.T) {
	requests := 0
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "unexpected outbound request", http.StatusInternalServerError)
	}))
	defer proxyServer.Close()

	c, err := client.New(client.Config{Proxy: proxyServer.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	s := &Server{client: c}
	result, err := s.handleMCPResearchCreate(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{
			"prompt": "model rejection smoke",
			"model":  "gemini-3.1-pro",
		}},
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if result == nil || !result.IsError || len(result.Content) != 1 {
		t.Fatalf("result = %#v", result)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok || text.Text != "deep research only supports model auto/unspecified" {
		t.Fatalf("content = %#v", result.Content)
	}
	if requests != 0 {
		t.Fatalf("outbound requests = %d, want 0", requests)
	}
}
