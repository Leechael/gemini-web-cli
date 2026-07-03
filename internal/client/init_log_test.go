package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/transport/rpclog"
)

func TestInitLogsRequestResponse(t *testing.T) {
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
		if r.URL.Path != "/app" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"SNlM0e":"init-token" "cfb2h":"bl" "FdrFJe":"sid" "TuX5cc":"en" "qKIAYe":"feeds/mcudyrk2a4khkz"`))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	t.Cleanup(func() { baseURL = origBase })

	c := newTestClient()
	if err := c.Init(t.Context()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	entry := readLatestInitEntry(t, dir)
	if entry.Kind != rpclog.KindInit {
		t.Fatalf("kind = %q", entry.Kind)
	}
	if !strings.HasSuffix(entry.URL, "/app") {
		t.Fatalf("url = %q", entry.URL)
	}
	if entry.Method != "GET" {
		t.Fatalf("method = %q", entry.Method)
	}
	if entry.Status != 200 {
		t.Fatalf("status = %d", entry.Status)
	}
	if !strings.Contains(entry.RespBody, "init-token") {
		t.Fatalf("resp_body missing token: %s", entry.RespBody)
	}
	if entry.ReqHeaders.Get("User-Agent") == "" {
		t.Fatalf("User-Agent header missing")
	}
}

func readLatestInitEntry(t *testing.T, dir string) rpclog.Entry {
	t.Helper()
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("no log entries")
	}
	var entry rpclog.Entry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return entry
}
