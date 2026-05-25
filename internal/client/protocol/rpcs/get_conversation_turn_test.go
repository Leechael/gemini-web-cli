package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeGetConversationTurn_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetConversationTurn("c_0000000000000001", "r_0000000000000001")
	if rpcID != "EqPOKe" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != "c_0000000000000001" || got[1] != "r_0000000000000001" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestDecodeGetConversationTurn_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "get_conversation_turn_basic.txt", "EqPOKe")
	turn, err := DecodeGetConversationTurn(body)
	if err != nil {
		t.Fatal(err)
	}
	if turn.UserPrompt != "Sample user prompt" || turn.AssistantResponse != "Sample assistant response." {
		t.Fatalf("turn = %+v", turn)
	}
}

func TestDecodeGetConversationTurn_EmptyBody(t *testing.T) {
	if _, err := DecodeGetConversationTurn(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeGetConversationTurn_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetConversationTurn([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
