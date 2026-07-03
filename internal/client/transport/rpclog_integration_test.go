package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/transport/rpclog"
)

func TestPostBatchLogsRequestResponse(t *testing.T) {
	dir := t.TempDir()
	orig := rpclog.Default()
	l := rpclog.New()
	l.SetDir(dir)
	l.SetEnabled(true)
	rpclog.SetDefault(l)
	defer func() {
		rpclog.SetDefault(orig)
		_ = l.Close()
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`)]}'
77
[["wrb.fr","rpc1","{}",null,null,[13],null,null,null,null,null,["generic"]]]
`))
	}))
	defer srv.Close()

	_, err := PostBatch(t.Context(), PostBatchRequest{
		Client:      srv.Client(),
		URL:         srv.URL,
		AccessToken: "secret-token",
		RPCID:       "rpc1",
		Payload:     "[1]",
	})
	if err != nil {
		t.Fatalf("PostBatch: %v", err)
	}

	entry := readLatestEntry(t, dir)
	if entry.Kind != rpclog.KindBatch {
		t.Fatalf("kind = %q", entry.Kind)
	}
	if !strings.Contains(entry.ReqBody, "at=<redacted>") {
		t.Fatalf("at not redacted in req_body: %s", entry.ReqBody)
	}
	if !strings.Contains(entry.ReqBody, "f.req=") {
		t.Fatalf("f.req missing from req_body: %s", entry.ReqBody)
	}
	if entry.Status != 200 {
		t.Fatalf("status = %d", entry.Status)
	}
	if entry.RejectCode == nil || *entry.RejectCode != 13 {
		t.Fatalf("reject_code = %v", entry.RejectCode)
	}
}

func TestPostStreamGenerateLogsStreamBody(t *testing.T) {
	dir := t.TempDir()
	orig := rpclog.Default()
	l := rpclog.New()
	l.SetDir(dir)
	l.SetEnabled(true)
	rpclog.SetDefault(l)
	defer func() {
		rpclog.SetDefault(orig)
		_ = l.Close()
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stream chunk"))
	}))
	defer srv.Close()

	rc, err := PostStreamGenerate(t.Context(), StreamGenerateRequest{
		Client:      srv.Client(),
		URL:         srv.URL,
		AccessToken: "secret-token",
		InnerReq:    []byte("[]"),
		UUID:        "uuid",
	})
	if err != nil {
		t.Fatalf("PostStreamGenerate: %v", err)
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	_ = rc.Close()
	if string(data) != "stream chunk" {
		t.Fatalf("body = %q", data)
	}

	entry := readLatestEntry(t, dir)
	if entry.Kind != rpclog.KindStream {
		t.Fatalf("kind = %q", entry.Kind)
	}
	if !strings.Contains(entry.ReqBody, "at=<redacted>") {
		t.Fatalf("at not redacted: %s", entry.ReqBody)
	}
	if entry.Status != 200 {
		t.Fatalf("status = %d", entry.Status)
	}
	if entry.RespBody != "stream chunk" {
		t.Fatalf("resp_body = %q", entry.RespBody)
	}
}

func TestPostUploadLogsStartAndFinalize(t *testing.T) {
	dir := t.TempDir()
	orig := rpclog.Default()
	l := rpclog.New()
	l.SetDir(dir)
	l.SetEnabled(true)
	rpclog.SetDefault(l)
	defer func() {
		rpclog.SetDefault(orig)
		_ = l.Close()
	}()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			w.Header().Set("X-Goog-Upload-Url", server.URL+"/session")
			w.WriteHeader(http.StatusOK)
		case "/session":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("upload-id"))
		}
	}))
	defer server.Close()

	_, err := PostUpload(t.Context(), UploadRequest{
		Client:       server.Client(),
		PushURL:      server.URL + "/upload",
		PushID:       "push-id",
		TenantID:     "tenant-id",
		Origin:       "https://gemini.google.com",
		Referer:      "https://gemini.google.com/",
		UserAgent:    "test-agent",
		CookieHeader: "a=b",
		FileName:     "sample.txt",
		Size:         11,
		Body:         strings.NewReader("hello world"),
	})
	if err != nil {
		t.Fatalf("PostUpload: %v", err)
	}

	var sawStart, sawFinalize bool
	for _, entry := range readAllEntries(t, dir) {
		if entry.Kind == rpclog.KindUploadStart {
			sawStart = true
			if entry.ReqBody != "File name: sample.txt" {
				t.Fatalf("upload start req_body = %q", entry.ReqBody)
			}
		}
		if entry.Kind == rpclog.KindUploadFinalize {
			sawFinalize = true
			if entry.ReqBody != "hello world" {
				t.Fatalf("upload finalize req_body = %q", entry.ReqBody)
			}
		}
	}
	if !sawStart {
		t.Fatalf("missing upload_start log")
	}
	if !sawFinalize {
		t.Fatalf("missing upload_finalize log")
	}
}

func TestPostStreamGenerateLogsHTTPError(t *testing.T) {
	dir := t.TempDir()
	orig := rpclog.Default()
	l := rpclog.New()
	l.SetDir(dir)
	l.SetEnabled(true)
	rpclog.SetDefault(l)
	defer func() {
		rpclog.SetDefault(orig)
		_ = l.Close()
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	_, err := PostStreamGenerate(t.Context(), StreamGenerateRequest{
		Client:      srv.Client(),
		URL:         srv.URL,
		AccessToken: "secret-token",
		InnerReq:    []byte("[]"),
		UUID:        "uuid",
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	entry := readLatestEntry(t, dir)
	if entry.Status != 429 {
		t.Fatalf("status = %d", entry.Status)
	}
	if entry.RespBody != "rate limited" {
		t.Fatalf("resp_body = %q", entry.RespBody)
	}
	if entry.Error == "" {
		t.Fatalf("expected error field")
	}
}

func TestPostBatchLogsNetworkError(t *testing.T) {
	dir := t.TempDir()
	orig := rpclog.Default()
	l := rpclog.New()
	l.SetDir(dir)
	l.SetEnabled(true)
	rpclog.SetDefault(l)
	defer func() {
		rpclog.SetDefault(orig)
		_ = l.Close()
	}()

	// No server running; the request will fail.
	_, err := PostBatch(t.Context(), PostBatchRequest{
		Client:      http.DefaultClient,
		URL:         "http://127.0.0.1:1/",
		AccessToken: "secret-token",
		RPCID:       "rpc1",
		Payload:     "[1]",
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	entry := readLatestEntry(t, dir)
	if entry.Status != 0 {
		t.Fatalf("status = %d", entry.Status)
	}
	if entry.Error == "" {
		t.Fatalf("expected error field")
	}
	if !strings.Contains(entry.ReqBody, "at=<redacted>") {
		t.Fatalf("at not redacted: %s", entry.ReqBody)
	}
}

func readLatestEntry(t *testing.T, dir string) rpclog.Entry {
	t.Helper()
	entries := readAllEntries(t, dir)
	if len(entries) == 0 {
		t.Fatalf("no log entries found")
	}
	return entries[len(entries)-1]
}

func readAllEntries(t *testing.T, dir string) []rpclog.Entry {
	t.Helper()
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	var out []rpclog.Entry
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var e rpclog.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("Unmarshal %q: %v", line, err)
		}
		out = append(out, e)
	}
	return out
}

// Ensure the error read closer used elsewhere still satisfies the interface.
var _ error = errors.New("test")
