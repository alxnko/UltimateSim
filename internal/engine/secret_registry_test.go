package engine

import (
	"sync"
	"testing"
)

// Phase 07.1: Secret Registry (String Interning) Test

func TestSecretRegistry_RegisterAndRetrieve(t *testing.T) {
	registry := GetSecretRegistry()

	// Given a string
	text1 := "The King is dead"

	// When registering it
	id1 := registry.RegisterSecret(text1)

	// It should assign a valid ID (not 0)
	if id1 == 0 {
		t.Fatalf("Expected non-zero ID for secret, got %d", id1)
	}

	// And getting it back should return the same string
	retrievedText, exists := registry.GetSecret(id1)
	if !exists || retrievedText != text1 {
		t.Errorf("Expected to retrieve %q, got %q (exists: %v)", text1, retrievedText, exists)
	}

	// When registering the exact same string again
	id2 := registry.RegisterSecret(text1)

	// It should return the exact same ID, saving RAM
	if id1 != id2 {
		t.Errorf("Expected ID %d for same string, got %d", id1, id2)
	}

	// When getting a non-existent secret
	_, exists = registry.GetSecret(9999)
	if exists {
		t.Errorf("Expected to not find secret 9999")
	}
}

func TestSecretRegistry_Concurrency(t *testing.T) {
	registry := &SecretRegistry{
		secrets: make(map[uint32]string),
		reverse: make(map[string]uint32),
		nextID:  1,
	}

	// Run multiple goroutines trying to insert the same string and different strings
	var wg sync.WaitGroup
	numWorkers := 100
	wg.Add(numWorkers)

	commonText := "Winter is coming"

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			registry.RegisterSecret(commonText)
			// also register a unique text to ensure no deadlocks and map concurrent writes are safe
			registry.RegisterSecret("Unique text")
		}()
	}

	wg.Wait()

	// Common text should only be registered once
	id := registry.RegisterSecret(commonText)
	if id == 0 {
		t.Errorf("Expected valid ID for common text")
	}

	// Ensure there are no panics
}
