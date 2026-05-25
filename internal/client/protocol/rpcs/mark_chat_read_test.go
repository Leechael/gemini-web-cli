package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeMarkChatRead_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeMarkChatRead("c_000000000000001")
	if rpcID != "k81mDb" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != "c_000000000000001" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestDecodeMarkChatRead_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "mark_chat_read_basic.txt", "k81mDb")
	if err := DecodeMarkChatRead(body); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeMarkChatRead_EmptyBody(t *testing.T) {
	if err := DecodeMarkChatRead([]byte("  \n")); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeMarkChatRead_UnexpectedShape(t *testing.T) {
	if err := DecodeMarkChatRead([]byte(`{"ok":true}`)); err == nil {
		t.Fatal("expected error")
	}
	if err := DecodeMarkChatRead([]byte(`[1]`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeMarkChatRead_MalformedJSON(t *testing.T) {
	if err := DecodeMarkChatRead([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
