package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

func TestRequireAuthProtectsV1Routes(t *testing.T) {
	s := &Server{mux: http.NewServeMux(), apiKey: "secret"}
	s.registerRoutes()

	unauthorized := httptest.NewRecorder()
	s.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	authorized := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret")
	s.ServeHTTP(authorized, req)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized status = %d, want %d", authorized.Code, http.StatusOK)
	}
}

func TestChatMessageContentArray(t *testing.T) {
	var msg chatMessage
	if err := json.Unmarshal([]byte(`{"role":"user","content":[{"type":"text","text":"hello"},{"type":"text","text":"world"}]}`), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Content != "hello\nworld" {
		t.Fatalf("content = %q, want joined text", msg.Content)
	}
}

func TestChatMessageRejectsUnsupportedContentParts(t *testing.T) {
	var msg chatMessage
	err := json.Unmarshal([]byte(`{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.test/image.png"}}]}`), &msg)
	if err == nil || !strings.Contains(err.Error(), "unsupported content part type") {
		t.Fatalf("err = %v, want unsupported content part type", err)
	}
}

func TestWriteSSEUsesUpdatedChatID(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()

	s.writeSSE(w, "", "gemini-test", func(emit func(chatID, delta, reasoning string) error) error {
		return emit("c_123", "hello", "")
	})

	body := w.Body.String()
	if !strings.Contains(body, `"id":"chatcmpl-c_123"`) {
		t.Fatalf("SSE body missing updated chat id: %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("SSE body missing DONE event: %s", body)
	}
}

func TestReasoningHiddenUnlessEnabled(t *testing.T) {
	out := &types.ModelOutput{Thoughts: "full", ThoughtsDelta: "delta"}

	hidden := &Server{}
	if hidden.reasoningText(out) != "" || hidden.reasoningDelta(out) != "" {
		t.Fatal("reasoning should be hidden by default")
	}

	exposed := &Server{exposeThoughts: true}
	if exposed.reasoningText(out) != "full" || exposed.reasoningDelta(out) != "delta" {
		t.Fatal("reasoning should be exposed when enabled")
	}
}

func TestChatBodySizeLimit(t *testing.T) {
	s := &Server{}
	body := `{"messages":[{"role":"user","content":"` + strings.Repeat("x", maxRequestBodyBytes) + `"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
