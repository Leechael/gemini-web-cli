package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUserProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `[[["me",1,["user-1",[],[[null,"Test User"]],[[null,"https://example.com/photo.jpg"]],null,null,null,null,null,[[null,"test@example.com"]]],null,[]]]]`
		_, _ = w.Write(makeTestBatchResponse("o30O0e", body, 0))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID = 1
	c.httpClient = srv.Client()

	profile, err := c.GetUserProfile(t.Context())
	if err != nil {
		t.Fatalf("GetUserProfile: %v", err)
	}
	if profile.UserID != "user-1" {
		t.Fatalf("UserID = %q", profile.UserID)
	}
	if profile.DisplayName != "Test User" {
		t.Fatalf("DisplayName = %q", profile.DisplayName)
	}
	if profile.Email != "test@example.com" {
		t.Fatalf("Email = %q", profile.Email)
	}
	if profile.PhotoURL != "https://example.com/photo.jpg" {
		t.Fatalf("PhotoURL = %q", profile.PhotoURL)
	}
}
