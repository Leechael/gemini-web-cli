package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeListFeatureFlags_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeListFeatureFlags()
	if rpcID != "MyzX6c" {
		t.Fatalf("rpcID = %q, want MyzX6c", rpcID)
	}
	if payload != `[]` {
		t.Fatalf("payload = %s", payload)
	}
	var decoded []any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
}

func TestDecodeListFeatureFlags_FromSampleFixture(t *testing.T) {
	flags, err := DecodeListFeatureFlags(rpcFixtureBody(t, "list_feature_flags_basic.txt", "MyzX6c"))
	if err != nil {
		t.Fatalf("DecodeListFeatureFlags: %v", err)
	}
	if len(flags) < 1 {
		t.Fatalf("len(flags) = %d, want at least 1", len(flags))
	}
	if flags[0].ID == "" {
		t.Fatalf("first flag ID is empty")
	}
	if flags[0].Value == nil {
		t.Fatalf("first flag Value is nil")
	}
}

func TestDecodeListFeatureFlags_StringID(t *testing.T) {
	flags, err := DecodeListFeatureFlags([]byte(`[false,[["string_flag",true,1]]]`))
	if err != nil {
		t.Fatalf("DecodeListFeatureFlags: %v", err)
	}
	if len(flags) != 1 || flags[0].ID != "string_flag" {
		t.Fatalf("flags = %#v", flags)
	}
}

func TestDecodeListFeatureFlags_EmptyBody(t *testing.T) {
	if _, err := DecodeListFeatureFlags(nil); err == nil {
		t.Fatalf("DecodeListFeatureFlags(nil) error = nil")
	}
}

func TestDecodeListFeatureFlags_MalformedJSON(t *testing.T) {
	if _, err := DecodeListFeatureFlags([]byte(`{not-json`)); err == nil {
		t.Fatalf("DecodeListFeatureFlags(malformed) error = nil")
	}
}
