package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeGetUploadLimits_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetUploadLimits()
	if rpcID != "mhs1xe" {
		t.Fatalf("rpcID = %q, want mhs1xe", rpcID)
	}
	if payload != `[[1,3]]` {
		t.Fatalf("payload = %s", payload)
	}
	var decoded []any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
}

func TestDecodeGetUploadLimits_FromSampleFixture(t *testing.T) {
	limits, err := DecodeGetUploadLimits(rpcFixtureBody(t, "get_upload_limits_basic.txt", "mhs1xe"))
	if err != nil {
		t.Fatalf("DecodeGetUploadLimits: %v", err)
	}
	if limits.MaxFiles != 500 || limits.MaxFileMB != 300 || limits.MaxTotalBytes != 500000 {
		t.Fatalf("limits = %#v", limits)
	}
}

func TestDecodeGetUploadLimits_EmptyBody(t *testing.T) {
	if _, err := DecodeGetUploadLimits(nil); err == nil {
		t.Fatalf("DecodeGetUploadLimits(nil) error = nil")
	}
}

func TestDecodeGetUploadLimits_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetUploadLimits([]byte(`{not-json`)); err == nil {
		t.Fatalf("DecodeGetUploadLimits(malformed) error = nil")
	}
}
