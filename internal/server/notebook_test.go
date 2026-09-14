package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotebookRoutesRequireAuth(t *testing.T) {
	s := &Server{mux: http.NewServeMux(), apiKey: "secret"}
	s.registerRoutes()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/notebooks"},
		{http.MethodGet, "/v1/notebooks/nb-1"},
		{http.MethodGet, "/v1/notebooks/nb-1/chats"},
		{http.MethodPost, "/v1/notebooks/nb-1/sources"},
		{http.MethodDelete, "/v1/notebooks/nb-1/sources/src-1"},
	}
	for _, rt := range routes {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest(rt.method, rt.path, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want %d", rt.method, rt.path, w.Code, http.StatusUnauthorized)
		}
	}

	auth := httptest.NewRequest(http.MethodPost, "/v1/notebooks", strings.NewReader(`{}`))
	auth.Header.Set("Authorization", "Bearer secret")
	auth.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, auth)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("authorized create still returned 401")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("authorized empty title status = %d, want 400", w.Code)
	}
}

func TestNotebookCreateRejectsOversizeBody(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.registerRoutes()
	body := `{"title":"` + strings.Repeat("x", maxRequestBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notebooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestNotebookAddSourceValidation(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.registerRoutes()

	cases := []struct {
		body string
		want int
	}{
		{`{}`, http.StatusBadRequest},
		{`{"url":"ftp://example.com"}`, http.StatusBadRequest},
		{`{"url":"https://"}`, http.StatusBadRequest},
		{`{"url":"http://:80"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/v1/notebooks/nb-1/sources", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("body %s: status = %d, want %d", tc.body, w.Code, tc.want)
		}
	}
}
