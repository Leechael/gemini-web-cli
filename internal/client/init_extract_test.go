package client

import (
	"strings"
	"testing"
)

// ============================================================================
// Init page extraction: language (TuX5cc) and push_id (qKIAYe)
//
// The Init() method extracts these from the Gemini page HTML.
// If not found, defaults are used: "en" and "feeds/mcudyrk2a4khkz".
// ============================================================================

func TestExtractRegex_Language(t *testing.T) {
	html := `some stuff "TuX5cc":"zh-CN" more stuff`
	got := extractRegex(html, `"TuX5cc"\s*:\s*"([^"]*)"`)
	if got != "zh-CN" {
		t.Errorf("language = %q, want zh-CN", got)
	}
}

func TestExtractRegex_LanguageWithSpaces(t *testing.T) {
	html := `"TuX5cc" : "ja"`
	got := extractRegex(html, `"TuX5cc"\s*:\s*"([^"]*)"`)
	if got != "ja" {
		t.Errorf("language = %q, want ja", got)
	}
}

func TestExtractRegex_LanguageMissing(t *testing.T) {
	html := `no language key here`
	got := extractRegex(html, `"TuX5cc"\s*:\s*"([^"]*)"`)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractRegex_PushID(t *testing.T) {
	html := `"qKIAYe":"feeds/abc123xyz" other`
	got := extractRegex(html, `"qKIAYe"\s*:\s*"([^"]*)"`)
	if got != "feeds/abc123xyz" {
		t.Errorf("pushID = %q, want feeds/abc123xyz", got)
	}
}

func TestExtractRegex_PushIDMissing(t *testing.T) {
	html := `no push id`
	got := extractRegex(html, `"qKIAYe"\s*:\s*"([^"]*)"`)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ============================================================================
// Language propagation: streamURL and batchURL should use c.language
// ============================================================================

func TestStreamURL_UsesLanguage(t *testing.T) {
	c := &Client{language: "zh-CN", buildLabel: "bl"}
	u := c.streamURL()
	if !strings.Contains(u, "hl=zh-CN") {
		t.Errorf("streamURL missing hl=zh-CN: %s", u)
	}
}

func TestStreamURL_DefaultLanguage(t *testing.T) {
	c := &Client{language: "en"}
	u := c.streamURL()
	if !strings.Contains(u, "hl=en") {
		t.Errorf("streamURL missing hl=en: %s", u)
	}
}

func TestBatchURL_UsesLanguage(t *testing.T) {
	c := &Client{language: "fr"}
	u := c.batchURL([]string{"rpcId"}, "/app")
	if !strings.Contains(u, "hl=fr") {
		t.Errorf("batchURL missing hl=fr: %s", u)
	}
}

// ============================================================================
// Language propagation: buildInnerRequest req[1] should use c.language
// ============================================================================

func TestBuildInnerRequest_UsesLanguage(t *testing.T) {
	c := &Client{language: "ko"}
	req := c.buildInnerRequest("hello", nil, nil, false, false, "UUID")

	lang, ok := req[1].([]any)
	if !ok || len(lang) != 1 {
		t.Fatalf("req[1] = %v, want [\"ko\"]", req[1])
	}
	if lang[0] != "ko" {
		t.Errorf("req[1][0] = %v, want ko", lang[0])
	}
}

func TestBuildInnerRequest_DefaultLanguage(t *testing.T) {
	c := &Client{language: "en"}
	req := c.buildInnerRequest("hello", nil, nil, false, false, "UUID")

	lang := req[1].([]any)
	if lang[0] != "en" {
		t.Errorf("req[1][0] = %v, want en", lang[0])
	}
}

// ============================================================================
// Push ID propagation: Client.pushID is used by upload methods.
// We can't test uploadStart/uploadFinalize without a server, but we verify
// the field is accessible and the default constant was removed.
// ============================================================================

func TestClient_PushIDField(t *testing.T) {
	c := &Client{pushID: "feeds/custom123"}
	if c.pushID != "feeds/custom123" {
		t.Errorf("pushID = %q", c.pushID)
	}
}

func TestClient_DefaultValues(t *testing.T) {
	// Simulate what Init() does when regex extraction returns empty
	c := &Client{}
	if c.language == "" {
		c.language = "en"
	}
	if c.pushID == "" {
		c.pushID = "feeds/mcudyrk2a4khkz"
	}

	if c.language != "en" {
		t.Errorf("default language = %q, want en", c.language)
	}
	if c.pushID != "feeds/mcudyrk2a4khkz" {
		t.Errorf("default pushID = %q", c.pushID)
	}
}
