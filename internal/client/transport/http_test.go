package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPostBatch(t *testing.T) {
	var gotForm url.Values
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm
		gotUA = r.Header.Get("User-Agent")
		if r.Header.Get("X-Same-Domain") != "1" {
			t.Fatalf("X-Same-Domain = %q, want 1", r.Header.Get("X-Same-Domain"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("raw-body"))
	}))
	defer srv.Close()

	body, err := PostBatch(t.Context(), PostBatchRequest{
		Client:      srv.Client(),
		URL:         srv.URL,
		AccessToken: "token",
		RPCID:       "rpc",
		Payload:     "[1]",
		UserAgent:   "test-agent",
	})
	if err != nil {
		t.Fatalf("PostBatch: %v", err)
	}
	if string(body) != "raw-body" {
		t.Fatalf("body = %q, want raw-body", body)
	}
	if gotUA != "test-agent" {
		t.Fatalf("User-Agent = %q, want test-agent", gotUA)
	}
	if gotForm.Get("at") != "token" {
		t.Fatalf("at = %q, want token", gotForm.Get("at"))
	}

	var outer [][][]any
	if err := json.Unmarshal([]byte(gotForm.Get("f.req")), &outer); err != nil {
		t.Fatalf("f.req is not expected JSON: %v", err)
	}
	if outer[0][0][0] != "rpc" || outer[0][0][1] != "[1]" || outer[0][0][3] != "generic" {
		t.Fatalf("unexpected f.req = %s", gotForm.Get("f.req"))
	}
}

func TestPostBatchMulti(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("raw-body"))
	}))
	defer srv.Close()

	body, err := PostBatchMulti(t.Context(), PostBatchMultiRequest{
		Client:      srv.Client(),
		URL:         srv.URL,
		AccessToken: "token",
		Calls: []RPCCall{
			{ID: "one", Payload: "[1]"},
			{ID: "two", Payload: "[2]"},
		},
	})
	if err != nil {
		t.Fatalf("PostBatchMulti: %v", err)
	}
	if string(body) != "raw-body" {
		t.Fatalf("body = %q, want raw-body", body)
	}
	if gotForm.Get("at") != "token" {
		t.Fatalf("at = %q, want token", gotForm.Get("at"))
	}
	var outer [][][]any
	if err := json.Unmarshal([]byte(gotForm.Get("f.req")), &outer); err != nil {
		t.Fatalf("f.req is not expected JSON: %v", err)
	}
	if len(outer) != 2 {
		t.Fatalf("len(outer) = %d, want 2", len(outer))
	}
	if outer[0][0][0] != "one" || outer[0][0][1] != "[1]" || outer[1][0][0] != "two" || outer[1][0][1] != "[2]" {
		t.Fatalf("unexpected f.req = %s", gotForm.Get("f.req"))
	}
}

func TestPostBatch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := PostBatch(t.Context(), PostBatchRequest{Client: srv.Client(), URL: srv.URL, RPCID: "rpc", Payload: "[]"})
	if err == nil {
		t.Fatalf("PostBatch error = nil")
	}
}
