package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/nicroldan/ans/shared/ace"
)

// MemoryStore is a thread-safe in-memory store for StoreEntry records.
type MemoryStore struct {
	mu          sync.RWMutex
	entries     map[string]ace.StoreEntry
	tokenHashes map[string]string // SHA-256 token hash -> store ID
	reports     []ace.StoreReport
}

// New creates a new MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{
		entries:     make(map[string]ace.StoreEntry),
		tokenHashes: make(map[string]string),
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

// StoreTokenHash associates a SHA-256 token hash with a store ID.
func (m *MemoryStore) StoreTokenHash(storeID, hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokenHashes[hash] = storeID
}

// GetStoreIDByTokenHash returns the store ID associated with the given token hash.
func (m *MemoryStore) GetStoreIDByTokenHash(hash string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.tokenHashes[hash]
	return id, ok
}

// DeleteStore removes the store entry and its associated token hash. Returns false if not found.
func (m *MemoryStore) DeleteStore(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.entries[id]; !ok {
		return false
	}
	// Remove any token hash pointing to this store.
	for hash, sid := range m.tokenHashes {
		if sid == id {
			delete(m.tokenHashes, hash)
			break
		}
	}
	delete(m.entries, id)
	return true
}

// AddReport stores a report against a store.
func (m *MemoryStore) AddReport(report ace.StoreReport) {
	m.mu.Lock()
	defer m.mu.Unlock()

	report.ID = generateID()
	m.reports = append(m.reports, report)
}

// ResolveToken hashes the raw token with SHA-256, looks up the associated store ID,
// and returns the store ID, store name, and whether the token was valid.
func (m *MemoryStore) ResolveToken(rawToken string) (storeID, storeName string, ok bool) {
	sum := sha256.Sum256([]byte(rawToken))
	hash := hex.EncodeToString(sum[:])

	m.mu.RLock()
	defer m.mu.RUnlock()

	sid, found := m.tokenHashes[hash]
	if !found {
		return "", "", false
	}
	entry, exists := m.entries[sid]
	if !exists {
		return "", "", false
	}
	return entry.ID, entry.Name, true
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return true
		}
	}
	return false
}
