// Package rpclog provides full-request logging for outbound Gemini HTTP calls.
//
// It writes one newline-delimited JSON object per request/response pair to a
// file named YYYY-MM-DD.ndjson under a configurable directory. Logging is
// disabled by default; callers enable it explicitly.
package rpclog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	KindBatch          = "batch"
	KindBatchMulti     = "batch_multi"
	KindStream         = "stream"
	KindUploadStart    = "upload_start"
	KindUploadFinalize = "upload_finalize"
	KindInit           = "init"
)

// Body points to a complete request or response body stored under the logger directory.
type Body struct {
	Path string `json:"path"`
	Size int64  `json:"size"`

	data    []byte
	text    string
	capture *BodyCapture
}

// Entry is one request/response log record.
type Entry struct {
	TS         string      `json:"ts"`
	Method     string      `json:"method"`
	URL        string      `json:"url"`
	Kind       string      `json:"kind"`
	ReqHeaders http.Header `json:"req_headers"`
	ReqBody    *Body       `json:"req_body,omitempty"`
	Status     int         `json:"status"`
	RespBody   *Body       `json:"resp_body,omitempty"`
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

// BodyCapture streams a body directly to its final blob file.
type BodyCapture struct {
	mu     sync.Mutex
	file   *os.File
	rel    string
	size   int64
	err    error
	closed bool
}

// Write appends bytes to the body blob without buffering them in memory.
func (c *BodyCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, os.ErrClosed
	}
	n, err := c.file.Write(p)
	c.size += int64(n)
	if err != nil && c.err == nil {
		c.err = err
	}
	return n, err
}

// Body returns the body reference that Logger.Log finalizes.
func (c *BodyCapture) Body() *Body {
	if c == nil {
		return nil
	}
	return &Body{capture: c}
}

func (c *BodyCapture) finish() (string, int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		if err := c.file.Close(); err != nil && c.err == nil {
			c.err = err
		}
		c.closed = true
	}
	return c.rel, c.size, c.err
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

// Enabled reports whether the package-level logger is enabled.
func Enabled() bool { return defaultLogger.Enabled() }

// SetDir sets the output directory for the package-level logger.
func SetDir(dir string) { defaultLogger.SetDir(dir) }

// StartBodyCapture streams a body to a blob file when logging is enabled.
func StartBodyCapture(label string) *BodyCapture {
	capture, err := defaultLogger.NewBodyCapture(label)
	if err != nil {
		log.Printf("rpc log: start %s body capture: %v", label, err)
		return nil
	}
	return capture
}

// Enabled reports whether this logger is enabled.
func (l *Logger) Enabled() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enabled && l.dir != ""
}

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
	if l.dir == dir {
		return
	}
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	l.date = ""
	l.dir = dir
}

// NewBodyCapture starts streaming a body to a blob file. It returns nil when
// logging is disabled.
func (l *Logger) NewBodyCapture(label string) (*BodyCapture, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled || l.dir == "" {
		return nil, nil
	}
	today := time.Now().Format("2006-01-02")
	blobDir := filepath.Join(l.dir, "blobs", today)
	if err := os.MkdirAll(blobDir, 0750); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(blobDir, label+"-*.blob")
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(l.dir, file.Name())
	if err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	return &BodyCapture{file: file, rel: filepath.ToSlash(rel)}, nil
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
		l.cleanupOldFilesLocked()
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
	if err := l.materializeBodyLocked(e.ReqBody, "req", today); err != nil {
		return err
	}
	if err := l.materializeBodyLocked(e.RespBody, "resp", today); err != nil {
		return err
	}

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(l.file, "%s\n", data); err != nil {
		return err
	}
	return nil
}

// Log writes a single entry using the package-level logger and reports failures.
func Log(ctx context.Context, e Entry) {
	if err := defaultLogger.Log(ctx, e); err != nil {
		log.Printf("rpc log: write entry: %v", err)
	}
}

var (
	atRe             = regexp.MustCompile(`(^|&)at=[^&]*`)
	initCredentialRe = regexp.MustCompile(`("(?:SNlM0e|FdrFJe)"\s*:\s*")[^"]*(")`)
)

