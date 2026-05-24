package rpcs

import (
	"encoding/json"
	"strings"
	"testing"
)

const harGetUserProfileBody = "[[[\"me\",1,[\"user-1234567890\",[],[[null,\"Test User\"]],[[null,\"https://example.com/photo.jpg\"]],null,null,null,null,null,[[null,\"test@example.com\"]]],null,[]]]]"

func TestEncodeGetUserProfile_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetUserProfile()
	if rpcID != "o30O0e" {
		t.Fatalf("rpcID = %q, want o30O0e", rpcID)
	}
	want := `[["me"],[[["person.photo","person.name","person.email"]],null,[1,7]]]`
	if payload != want {
		t.Fatalf("payload = %s, want %s", payload, want)
	}
	var decoded []any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
}

func TestDecodeGetUserProfile_FromHARSample(t *testing.T) {
	profile, err := DecodeGetUserProfile([]byte(harGetUserProfileBody))
	if err != nil {
		t.Fatalf("DecodeGetUserProfile: %v", err)
	}
	if profile.UserID == "" {
		t.Fatalf("UserID is empty")
	}
	if profile.DisplayName == "" {
		t.Fatalf("DisplayName is empty")
	}
	if !strings.Contains(profile.Email, "@") {
		t.Fatalf("Email = %q, want email-like value", profile.Email)
	}
	if !strings.HasPrefix(profile.PhotoURL, "https://") {
		t.Fatalf("PhotoURL = %q, want https URL", profile.PhotoURL)
	}
}

func TestDecodeGetUserProfile_EmptyBody(t *testing.T) {
	if _, err := DecodeGetUserProfile(nil); err == nil {
		t.Fatalf("DecodeGetUserProfile(nil) error = nil")
	}
}

func TestDecodeGetUserProfile_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetUserProfile([]byte(`{not-json`)); err == nil {
		t.Fatalf("DecodeGetUserProfile(malformed) error = nil")
	}
}
