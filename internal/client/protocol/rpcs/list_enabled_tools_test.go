package rpcs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodeListEnabledTools_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeListEnabledTools()
	if rpcID != "cYRIkd" {
		t.Fatalf("rpcID = %q, want cYRIkd", rpcID)
	}
	if payload != `["en"]` {
		t.Fatalf("payload = %s", payload)
	}
	var decoded []any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
}

func TestDecodeListEnabledTools_FromSampleFixture(t *testing.T) {
	tools, err := DecodeListEnabledTools(rpcFixtureBody(t, "list_enabled_tools_basic.txt", "cYRIkd"))
	if err != nil {
		t.Fatalf("DecodeListEnabledTools: %v", err)
	}
	if len(tools) < 1 {
		t.Fatalf("len(tools) = %d, want at least 1", len(tools))
	}
	if tools[0].Name != "OpenStax" {
		t.Fatalf("tools[0].Name = %q, want OpenStax", tools[0].Name)
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Fatalf("tool has empty Name: %#v", tool)
		}
		if tool.IconURL != "" && !strings.HasPrefix(tool.IconURL, "https://") {
			t.Fatalf("IconURL = %q, want https URL", tool.IconURL)
		}
	}
}

func TestDecodeListEnabledTools_EmptyBody(t *testing.T) {
	if _, err := DecodeListEnabledTools(nil); err == nil {
		t.Fatalf("DecodeListEnabledTools(nil) error = nil")
	}
}

func TestDecodeListEnabledTools_MalformedJSON(t *testing.T) {
	if _, err := DecodeListEnabledTools([]byte(`{not-json`)); err == nil {
		t.Fatalf("DecodeListEnabledTools(malformed) error = nil")
	}
}
