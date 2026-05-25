package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeDeepResearchAck_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeDeepResearchAck("r_000000000000001")
	if rpcID != "PCck7e" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	if payload != `["r_000000000000001"]` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestEncodeDeepResearchAck_WireParity(t *testing.T) {
	_, got := EncodeDeepResearchAck("r_000000000000001")
	wantBytes, _ := json.Marshal([]any{"r_000000000000001"})
	if got != string(wantBytes) {
		t.Fatalf("payload = %s, want %s", got, string(wantBytes))
	}
}

func TestDecodeDeepResearchAck_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "research_deep_research_ack_basic.txt", "PCck7e")
	if err := DecodeDeepResearchAck(body); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeDeepResearchAck_EmptyBody(t *testing.T) {
	if err := DecodeDeepResearchAck(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeDeepResearchAck_MalformedJSON(t *testing.T) {
	if err := DecodeDeepResearchAck([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
