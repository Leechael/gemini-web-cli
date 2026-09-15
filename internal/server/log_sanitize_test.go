package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSanitizeUpstreamErrorRedactsGeminiQuery(t *testing.T) {
	input := `stream request failed: Post "https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate?_reqid=572208&bl=label&f.sid=secret&hl=en": unexpected EOF`
	got := sanitizeUpstreamError(input)
	if strings.Contains(got, "f.sid=secret") || strings.Contains(got, "_reqid=572208") {
		t.Fatalf("query was not redacted: %s", got)
	}
	if !strings.Contains(got, "https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate?redacted") {
		t.Fatalf("sanitized URL missing: %s", got)
	}
	if !strings.Contains(got, "unexpected EOF") {
		t.Fatalf("error context missing: %s", got)
	}
}

func TestSanitizeUpstreamErrorRedactsSecrets(t *testing.T) {
	input := `load failed Cookie: __Secure-1PSID=g.a000secret; SID=abc123 Bearer ya29.token at=sometoken&bl=1`
	got := sanitizeUpstreamError(input)
	for _, leak := range []string{"g.a000secret", "abc123", "ya29.token", "sometoken"} {
		if strings.Contains(got, leak) {
			t.Fatalf("leaked %q in %s", leak, got)
		}
	}
	if !strings.Contains(got, "__Secure-1PSID=<redacted>") || !strings.Contains(got, "SID=<redacted>") {
		t.Fatalf("cookie names missing: %s", got)
	}
	if !strings.Contains(got, "Bearer <redacted>") {
		t.Fatalf("bearer missing: %s", got)
	}
	if !strings.Contains(got, "at=<redacted>") {
		t.Fatalf("at missing: %s", got)
	}

	got = sanitizeUpstreamError(`refresh failed PSIDTS=g.a000tssecret`)
	if strings.Contains(got, "g.a000tssecret") {
		t.Fatalf("PSIDTS leaked: %s", got)
	}
	if !strings.Contains(got, "PSIDTS=<redacted>") {
		t.Fatalf("PSIDTS not redacted: %s", got)
	}
}

func TestAccessLogOmitsQueryAndHeaders(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	h := accessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/models?key=secret", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Cookie", "__Secure-1PSID=g.a000secret")
	req.RemoteAddr = "1.2.3.4:1234"
	h.ServeHTTP(httptest.NewRecorder(), req)

	got := buf.String()
	for _, leak := range []string{"secret", "key=", "Bearer", "PSID", "g.a000"} {
		if strings.Contains(got, leak) {
			t.Fatalf("leaked %q in %s", leak, got)
		}
	}
	if !strings.Contains(got, `GET "/v1/models"`) || !strings.Contains(got, "status=204") {
		t.Fatalf("missing access line: %s", got)
	}
}

func TestAccessLogQuotesCRLFPath(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	h := accessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.URL.Path = "/v1/x\nINJECTED"
	h.ServeHTTP(httptest.NewRecorder(), req)
	got := buf.String()
	if strings.Contains(got, "\nINJECTED") {
		t.Fatalf("CRLF path was not quoted: %q", got)
	}
}
