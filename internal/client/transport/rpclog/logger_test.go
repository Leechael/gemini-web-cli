package rpclog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
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

func TestLoggerDoesNotCaptureBodiesWhenDisabled(t *testing.T) {
	l := New()
	l.SetDir(t.TempDir())

	capture, err := l.NewBodyCapture("response")
	if err != nil {
		t.Fatalf("NewBodyCapture: %v", err)
	}
	if capture != nil {
		t.Fatalf("capture = %+v, want nil", capture)
	}
}

func TestLoggerStreamsCompleteBodyToBlobFile(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()

	capture, err := l.NewBodyCapture("response")
	if err != nil {
		t.Fatalf("NewBodyCapture: %v", err)
	}
	for _, chunk := range [][]byte{[]byte("first "), []byte("second "), {0xff, 0x00}} {
		if _, err := capture.Write(chunk); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := l.Log(context.Background(), Entry{RespBody: capture.Body()}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	entry := readLatestLoggerEntry(t, dir)
	assertBodyBlob(t, dir, entry.RespBody, []byte{'f', 'i', 'r', 's', 't', ' ', 's', 'e', 'c', 'o', 'n', 'd', ' ', 0xff, 0x00})
}

func TestLoggerStoresBodiesInReferencedBlobFiles(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()

	if err := l.Log(context.Background(), Entry{
		Method:   "POST",
		URL:      "https://gemini.google.com/batch",
		Kind:     KindBatch,
		ReqBody:  BytesBody([]byte("request body")),
		RespBody: BytesBody([]byte{0xff, 0xfe, 0x00}),
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	entry := readLatestLoggerEntry(t, dir)
	assertBodyBlob(t, dir, entry.ReqBody, []byte("request body"))
	assertBodyBlob(t, dir, entry.RespBody, []byte{0xff, 0xfe, 0x00})

	logPath := filepath.Join(dir, time.Now().Format("2006-01-02")+".ndjson")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile log: %v", err)
	}
	if strings.Contains(string(logData), "request body") {
		t.Fatalf("body was embedded in ndjson: %s", logData)
	}
}

func assertBodyBlob(t *testing.T, dir string, body *Body, want []byte) {
	t.Helper()
	if body == nil || body.Path == "" {
		t.Fatalf("body reference = %+v", body)
	}
	if body.Size != int64(len(want)) {
		t.Fatalf("body size = %d, want %d", body.Size, len(want))
	}
	got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(body.Path)))
	if err != nil {
		t.Fatalf("ReadFile body: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("body = %v, want %v", got, want)
	}
}

func readLatestLoggerEntry(t *testing.T, dir string) Entry {
	t.Helper()
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return entry
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

func TestLoggerSetDirSwitchesTheActiveFile(t *testing.T) {
	firstDir := t.TempDir()
	secondDir := t.TempDir()
	l := New()
	l.SetDir(firstDir)
	l.SetEnabled(true)
	defer l.Close()

	if err := l.Log(context.Background(), Entry{URL: "https://example.com/first"}); err != nil {
		t.Fatalf("Log first: %v", err)
	}
	l.SetDir(secondDir)
	if err := l.Log(context.Background(), Entry{URL: "https://example.com/second"}); err != nil {
		t.Fatalf("Log second: %v", err)
	}

	first := readLatestLoggerEntry(t, firstDir)
	if first.URL != "https://example.com/first" {
		t.Fatalf("first URL = %q", first.URL)
	}
	second := readLatestLoggerEntry(t, secondDir)
	if second.URL != "https://example.com/second" {
		t.Fatalf("second URL = %q", second.URL)
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

func TestLoggerCleansOldLogsAndBlobsOnDailyRotation(t *testing.T) {
	dir := t.TempDir()
	l := New()
	l.SetDir(dir)
	l.SetEnabled(true)
	defer l.Close()
	if err := l.Log(context.Background(), Entry{URL: "https://example.com/today"}); err != nil {
		t.Fatalf("Log initial: %v", err)
	}

	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	oldDate := oldTime.Format("2006-01-02")
	oldLog := filepath.Join(dir, oldDate+".ndjson")
	if err := os.WriteFile(oldLog, []byte("old\n"), 0640); err != nil {
		t.Fatalf("WriteFile old log: %v", err)
	}
	oldBlobDir := filepath.Join(dir, "blobs", oldDate)
	if err := os.MkdirAll(oldBlobDir, 0750); err != nil {
		t.Fatalf("MkdirAll old blobs: %v", err)
	}
	oldBlob := filepath.Join(oldBlobDir, "response.blob")
	if err := os.WriteFile(oldBlob, []byte("old body"), 0600); err != nil {
		t.Fatalf("WriteFile old blob: %v", err)
	}
	if err := os.Chtimes(oldLog, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes old log: %v", err)
	}
	if err := os.Chtimes(oldBlob, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes old blob: %v", err)
	}
	if err := os.Chtimes(oldBlobDir, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes old blob dir: %v", err)
	}

	l.mu.Lock()
	l.date = oldDate
	l.mu.Unlock()
	if err := l.Log(context.Background(), Entry{URL: "https://example.com/rotated"}); err != nil {
		t.Fatalf("Log rotated: %v", err)
	}
	if _, err := os.Stat(oldLog); !os.IsNotExist(err) {
		t.Fatalf("old log still exists: %v", err)
	}
	if _, err := os.Stat(oldBlobDir); !os.IsNotExist(err) {
		t.Fatalf("old blob directory still exists: %v", err)
	}
}

func TestPackageLogReportsWriteFailure(t *testing.T) {
	blockingPath := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockingPath, []byte("not a directory"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	logger := New()
	logger.SetDir(filepath.Join(blockingPath, "rpc-logs"))
	logger.SetEnabled(true)
	origLogger := Default()
	SetDefault(logger)
	defer SetDefault(origLogger)

	var output bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(origOutput)

	Log(context.Background(), Entry{URL: "https://example.com"})
	if !strings.Contains(output.String(), "rpc log: write entry:") {
		t.Fatalf("log output = %q", output.String())
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

func TestCaptureReadCloserIgnoresCaptureWriteFailure(t *testing.T) {
	_, _, capture := newStreamCapture(t)
	if _, _, err := capture.finish(); err != nil {
		t.Fatalf("finish capture: %v", err)
	}

	inner := &trackingReadCloser{Reader: strings.NewReader("upload payload")}
	wrapped := CaptureReadCloser(inner, capture)
	got, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "upload payload" {
		t.Fatalf("body = %q, want %q", got, "upload payload")
	}
	if err := wrapped.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !inner.closed {
		t.Fatal("original request body was not closed")
	}
}

func TestStreamReadCloserLogsOnEOF(t *testing.T) {
	dir, logger, capture := newStreamCapture(t)
	inner := io.NopCloser(strings.NewReader("stream data"))
	var loggedErr error
	wrapped := WrapStreamReadCloser(inner, capture, func(body *Body, err error) {
		loggedErr = err
		if logErr := logger.Log(context.Background(), Entry{RespBody: body}); logErr != nil {
			t.Fatalf("Log: %v", logErr)
		}
	})

	if _, err := io.Copy(io.Discard, wrapped); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if err := wrapped.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	entry := readLatestLoggerEntry(t, dir)
	assertBodyBlob(t, dir, entry.RespBody, []byte("stream data"))
	if loggedErr != io.EOF {
		t.Fatalf("logged err = %v, want EOF", loggedErr)
	}
}

func TestStreamReadCloserLogsOnError(t *testing.T) {
	dir, logger, capture := newStreamCapture(t)
	inner := &errorReadCloser{err: errors.New("unexpected EOF")}
	var loggedErr error
	wrapped := WrapStreamReadCloser(inner, capture, func(body *Body, err error) {
		loggedErr = err
		if logErr := logger.Log(context.Background(), Entry{RespBody: body}); logErr != nil {
			t.Fatalf("Log: %v", logErr)
		}
	})

	_, _ = io.Copy(io.Discard, wrapped)
	_ = wrapped.Close()

	if loggedErr == nil || loggedErr.Error() != "unexpected EOF" {
		t.Fatalf("logged err = %v", loggedErr)
	}
	entry := readLatestLoggerEntry(t, dir)
	assertBodyBlob(t, dir, entry.RespBody, []byte("partial"))
}

func TestStreamReadCloserDrainsCompleteBodyOnClose(t *testing.T) {
	dir, logger, capture := newStreamCapture(t)
	inner := io.NopCloser(strings.NewReader("never read"))
	wrapped := WrapStreamReadCloser(inner, capture, func(body *Body, err error) {
		if logErr := logger.Log(context.Background(), Entry{RespBody: body}); logErr != nil {
			t.Fatalf("Log: %v", logErr)
		}
	})

	if err := wrapped.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	entry := readLatestLoggerEntry(t, dir)
	assertBodyBlob(t, dir, entry.RespBody, []byte("never read"))
}

func newStreamCapture(t *testing.T) (string, *Logger, *BodyCapture) {
	t.Helper()
	dir := t.TempDir()
	logger := New()
	logger.SetDir(dir)
	logger.SetEnabled(true)
	t.Cleanup(func() { _ = logger.Close() })
	capture, err := logger.NewBodyCapture("stream-response")
	if err != nil {
		t.Fatalf("NewBodyCapture: %v", err)
	}
	return dir, logger, capture
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

type errorReadCloser struct {
	err error
}

func (r *errorReadCloser) Read(p []byte) (int, error) {
	copy(p, "partial")
	return len("partial"), r.err
}

func (r *errorReadCloser) Close() error { return nil }
