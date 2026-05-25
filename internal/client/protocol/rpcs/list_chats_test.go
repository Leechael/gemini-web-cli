package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeListChats_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeListChats(13, "")
	if rpcID != "MaZiqc" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != float64(13) || got[1] != nil {
		t.Fatalf("payload = %#v", got)
	}
	flags := got[2].([]any)
	if flags[0] != float64(1) || flags[2] != float64(1) {
		t.Fatalf("flags = %#v", flags)
	}
}

func TestEncodeListChats_WithCursor(t *testing.T) {
	_, payload := EncodeListChats(20, "next_cursor")
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != float64(20) || got[1] != "next_cursor" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestDecodeListChats_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "list_chats_basic.txt", "MaZiqc")
	items, cursor, err := DecodeListChats(body)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "next_cursor" {
		t.Fatalf("cursor = %q", cursor)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Cid != "c_000000000000001" || items[0].Title != "Sample chat title" {
		t.Fatalf("item[0] = %+v", items[0])
	}
	if items[0].UpdatedAt != "2023-11-14T22:13" {
		t.Fatalf("UpdatedAt = %q", items[0].UpdatedAt)
	}
}

func TestDecodeListChats_EmptyBody(t *testing.T) {
	items, cursor, err := DecodeListChats([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 || cursor != "" {
		t.Fatalf("items=%d cursor=%q", len(items), cursor)
	}
}

func TestDecodeListChats_MalformedJSON(t *testing.T) {
	if _, _, err := DecodeListChats([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
