package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPostUpload(t *testing.T) {
	var mu sync.Mutex
	var sawStart bool
	var sawFinalize bool
	var handlerFailures []string
	recordFailure := func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		handlerFailures = append(handlerFailures, fmt.Sprintf(format, args...))
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			mu.Lock()
			sawStart = true
			mu.Unlock()
			if r.Header.Get("X-Goog-Upload-Command") != "start" {
				recordFailure("start command = %q", r.Header.Get("X-Goog-Upload-Command"))
			}
			if r.Header.Get("X-Goog-Upload-Protocol") != "resumable" {
				recordFailure("upload protocol = %q", r.Header.Get("X-Goog-Upload-Protocol"))
			}
			if r.Header.Get("X-Goog-Upload-Header-Content-Length") != "11" {
				recordFailure("content length header = %q", r.Header.Get("X-Goog-Upload-Header-Content-Length"))
			}
			if r.Header.Get("Push-Id") != "push-id" {
				recordFailure("Push-Id = %q", r.Header.Get("Push-Id"))
			}
			if r.Header.Get("Cookie") != "a=b" {
				recordFailure("Cookie = %q", r.Header.Get("Cookie"))
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "File name: sample.txt" {
				recordFailure("start body = %q", body)
			}
			w.Header().Set("X-Goog-Upload-Url", server.URL+"/session")
			w.WriteHeader(http.StatusOK)
		case "/session":
			mu.Lock()
			sawFinalize = true
			mu.Unlock()
			if r.Header.Get("X-Goog-Upload-Command") != "upload, finalize" {
				recordFailure("finalize command = %q", r.Header.Get("X-Goog-Upload-Command"))
			}
			if r.Header.Get("X-Goog-Upload-Offset") != "0" {
				recordFailure("offset = %q", r.Header.Get("X-Goog-Upload-Offset"))
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "hello world" {
				recordFailure("finalize body = %q", body)
			}
			_, _ = w.Write([]byte("upload_000000000000001\n"))
		default:
			recordFailure("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	uploadID, err := PostUpload(context.Background(), UploadRequest{
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
	if uploadID != "upload_000000000000001" {
		t.Fatalf("uploadID = %q", uploadID)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(handlerFailures) > 0 {
		t.Fatalf("handler failures: %s", strings.Join(handlerFailures, "; "))
	}
	if !sawStart || !sawFinalize {
		t.Fatalf("sawStart=%v sawFinalize=%v", sawStart, sawFinalize)
	}
}
