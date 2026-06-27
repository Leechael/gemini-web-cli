package state

import (
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestChatMapStoreLoadSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat-map.pb")
	store, err := NewChatMapStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&ChatMapEntry{
		RootHash:     "root",
		ChatId:       "c_1",
		Rid:          "r_1",
		Rcid:         "rc_1",
		Context:      "ctx",
		MessageCount: 2,
		Confidence:   ChatMapConfidence_VERIFIED,
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewChatMapStore(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := reloaded.Lookup("root")
	if !ok {
		t.Fatal("entry missing after reload")
	}
	if entry.ChatId != "c_1" || entry.Confidence != ChatMapConfidence_VERIFIED {
		t.Fatalf("entry = %#v", entry)
	}
}

func TestChatMapStoreVersionCheck(t *testing.T) {
	t.Run("current version loads", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "chat-map.pb")
		db := &ChatMapDB{
			Version: currentChatMapVersion,
			Entries: []*ChatMapEntry{{RootHash: "root", ChatId: "c_1"}},
		}
		data, err := proto.Marshal(db)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		store, err := NewChatMapStore(path)
		if err != nil {
			t.Fatalf("expected load to succeed, got %v", err)
		}
		if _, ok := store.Lookup("root"); !ok {
			t.Fatal("entry missing after load")
		}
	})

	t.Run("incompatible version rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "chat-map.pb")
		db := &ChatMapDB{
			Version: currentChatMapVersion + 1,
			Entries: []*ChatMapEntry{{RootHash: "root", ChatId: "c_1"}},
		}
		data, err := proto.Marshal(db)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewChatMapStore(path); err == nil {
			t.Fatal("expected error for incompatible version, got nil")
		}
	})
}

func TestChatMapStoreDoesNotDowngradeVerified(t *testing.T) {
	store, err := NewChatMapStore("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&ChatMapEntry{RootHash: "root", ChatId: "verified", Confidence: ChatMapConfidence_VERIFIED}); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&ChatMapEntry{RootHash: "root", ChatId: "synthetic", Confidence: ChatMapConfidence_SYNTHETIC}); err != nil {
		t.Fatal(err)
	}
	entry, ok := store.Lookup("root")
	if !ok {
		t.Fatal("entry missing")
	}
	if entry.ChatId != "verified" || entry.Confidence != ChatMapConfidence_VERIFIED {
		t.Fatalf("entry downgraded: %#v", entry)
	}
}
