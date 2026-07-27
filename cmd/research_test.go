package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResearchRunRejectsExplicitModelBeforeNetwork(t *testing.T) {
	requests := 0
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "unexpected outbound request", http.StatusInternalServerError)
	}))
	defer proxyServer.Close()

	oldModelName := modelName
	oldProxy := proxy
	t.Cleanup(func() {
		modelName = oldModelName
		proxy = oldProxy
	})
	modelName = "gemini-3.1-pro"
	proxy = proxyServer.URL

	err := runResearchRun(nil, []string{"model rejection smoke"})
	if err == nil || err.Error() != "deep research only supports model auto/unspecified" {
		t.Fatalf("err = %v", err)
	}
	if requests != 0 {
		t.Fatalf("outbound requests = %d, want 0", requests)
	}
}
