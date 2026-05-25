package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeDeepResearchBootstrap_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeDeepResearchBootstrap("en")
	if rpcID != "ku4Jyf" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	if payload != `["en",null,null,null,4,null,null,[2,4,7,15],null,[[5]]]` {
		t.Fatalf("payload = %s", payload)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
}

func TestEncodeDeepResearchBootstrap_DefaultLang(t *testing.T) {
	_, payload := EncodeDeepResearchBootstrap("")
	if payload[:5] != `["en"` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestEncodeDeepResearchBootstrap_WireParity(t *testing.T) {
	_, got := EncodeDeepResearchBootstrap("en")
	wantBytes, _ := json.Marshal([]any{"en", nil, nil, nil, 4, nil, nil, []any{2, 4, 7, 15}, nil, []any{[]any{5}}})
	if got != string(wantBytes) {
		t.Fatalf("payload = %s, want %s", got, string(wantBytes))
	}
}

func TestDecodeDeepResearchBootstrap_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "research_deep_research_bootstrap_basic.txt", "ku4Jyf")
	if err := DecodeDeepResearchBootstrap(body); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeDeepResearchBootstrap_EmptyBody(t *testing.T) {
	if err := DecodeDeepResearchBootstrap(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeDeepResearchBootstrap_MalformedJSON(t *testing.T) {
	if err := DecodeDeepResearchBootstrap([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
