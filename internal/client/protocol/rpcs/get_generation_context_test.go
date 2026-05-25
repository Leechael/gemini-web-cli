package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeGetGenerationContext_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetGenerationContext("00000000-0000-0000-0000-000000000001")
	if rpcID != "kwDCne" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestDecodeGetGenerationContext_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "get_generation_context_basic.txt", "kwDCne")
	ctx, err := DecodeGetGenerationContext(body)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.ChatID != "c_000000000000001" || ctx.Prompt != "Sample user prompt" || ctx.RequestID != "r_000000000000001" {
		t.Fatalf("ctx = %+v", ctx)
	}
}

func TestDecodeGetGenerationContext_EmptyBody(t *testing.T) {
	if _, err := DecodeGetGenerationContext(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeGetGenerationContext_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetGenerationContext([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
