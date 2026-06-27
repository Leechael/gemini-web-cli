package server

import (
	"path/filepath"
	"testing"

	serverstate "github.com/Leechael/gemini-web-cli/internal/server/state"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

func TestPlanMappedChatVerifiedSendsOnlyLastUser(t *testing.T) {
	messages := []chatMessage{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "u2"},
	}
	prefixes, err := completedChatPrefixes(messages)
	if err != nil {
		t.Fatal(err)
	}
	store, err := serverstate.NewChatMapStore("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&serverstate.ChatMapEntry{
		RootHash:     prefixes[0].Hash,
		ChatId:       "c_1",
		Rid:          "r_1",
		Rcid:         "rc_1",
		Context:      "ctx_1",
		MessageCount: prefixes[0].MessageCount,
		Confidence:   serverstate.ChatMapConfidence_VERIFIED,
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := (&Server{chatMap: store}).planMappedChat(messages)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Prompt != "u2" {
		t.Fatalf("prompt = %q, want final user only", plan.Prompt)
	}
	if plan.Source != "mapped_verified" {
		t.Fatalf("source = %q, want mapped_verified", plan.Source)
	}
	if got := plan.Metadata[0]; got != "c_1" {
		t.Fatalf("metadata chat id = %q", got)
	}
}

func TestPlanMappedChatSyntheticFlattensSuffix(t *testing.T) {
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
	store, err := serverstate.NewChatMapStore("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&serverstate.ChatMapEntry{
		RootHash:     prefixes[0].Hash,
		ChatId:       "c_1",
		MessageCount: prefixes[0].MessageCount,
		Confidence:   serverstate.ChatMapConfidence_SYNTHETIC,
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := (&Server{chatMap: store}).planMappedChat(messages)
	if err != nil {
		t.Fatal(err)
	}
	want := "[User]\nu2\n\n[Assistant]\na2\n\n[User]\nu3"
	if plan.Prompt != want {
		t.Fatalf("prompt = %q, want %q", plan.Prompt, want)
	}
	if plan.Source != "mapped_synthetic" {
		t.Fatalf("source = %q, want mapped_synthetic", plan.Source)
	}
}

func TestSaveChatMappingWritesVerifiedAndSynthetic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat-map.pb")
	store, err := serverstate.NewChatMapStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{chatMap: store}
	request := []chatMessage{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "u2"},
	}
	out := &types.ModelOutput{Text: "a2", Metadata: []string{"c_1", "r_2", "rc_2", "", "", "", "", "", "", "ctx_2"}}
	if err := s.saveChatMapping(request, out); err != nil {
		t.Fatal(err)
	}

	completed := append(append([]chatMessage{}, request...), chatMessage{Role: "assistant", Content: "a2"})
	root, ok, err := completedChatRoot(completed)
	if err != nil || !ok {
		t.Fatalf("completed root ok=%v err=%v", ok, err)
	}
	entry, ok := store.Lookup(root.Hash)
	if !ok {
		t.Fatal("verified entry missing")
	}
	if entry.Confidence != serverstate.ChatMapConfidence_VERIFIED || entry.ChatId != "c_1" {
		t.Fatalf("verified entry = %#v", entry)
	}

	prefixes, err := completedChatPrefixes(request)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok = store.Lookup(prefixes[0].Hash)
	if !ok {
		t.Fatal("synthetic entry missing")
	}
	if entry.Confidence != serverstate.ChatMapConfidence_SYNTHETIC {
		t.Fatalf("synthetic confidence = %v", entry.Confidence)
	}
}
