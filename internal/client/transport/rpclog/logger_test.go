package rpclog

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoggerDisabledByDefault(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	if err := l.Log(context.Background(), Entry{Method: "GET", URL: "https://example.com"}); err != nil {
		t.Fatalf("Log: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files, got %d", len(entries))
	}
}

func TestLoggerWritesEntry(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()

	if err := l.Log(context.Background(), Entry{
		Method: "POST",
		URL:    "https://gemini.google.com/batch",
		Kind:   KindBatch,
		Status: 200,
		RPCIDs: []string{"rpc1"},
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, today+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Method != "POST" {
		t.Fatalf("method = %q", got.Method)
	}
	if got.URL != "https://gemini.google.com/batch" {
		t.Fatalf("url = %q", got.URL)
	}
	if got.Kind != KindBatch {
		t.Fatalf("kind = %q", got.Kind)
	}
	if got.Status != 200 {
		t.Fatalf("status = %d", got.Status)
	}
	if len(got.RPCIDs) != 1 || got.RPCIDs[0] != "rpc1" {
		t.Fatalf("rpc_ids = %v", got.RPCIDs)
	}
	if got.TS == "" {
		t.Fatalf("ts not set")
	}
}

func TestLoggerAppendsMultipleEntries(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()

	for i := 0; i < 3; i++ {
		if err := l.Log(context.Background(), Entry{Method: "GET", URL: "https://example.com"}); err != nil {
			t.Fatalf("Log: %v", err)
		}
	}

	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestLoggerCleanupOldFiles(t *testing.T) {
	dir := t.TempDir()

	oldFile := filepath.Join(dir, time.Now().Add(-8*24*time.Hour).Format("2006-01-02")+".ndjson")
	if err := os.WriteFile(oldFile, []byte("old\n"), 0644); err != nil {
		t.Fatalf("WriteFile old: %v", err)
	}
	if err := os.Chtimes(oldFile, time.Now().Add(-8*24*time.Hour), time.Now().Add(-8*24*time.Hour)); err != nil {
		t.Fatalf("Chtimes old: %v", err)
	}
	recentFile := filepath.Join(dir, time.Now().Add(-1*24*time.Hour).Format("2006-01-02")+".ndjson")
	if err := os.WriteFile(recentFile, []byte("recent\n"), 0644); err != nil {
		t.Fatalf("WriteFile recent: %v", err)
	}
	if err := os.Chtimes(recentFile, time.Now().Add(-1*24*time.Hour), time.Now().Add(-1*24*time.Hour)); err != nil {
		t.Fatalf("Chtimes recent: %v", err)
	}

	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	if entries[0].Name() != filepath.Base(recentFile) {
		t.Fatalf("kept wrong file: %s", entries[0].Name())
	}
}

func TestRedactAT(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"at=secret123", "at=<redacted>"},
		{"f.req=%5B%5D&at=secret123", "f.req=%5B%5D&at=<redacted>"},
		{"at=secret123&f.req=%5B%5D", "at=<redacted>&f.req=%5B%5D"},
		{"f.req=%5B%5D", "f.req=%5B%5D"},
		{"", ""},
	}
	for _, tc := range cases {
		got := RedactAT(tc.in)
		if got != tc.want {
			t.Fatalf("RedactAT(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRedactHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Cookie", "__Secure-1PSID=abc123; __Secure-1PSIDTS=def456")
	h.Set("Authorization", "Bearer ya29.xxx")
	h.Set("Content-Type", "application/json")

	got := RedactHeaders(h)

	if got.Get("Cookie") != "__Secure-1PSID=<redacted>; __Secure-1PSIDTS=<redacted>" {
		t.Fatalf("cookie = %q", got.Get("Cookie"))
	}
	if got.Get("Authorization") != "Bearer <redacted>" {
		t.Fatalf("authorization = %q", got.Get("Authorization"))
	}
	if got.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type = %q", got.Get("Content-Type"))
	}
	if h.Get("Cookie") != "__Secure-1PSID=abc123; __Secure-1PSIDTS=def456" {
		t.Fatalf("original cookie was mutated: %q", h.Get("Cookie"))
	}
}

func TestRedactHeadersCaseInsensitive(t *testing.T) {
	h := http.Header{}
	h.Set("cookie", "session=secret")
	h.Set("authorization", "Basic dXNlcjpwYXNz")

	got := RedactHeaders(h)

	if got.Get("Cookie") != "session=<redacted>" {
		t.Fatalf("cookie = %q", got.Get("Cookie"))
	}
	if got.Get("Authorization") != "Basic <redacted>" {
		t.Fatalf("authorization = %q", got.Get("Authorization"))
	}
}

func TestLoggerRedactsHeaders(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()

	h := http.Header{}
	h.Set("Cookie", "__Secure-1PSID=abc123; __Secure-1PSIDTS=def456")
	h.Set("Authorization", "Bearer ya29.xxx")

	if err := l.Log(context.Background(), Entry{
		Method:     "POST",
		URL:        "https://gemini.google.com/upload",
		Kind:       KindUploadStart,
		ReqHeaders: h,
		Status:     200,
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ReqHeaders.Get("Cookie") != "__Secure-1PSID=<redacted>; __Secure-1PSIDTS=<redacted>" {
		t.Fatalf("cookie = %q", got.ReqHeaders.Get("Cookie"))
	}
	if got.ReqHeaders.Get("Authorization") != "Bearer <redacted>" {
		t.Fatalf("authorization = %q", got.ReqHeaders.Get("Authorization"))
	}
	if h.Get("Cookie") != "__Secure-1PSID=abc123; __Secure-1PSIDTS=def456" {
		t.Fatalf("original cookie was mutated: %q", h.Get("Cookie"))
	}
}

func TestBytesBodyUTF8AndBinary(t *testing.T) {
	if got := BytesBody([]byte("hello")); got != "hello" {
		t.Fatalf("UTF-8 body = %q", got)
	}
	binary := []byte{0xff, 0xfe}
	got := BytesBody(binary)
	if got == string(binary) {
		t.Fatalf("binary body should be base64 encoded")
	}
}

func TestStreamReadCloserLogsOnEOF(t *testing.T) {
	inner := io.NopCloser(strings.NewReader("stream data"))
	var logged []byte
	var loggedErr error
	wrapped := WrapStreamReadCloser(inner, func(buf []byte, err error) {
		logged = buf
		loggedErr = err
	})

	if _, err := io.Copy(io.Discard, wrapped); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if err := wrapped.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if string(logged) != "stream data" {
		t.Fatalf("logged body = %q", logged)
	}
	if loggedErr != io.EOF {
		t.Fatalf("logged err = %v, want EOF", loggedErr)
	}
}

func TestStreamReadCloserLogsOnError(t *testing.T) {
	inner := &errorReadCloser{err: errors.New("unexpected EOF")}
	var logged []byte
	var loggedErr error
	wrapped := WrapStreamReadCloser(inner, func(buf []byte, err error) {
		logged = buf
		loggedErr = err
	})

	_, _ = io.Copy(io.Discard, wrapped)
	_ = wrapped.Close()

	if loggedErr == nil || loggedErr.Error() != "unexpected EOF" {
		t.Fatalf("logged err = %v", loggedErr)
	}
	if string(logged) != "partial" {
		t.Fatalf("logged body = %q", logged)
	}
}

func TestStreamReadCloserLogsOnCloseWithoutEOF(t *testing.T) {
	inner := io.NopCloser(strings.NewReader("never read"))
	var logged []byte
	wrapped := WrapStreamReadCloser(inner, func(buf []byte, err error) {
		logged = buf
	})

	if err := wrapped.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(logged) != 0 {
		t.Fatalf("expected empty body, got %q", logged)
	}
}

type errorReadCloser struct {
	err error
}

func (r *errorReadCloser) Read(p []byte) (int, error) {
	copy(p, "partial")
	return len("partial"), r.err
}

func (r *errorReadCloser) Close() error { return nil }
