package client

import "testing"

func TestParseAbuseStatus_Clean(t *testing.T) {
	cases := []string{"", "[]", `[null]`, `[null,null]`, `[null,[]]`}
	for _, body := range cases {
		s := parseAbuseStatus(body)
		if s == nil {
			t.Errorf("body=%q: got nil, want clean status", body)
			continue
		}
		if !s.IsClean {
			t.Errorf("body=%q: IsClean=false, want true", body)
		}
	}
}

func TestParseAbuseStatus_Flagged(t *testing.T) {
	// abuse_info at [1]: status code = 5_000_000 (→ 5 after divide), signal at [3][1]
	body := `[null,[null,5000000,null,[null,"suspicious-traffic"]]]`
	s := parseAbuseStatus(body)
	if s == nil {
		t.Fatal("expected non-nil")
	}
	if s.IsClean {
		t.Error("expected IsClean=false")
	}
	if s.StatusCode != 5 {
		t.Errorf("StatusCode = %d, want 5", s.StatusCode)
	}
	if s.Signal != "suspicious-traffic" {
		t.Errorf("Signal = %q", s.Signal)
	}
}

func TestParseAbuseStatus_Malformed(t *testing.T) {
	if parseAbuseStatus("not json") != nil {
		t.Error("expected nil for malformed JSON")
	}
}
