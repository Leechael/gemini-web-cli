package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeBardSettings_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeBardSettings("bard_activity_enabled")
	if rpcID != "ESY5D" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	if payload != `[[["bard_activity_enabled"]]]` {
		t.Fatalf("payload = %s", payload)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
}

func TestEncodeBardSettings_WireParity(t *testing.T) {
	_, got := EncodeBardSettings("bard_activity_enabled")
	wantBytes, _ := json.Marshal([]any{[]any{[]any{"bard_activity_enabled"}}})
	if got != string(wantBytes) {
		t.Fatalf("payload = %s, want %s", got, string(wantBytes))
	}
}

func TestDecodeBardSettings_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "research_bard_settings_basic.txt", "ESY5D")
	if err := DecodeBardSettings(body); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeBardSettings_EmptyBody(t *testing.T) {
	if err := DecodeBardSettings(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeBardSettings_MalformedJSON(t *testing.T) {
	if err := DecodeBardSettings([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
