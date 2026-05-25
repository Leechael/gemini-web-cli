package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostUpload(t *testing.T) {
	var sawStart bool
	var sawFinalize bool

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			sawStart = true
			if r.Header.Get("X-Goog-Upload-Command") != "start" {
				t.Fatalf("start command = %q", r.Header.Get("X-Goog-Upload-Command"))
			}
			if r.Header.Get("X-Goog-Upload-Protocol") != "resumable" {
				t.Fatalf("upload protocol = %q", r.Header.Get("X-Goog-Upload-Protocol"))
			}
			if r.Header.Get("X-Goog-Upload-Header-Content-Length") != "11" {
				t.Fatalf("content length header = %q", r.Header.Get("X-Goog-Upload-Header-Content-Length"))
			}
			if r.Header.Get("Push-Id") != "push-id" {
				t.Fatalf("Push-Id = %q", r.Header.Get("Push-Id"))
			}
			if r.Header.Get("Cookie") != "a=b" {
				t.Fatalf("Cookie = %q", r.Header.Get("Cookie"))
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "File name: sample.txt" {
				t.Fatalf("start body = %q", body)
			}
			w.Header().Set("X-Goog-Upload-Url", server.URL+"/session")
			w.WriteHeader(http.StatusOK)
		case "/session":
			sawFinalize = true
			if r.Header.Get("X-Goog-Upload-Command") != "upload, finalize" {
				t.Fatalf("finalize command = %q", r.Header.Get("X-Goog-Upload-Command"))
			}
			if r.Header.Get("X-Goog-Upload-Offset") != "0" {
				t.Fatalf("offset = %q", r.Header.Get("X-Goog-Upload-Offset"))
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "hello world" {
				t.Fatalf("finalize body = %q", body)
			}
			_, _ = w.Write([]byte("upload_000000000000001\n"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
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
	if !sawStart || !sawFinalize {
		t.Fatalf("sawStart=%v sawFinalize=%v", sawStart, sawFinalize)
	}
}
