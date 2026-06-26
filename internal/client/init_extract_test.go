package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	c := &Client{}
	req := c.buildInnerRequest("hello", nil, nil, nil, false, "UUID", "ko", "")

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
	req := c.buildInnerRequest("hello", nil, nil, nil, false, "UUID", "en", "")

	lang := req[1].([]any)
	if lang[0] != "en" {
		t.Errorf("req[1][0] = %v, want en", lang[0])
	}
}

// ============================================================================
// Init() fallback defaults: when HTML lacks TuX5cc/qKIAYe, defaults apply.
// Uses httptest to exercise the real Init() code path.
// ============================================================================

// fakeInitServer returns an httptest.Server that serves a Gemini-like HTML page.
// tokenKey "SNlM0e" is always present (required for Init to succeed).
// extraHTML is appended to include or omit TuX5cc/qKIAYe keys.
func fakeInitServer(extraHTML string) *httptest.Server {
	html := fmt.Sprintf(`"SNlM0e":"faketoken123" "cfb2h":"bl" "FdrFJe":"sid" %s`, extraHTML)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(html))
	}))
}

func TestInit_FallbackDefaults(t *testing.T) {
	// HTML has SNlM0e but no TuX5cc or qKIAYe → defaults should apply
	srv := fakeInitServer("")
	defer srv.Close()

	// Temporarily override baseURL
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	if err := c.Init(t.Context()); err != nil {
		t.Fatalf("Init() = %v", err)
	}

	if c.language != "en" {
		t.Errorf("language = %q, want en", c.language)
	}
	if c.pushID != "feeds/mcudyrk2a4khkz" {
		t.Errorf("pushID = %q, want feeds/mcudyrk2a4khkz", c.pushID)
	}
}

func TestInit_ExtractsLanguageAndPushID(t *testing.T) {
	srv := fakeInitServer(`"TuX5cc":"ja" "qKIAYe":"feeds/custom999"`)
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	if err := c.Init(t.Context()); err != nil {
		t.Fatalf("Init() = %v", err)
	}

	if c.language != "ja" {
		t.Errorf("language = %q, want ja", c.language)
	}
	if c.pushID != "feeds/custom999" {
		t.Errorf("pushID = %q, want feeds/custom999", c.pushID)
	}
}

// ============================================================================
// Push ID propagation: resumableUpload forwards c.pushID through transport.
// Uses httptest to capture the actual HTTP requests.
// ============================================================================

func TestResumableUpload_SendsPushIDHeader(t *testing.T) {
	var startPushID string
	var finalizePushID string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			startPushID = r.Header.Get("Push-Id")
			w.Header().Set("X-Goog-Upload-Url", srv.URL+"/target")
			w.WriteHeader(200)
		case "/target":
			finalizePushID = r.Header.Get("Push-Id")
			w.WriteHeader(200)
			_, _ = w.Write([]byte("upload-id-123"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	origUpload := uploadURL
	uploadURL = srv.URL + "/"
	defer func() { uploadURL = origUpload }()

	c := newTestClient()
	c.pushID = "feeds/my-custom-push-id"

	_, err := c.resumableUpload(t.Context(), strings.NewReader("data"), "test.txt", "text/plain", 4)
	if err != nil {
		t.Fatalf("resumableUpload() = %v", err)
	}
	if startPushID != "feeds/my-custom-push-id" {
		t.Errorf("start Push-Id header = %q, want feeds/my-custom-push-id", startPushID)
	}
	if finalizePushID != "feeds/my-custom-push-id" {
		t.Errorf("finalize Push-Id header = %q, want feeds/my-custom-push-id", finalizePushID)
	}
}

// newTestClient creates a minimal Client with an HTTP client (no cookies needed for tests).
func newTestClient() *Client {
	c, _ := New(Config{})
	return c
}
