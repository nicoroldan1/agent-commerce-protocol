package store

import "github.com/nicroldan/ans/shared/ace"

// TestEntry creates a minimal StoreEntry for testing purposes.
func TestEntry(name string) ace.StoreEntry {
	return ace.StoreEntry{
		Name:         name,
		HealthStatus: "healthy",
	}
}
