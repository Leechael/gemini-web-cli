package server

import (
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
