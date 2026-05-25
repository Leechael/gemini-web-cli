package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeGetChatMetadata_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetChatMetadata("c_000000000000001")
	if rpcID != "MUAZcd" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != nil {
		t.Fatalf("payload = %#v", got)
	}
}

func TestDecodeGetChatMetadata_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "get_chat_metadata_basic.txt", "MUAZcd")
	meta, err := DecodeGetChatMetadata(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Cid != "c_000000000000001" || meta.Title != "Sample chat title" || meta.UpdatedAt != 1700000000 || !meta.Unread {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestDecodeGetChatMetadata_PositionalShape(t *testing.T) {
	body, _ := json.Marshal([]any{[]any{"c_000000000000001", "c_prefixed sample title", float64(1700000000), false, true}})
	meta, err := DecodeGetChatMetadata(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "c_prefixed sample title" || meta.Unread {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestDecodeGetChatMetadata_MissingShape(t *testing.T) {
	body, _ := json.Marshal([]any{[]any{"noise", true}, []any{"c_000000000000001", "Sample chat title", float64(1700000000), false}})
	if _, err := DecodeGetChatMetadata(body); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeGetChatMetadata_EmptyBody(t *testing.T) {
	if _, err := DecodeGetChatMetadata(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeGetChatMetadata_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetChatMetadata([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
