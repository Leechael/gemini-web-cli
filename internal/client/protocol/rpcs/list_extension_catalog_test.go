package rpcs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodeListExtensionCatalog_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeListExtensionCatalog()
	if rpcID != "uPDUsc" {
		t.Fatalf("rpcID = %q, want uPDUsc", rpcID)
	}
	if payload != `["en"]` {
		t.Fatalf("payload = %s", payload)
	}
	var decoded []any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
}

func TestDecodeListExtensionCatalog_FromSampleFixture(t *testing.T) {
	extensions, err := DecodeListExtensionCatalog(rpcFixtureBody(t, "list_extension_catalog_basic.txt", "uPDUsc"))
	if err != nil {
		t.Fatalf("DecodeListExtensionCatalog: %v", err)
	}
	if len(extensions) < 1 {
		t.Fatalf("len(extensions) = %d, want at least 1", len(extensions))
	}
	first := extensions[0]
	if first.ID == "" || first.Name == "" || first.Description == "" {
		t.Fatalf("first extension missing fields: %#v", first)
	}
	if first.IconURL != "" && !strings.HasPrefix(first.IconURL, "https://") {
		t.Fatalf("IconURL = %q, want https URL", first.IconURL)
	}
}

func TestDecodeListExtensionCatalog_EmptyBody(t *testing.T) {
	if _, err := DecodeListExtensionCatalog(nil); err == nil {
		t.Fatalf("DecodeListExtensionCatalog(nil) error = nil")
	}
}

func TestDecodeListExtensionCatalog_MalformedJSON(t *testing.T) {
	if _, err := DecodeListExtensionCatalog([]byte(`{not-json`)); err == nil {
		t.Fatalf("DecodeListExtensionCatalog(malformed) error = nil")
	}
}
