package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeReadChat_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeReadChat("c_000000000000001", 30)
	if rpcID != "hNvQHb" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got[0] != "c_000000000000001" || got[1] != float64(30) {
		t.Fatalf("payload = %#v", got)
	}
	if len(got) != 8 {
		t.Fatalf("payload len = %d", len(got))
	}
}

func TestDecodeReadChat_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "read_chat_basic.txt", "hNvQHb")
	turns, err := DecodeReadChat(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("turns = %d", len(turns))
	}
	if turns[0].UserPrompt != "Sample user prompt" || turns[0].AssistantResponse != "Sample assistant response." {
		t.Fatalf("turn[0] = %+v", turns[0])
	}
	if turns[0].Rid != "r_000000000000001" || turns[0].RCid != "rcid_000000000000001" {
		t.Fatalf("metadata = %+v", turns[0])
	}
	if turns[0].CreatedAtUnix != 1700000000 {
		t.Fatalf("CreatedAtUnix = %d", turns[0].CreatedAtUnix)
	}
}

func TestDecodeReadChat_EmptyBody(t *testing.T) {
	turns, err := DecodeReadChat(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 0 {
		t.Fatalf("turns = %d", len(turns))
	}
}

func TestDecodeReadChat_MalformedJSON(t *testing.T) {
	if _, err := DecodeReadChat([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeReadChatRaw_EmptyBody(t *testing.T) {
	turns, err := DecodeReadChatRaw(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 0 {
		t.Fatalf("turns = %d", len(turns))
	}
}

func TestDecodeReadChatRaw_FallbackToOuterArray(t *testing.T) {
	body := []byte(`[{"not":"turn list"}]`)
	turns, err := DecodeReadChatRaw(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || string(turns[0]) != `{"not":"turn list"}` {
		t.Fatalf("turns = %v", turns)
	}
}

func TestDecodeReadChatRaw_MalformedJSON(t *testing.T) {
	if _, err := DecodeReadChatRaw([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeReadChat_PureMediaTurn(t *testing.T) {
	item := []any{nil, nil, nil, []any{nil, 1, "file.png", "https://lh3.googleusercontent.com/sample"}}
	mediaData := make([]any, 8)
	mediaData[7] = []any{[]any{[]any{item}}}
	candidate := make([]any, 13)
	candidate[0] = "rcid_000000000000001"
	candidate[1] = []any{"http://googleusercontent.com/image_generation_content/0"}
	candidate[12] = mediaData
	turn := []any{[]any{"c_000000000000001", "r_000000000000001"}, nil, []any{[]any{""}}, []any{[]any{candidate}}}
	body, _ := json.Marshal([]any{[]any{turn}})
	turns, err := DecodeReadChat(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || len(turns[0].Images) != 1 || turns[0].AssistantResponse != "" {
		t.Fatalf("turns = %+v", turns)
	}
}

func TestDecodeReadChat_CardURLLines(t *testing.T) {
	candidate := []any{"rcid_000000000000001", []any{"Sample assistant response.\nhttp://googleusercontent.com/card_content/0"}}
	turn := []any{[]any{"c_000000000000001", "r_000000000000001"}, nil, []any{[]any{"Sample user prompt"}}, []any{[]any{candidate}}}
	body, _ := json.Marshal([]any{[]any{turn}})
	turns, err := DecodeReadChat(body)
	if err != nil {
		t.Fatal(err)
	}
	if turns[0].AssistantResponse != "Sample assistant response." {
		t.Fatalf("AssistantResponse = %q", turns[0].AssistantResponse)
	}
}
