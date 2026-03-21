package store

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/nicroldan/ans/shared/ace"
)

// MemoryStore is a thread-safe in-memory store for StoreEntry records.
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]ace.StoreEntry
}

// New creates a new MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]ace.StoreEntry),
	}
}

// generateID creates a unique store ID with "str_" prefix.
func generateID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "str_" + hex.EncodeToString(b)
}

// Create adds a new StoreEntry and returns it with a generated ID.
func (m *MemoryStore) Create(entry ace.StoreEntry) ace.StoreEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry.ID = generateID()
	m.entries[entry.ID] = entry
	return entry
}

// GetByID returns a StoreEntry by its ID. Returns false if not found.
func (m *MemoryStore) GetByID(id string) (ace.StoreEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[id]
	return entry, ok
}

// ListFilters defines the filtering/pagination options for List.
type ListFilters struct {
	Query    string
	Category string
	Country  string
	Currency string
	Offset   int
	Limit    int
}

// List returns store entries matching the given filters with pagination.
func (m *MemoryStore) List(f ListFilters) ([]ace.StoreEntry, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []ace.StoreEntry
	for _, e := range m.entries {
		if f.Query != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(f.Query)) {
			continue
		}
		if f.Category != "" && !containsStr(e.Categories, f.Category) {
			continue
		}
		if f.Country != "" && !strings.EqualFold(e.Country, f.Country) {
			continue
		}
		if f.Currency != "" && !containsStr(e.Currencies, f.Currency) {
			continue
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)

	// Apply pagination.
	if f.Offset >= len(filtered) {
		return []ace.StoreEntry{}, total
	}
	filtered = filtered[f.Offset:]
	if f.Limit > 0 && f.Limit < len(filtered) {
		filtered = filtered[:f.Limit]
	}

	return filtered, total
}

// Update replaces the StoreEntry for the given ID. Returns false if not found.
func (m *MemoryStore) Update(entry ace.StoreEntry) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.entries[entry.ID]; !ok {
		return false
	}
	m.entries[entry.ID] = entry
	return true
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return true
		}
	}
	return false
}
