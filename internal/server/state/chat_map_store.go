package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"google.golang.org/protobuf/proto"
)

const currentChatMapVersion = 1

type ChatMapStore struct {
	mu      sync.RWMutex
	path    string
	entries map[string]*ChatMapEntry
}

func NewChatMapStore(path string) (*ChatMapStore, error) {
	store := &ChatMapStore{
		path:    path,
		entries: make(map[string]*ChatMapEntry),
	}
	if path == "" {
		return store, nil
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ChatMapStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *ChatMapStore) Lookup(rootHash string) (*ChatMapEntry, bool) {
	if s == nil || rootHash == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[rootHash]
	if !ok {
		return nil, false
	}
	return proto.Clone(entry).(*ChatMapEntry), true
}

func (s *ChatMapStore) Upsert(entry *ChatMapEntry) error {
	if s == nil || entry == nil || entry.RootHash == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing := s.entries[entry.RootHash]; existing != nil {
		if existing.Confidence == ChatMapConfidence_VERIFIED && entry.Confidence == ChatMapConfidence_SYNTHETIC {
			return nil
		}
	}
	s.entries[entry.RootHash] = proto.Clone(entry).(*ChatMapEntry)
	return s.saveLocked()
}

func (s *ChatMapStore) UpsertMany(entries []*ChatMapEntry) error {
	if s == nil || len(entries) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	for _, entry := range entries {
		if entry == nil || entry.RootHash == "" {
			continue
		}
		if existing := s.entries[entry.RootHash]; existing != nil {
			if existing.Confidence == ChatMapConfidence_VERIFIED && entry.Confidence == ChatMapConfidence_SYNTHETIC {
				continue
			}
		}
		s.entries[entry.RootHash] = proto.Clone(entry).(*ChatMapEntry)
		changed = true
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
}

func (s *ChatMapStore) Entries() []*ChatMapEntry {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]*ChatMapEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, proto.Clone(entry).(*ChatMapEntry))
	}
	return entries
}

func (s *ChatMapStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read chat map: %w", err)
	}
	var db ChatMapDB
	if err := proto.Unmarshal(data, &db); err != nil {
		return fmt.Errorf("decode chat map: %w", err)
	}
	for _, entry := range db.Entries {
		if entry == nil || entry.RootHash == "" {
			continue
		}
		s.entries[entry.RootHash] = proto.Clone(entry).(*ChatMapEntry)
	}
	return nil
}

func (s *ChatMapStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create chat map dir: %w", err)
	}
	db := &ChatMapDB{Version: currentChatMapVersion}
	for _, entry := range s.entries {
		db.Entries = append(db.Entries, proto.Clone(entry).(*ChatMapEntry))
	}
	data, err := proto.Marshal(db)
	if err != nil {
		return fmt.Errorf("encode chat map: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write chat map temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace chat map: %w", err)
	}
	return nil
}
