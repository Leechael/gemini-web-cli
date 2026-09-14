package client

import "testing"

func TestIsHTTPURL(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://example.com/path", true},
		{"http://user@host:8080", true},
		{"http://[::1]/path", true},
		{"HTTPS://Example.COM", true},
		{"http:foo", false},
		{"//host/path", false},
		{"http:///", false},
		{"http://:80", false},
		{"ftp://example.com", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsHTTPURL(tc.raw); got != tc.want {
			t.Errorf("IsHTTPURL(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}
