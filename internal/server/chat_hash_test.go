package server

import "testing"

func TestChatHistoryHashesStable(t *testing.T) {
	messages := []chatMessage{
		{Role: "User", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	first, err := chatHistoryHashes(messages)
	if err != nil {
		t.Fatal(err)
	}
	second, err := chatHistoryHashes(messages)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || len(second) != 2 || first[1].Hash != second[1].Hash {
		t.Fatalf("hashes are not stable: %#v %#v", first, second)
	}
	if !first[1].Completed || first[0].Completed {
		t.Fatalf("completed flags = %#v", first)
	}
}

func TestCompletedChatPrefixes(t *testing.T) {
	messages := []chatMessage{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "u2"},
		{Role: "assistant", Content: "a2"},
		{Role: "user", Content: "u3"},
	}
	prefixes, err := completedChatPrefixes(messages)
	if err != nil {
		t.Fatal(err)
	}
	if len(prefixes) != 2 {
		t.Fatalf("completed prefixes = %d, want 2", len(prefixes))
	}
	if prefixes[0].MessageCount != 2 || prefixes[1].MessageCount != 4 {
		t.Fatalf("message counts = %#v", prefixes)
	}
}

func TestFlattenChatMessages(t *testing.T) {
	got, err := flattenChatMessages([]chatMessage{
		{Role: "system", Content: "rules"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[System]\nrules\n\n[User]\nhello\n\n[Assistant]\nhi"
	if got != want {
		t.Fatalf("flatten = %q, want %q", got, want)
	}
}

func TestCanonicalChatMessageRejectsUnsupportedRole(t *testing.T) {
	_, _, err := canonicalChatMessage(chatMessage{Role: "tool", Content: "x"})
	if err == nil {
		t.Fatal("expected unsupported role error")
	}
}
