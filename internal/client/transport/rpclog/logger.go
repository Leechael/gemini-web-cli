// Package rpclog provides full-request logging for outbound Gemini HTTP calls.
//
// It writes one newline-delimited JSON object per request/response pair to a
// file named YYYY-MM-DD.ndjson under a configurable directory. Logging is
// disabled by default; callers enable it explicitly.
package rpclog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	KindBatch          = "batch"
	KindBatchMulti     = "batch_multi"
	KindStream         = "stream"
	KindUploadStart    = "upload_start"
	KindUploadFinalize = "upload_finalize"
	KindInit           = "init"
)

// Entry is one request/response log record.
type Entry struct {
	TS         string      `json:"ts"`
	Method     string      `json:"method"`
	URL        string      `json:"url"`
	Kind       string      `json:"kind"`
	ReqHeaders http.Header `json:"req_headers"`
	ReqBody    string      `json:"req_body"`
	Status     int         `json:"status"`
	RespBody   string      `json:"resp_body"`
	RejectCode *int        `json:"reject_code"`
	DurMS      int64       `json:"dur_ms"`
	Error      string      `json:"error"`
	RPCIDs     []string    `json:"rpc_ids"`
}

// Logger writes ndjson request/response logs with daily rotation.
type Logger struct {
	mu      sync.Mutex
	enabled bool
	dir     string
	file    *os.File
	date    string
}

// New creates a disabled logger with the default directory.
func New() *Logger {
	dir := os.Getenv("GEMINI_WEB_CLI_RPC_LOG_DIR")
	if dir == "" {
		dir = "data/rpc_logs"
	}
	return &Logger{dir: dir}
}

var defaultLogger = New()

// Default returns the package-level logger.
func Default() *Logger { return defaultLogger }

// SetDefault replaces the package-level logger. It is intended for tests that
// must avoid mutating shared global state.
func SetDefault(l *Logger) {
	defaultLogger = l
}

// SetEnabled enables or disables the package-level logger.
func SetEnabled(enabled bool) { defaultLogger.SetEnabled(enabled) }

// SetDir sets the output directory for the package-level logger.
func SetDir(dir string) { defaultLogger.SetDir(dir) }

// SetEnabled enables or disables this logger. Enabling triggers cleanup of
// files older than 7 days.
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
	if enabled {
		l.cleanupOldFilesLocked()
	}
}

// SetDir sets the output directory.
func (l *Logger) SetDir(dir string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.dir = dir
}

// Close flushes and closes the current log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	l.date = ""
	return err
}

// Log writes a single entry to today's ndjson file.
func (l *Logger) Log(ctx context.Context, e Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled || l.dir == "" {
		return nil
	}
	if err := os.MkdirAll(l.dir, 0750); err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")
	if l.date != today {
		if l.file != nil {
			_ = l.file.Close()
			l.file = nil
		}
		l.date = today
	}
	if l.file == nil {
		path := filepath.Join(l.dir, today+".ndjson")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
		if err != nil {
			return err
		}
		l.file = f
	}

	if e.TS == "" {
		e.TS = time.Now().Format(time.RFC3339)
	}

	e.ReqHeaders = RedactHeaders(e.ReqHeaders)

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(l.file, "%s\n", data); err != nil {
		return err
	}
	return nil
}

// Log writes a single entry using the package-level logger.
func Log(ctx context.Context, e Entry) error {
	return defaultLogger.Log(ctx, e)
}

var atRe = regexp.MustCompile(`(^|&)at=[^&]*`)

// RedactAT replaces `at=<token>` form values with `at=<redacted>`.
func RedactAT(s string) string {
	return atRe.ReplaceAllStringFunc(s, func(m string) string {
		if strings.HasPrefix(m, "&") {
			return "&at=<redacted>"
		}
		return "at=<redacted>"
	})
}

// RedactHeaders returns a copy of h with sensitive header values redacted.
// Cookie values are replaced by name; Authorization values keep their scheme.
func RedactHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	redacted := h.Clone()
	for key, values := range redacted {
		k := strings.ToLower(key)
		switch k {
		case "cookie":
			for i, v := range values {
				redacted[key][i] = redactCookieValue(v)
			}
		case "authorization":
			for i, v := range values {
				redacted[key][i] = redactAuthorizationValue(v)
			}
		}
	}
	return redacted
}

func redactCookieValue(v string) string {
	pairs := strings.Split(v, "; ")
	for i, p := range pairs {
		idx := strings.Index(p, "=")
		if idx < 0 {
			pairs[i] = "<redacted>"
			continue
		}
		pairs[i] = p[:idx+1] + "<redacted>"
	}
	return strings.Join(pairs, "; ")
}

func redactAuthorizationValue(v string) string {
	fields := strings.Fields(v)
	if len(fields) == 0 {
		return "<redacted>"
	}
	if len(fields) == 1 {
		return "<redacted>"
	}
	return fields[0] + " <redacted>"
}

// BytesBody converts a byte slice to the log string. Valid UTF-8 is kept as
// text; binary content is base64 encoded.
func BytesBody(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// StreamReadCloser wraps a stream response body so the full response can be
// logged after the stream ends or fails.
type StreamReadCloser struct {
	rc     io.ReadCloser
	buf    []byte
	err    error
	logged bool
	mu     sync.Mutex
	fn     func([]byte, error)
}

// WrapStreamReadCloser wraps rc. When the stream reaches EOF, an error, or is
// closed, fn is called once with the accumulated bytes and the final read error
// (io.EOF is passed through unchanged).
func WrapStreamReadCloser(rc io.ReadCloser, fn func([]byte, error)) *StreamReadCloser {
	return &StreamReadCloser{rc: rc, fn: fn}
}

func (r *StreamReadCloser) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	if n > 0 {
		r.buf = append(r.buf, p[:n]...)
	}
	if err != nil {
		r.mu.Lock()
		if r.err == nil {
			r.err = err
		}
		r.mu.Unlock()
		r.flush()
	}
	return n, err
}

func (r *StreamReadCloser) Close() error {
	r.flush()
	return r.rc.Close()
}

func (r *StreamReadCloser) flush() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.logged {
		return
	}
	r.logged = true
	r.fn(r.buf, r.err)
}

// CaptureWriter is an io.Writer that records everything written to it.
type CaptureWriter struct {
	buf []byte
}

func (w *CaptureWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// Bytes returns the captured bytes.
func (w *CaptureWriter) Bytes() []byte {
	return w.buf
}

func (l *Logger) cleanupOldFilesLocked() {
	if l.dir == "" {
		return
	}
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".ndjson") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(l.dir, e.Name()))
		}
	}
}

// Compile-time interface check.
var _ io.ReadCloser = (*StreamReadCloser)(nil)