// RedactAT replaces `at=<token>` form values with `at=<redacted>`.
func RedactAT(s string) string {
	return atRe.ReplaceAllStringFunc(s, func(m string) string {
		if strings.HasPrefix(m, "&") {
			return "&at=<redacted>"
		}
		return "at=<redacted>"
	})
}

// RedactInitBody removes credentials embedded in the Gemini app HTML.
func RedactInitBody(body []byte) []byte {
	return initCredentialRe.ReplaceAll(body, []byte(`${1}<redacted>${2}`))
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

// BytesBody prepares a complete byte body for blob storage when the entry is logged.
func BytesBody(b []byte) *Body {
	return &Body{data: b}
}

// StringBody prepares a complete string body for blob storage when the entry is logged.
func StringBody(s string) *Body {
	return &Body{text: s}
}

func (l *Logger) materializeBodyLocked(body *Body, label, today string) error {
	if body == nil || body.Path != "" {
		return nil
	}
	if body.capture != nil {
		path, size, err := body.capture.finish()
		if err != nil {
			return err
		}
		body.Path = path
		body.Size = size
		body.capture = nil
		return nil
	}
	blobDir := filepath.Join(l.dir, "blobs", today)
	if err := os.MkdirAll(blobDir, 0750); err != nil {
		return err
	}
	file, err := os.CreateTemp(blobDir, label+"-*.blob")
	if err != nil {
		return err
	}
	path := file.Name()
	removeOnError := true
	defer func() {
		_ = file.Close()
		if removeOnError {
			_ = os.Remove(path)
		}
	}()

	var size int64
	if body.data != nil {
		n, writeErr := file.Write(body.data)
		size = int64(n)
		if writeErr != nil {
			return writeErr
		}
	} else {
		n, writeErr := io.WriteString(file, body.text)
		size = int64(n)
		if writeErr != nil {
			return writeErr
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	rel, err := filepath.Rel(l.dir, path)
	if err != nil {
		return err
	}
	body.Path = filepath.ToSlash(rel)
	body.Size = size
	body.data = nil
	body.text = ""
	removeOnError = false
	return nil
}

// StreamReadCloser copies a stream response directly to a body blob and logs
// it when the stream ends, fails, or is closed.
type StreamReadCloser struct {
	rc      io.ReadCloser
	capture *BodyCapture
	err     error
	logged  bool
	mu      sync.Mutex
	fn      func(*Body, error)
}

// WrapStreamReadCloser wraps rc with disk-backed body capture.
func WrapStreamReadCloser(rc io.ReadCloser, capture *BodyCapture, fn func(*Body, error)) *StreamReadCloser {
	return &StreamReadCloser{rc: rc, capture: capture, fn: fn}
}

func (r *StreamReadCloser) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	if n > 0 && r.capture != nil {
		_, _ = r.capture.Write(p[:n])
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
	r.fn(r.capture.Body(), r.err)
}

// CaptureReadCloser copies reads directly to capture while preserving the
// original closer.
func CaptureReadCloser(rc io.ReadCloser, capture *BodyCapture) io.ReadCloser {
	if rc == nil || capture == nil {
		return rc
	}
	return &capturingReadCloser{Reader: io.TeeReader(rc, capture), Closer: rc}
}

type capturingReadCloser struct {
	io.Reader
	io.Closer
}

func (l *Logger) cleanupOldFilesLocked() {
	if l.dir == "" {
		return
	}
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	entries, err := os.ReadDir(l.dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ndjson") {
				continue
			}
			info, infoErr := entry.Info()
			if infoErr == nil && info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(l.dir, entry.Name()))
			}
		}
	}

	blobRoot := filepath.Join(l.dir, "blobs")
	blobDirs, err := os.ReadDir(blobRoot)
	if err != nil {
		return
	}
	for _, entry := range blobDirs {
		if !entry.IsDir() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(blobRoot, entry.Name()))
		}
	}
}

// Compile-time interface check.
var _ io.ReadCloser = (*StreamReadCloser)(nil)
