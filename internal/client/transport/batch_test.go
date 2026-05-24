package transport

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildBatchURL(t *testing.T) {
	got := BuildBatchURL(BatchURLConfig{
		BaseURL:     "https://gemini.google.com/",
		AccountPath: "/u/2",
		RPCIDs:      []string{"a", "b"},
		ReqID:       12345,
		Language:    "fr",
		BuildLabel:  "bl",
		SessionID:   "sid",
		SourcePath:  "/u/2/app/c_1",
	})
	if !strings.HasPrefix(got, "https://gemini.google.com/u/2/_/BardChatUi/data/batchexecute?") {
		t.Fatalf("url path = %s", got)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	checks := map[string]string{
		"rpcids":      "a,b",
		"_reqid":      "12345",
		"rt":          "c",
		"hl":          "fr",
		"pageId":      "none",
		"source-path": "/u/2/app/c_1",
		"bl":          "bl",
		"f.sid":       "sid",
	}
	for key, want := range checks {
		if got := q.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q", key, got, want)
		}
	}
}

func TestBuildBatchURL_Defaults(t *testing.T) {
	got := BuildBatchURL(BatchURLConfig{RPCIDs: []string{"rpc"}, ReqID: 1})
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("hl") != "en" {
		t.Fatalf("hl = %q, want en", u.Query().Get("hl"))
	}
	if u.Query().Get("source-path") != "/app" {
		t.Fatalf("source-path = %q, want /app", u.Query().Get("source-path"))
	}
}
