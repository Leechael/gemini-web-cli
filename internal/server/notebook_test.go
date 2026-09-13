package server

import (
	"net/http"
	"net/http/httptest"
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
}
