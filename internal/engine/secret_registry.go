package engine

import (
	"sync"
)

// Phase 07.1: Secret Registry (String Interning)

// SecretRegistry is a global map to store strings precisely once to avoid RAM fragmentation.
type SecretRegistry struct {
	mu        sync.RWMutex
	secrets   map[uint32]string
	reverse   map[string]uint32
	nextID    uint32
}

var (
	GlobalSecretRegistry *SecretRegistry
	registryOnce         sync.Once
)

// GetSecretRegistry returns the singleton instance of the SecretRegistry.
func GetSecretRegistry() *SecretRegistry {
	registryOnce.Do(func() {
		GlobalSecretRegistry = &SecretRegistry{
			secrets: make(map[uint32]string),
			reverse: make(map[string]uint32),
			nextID:  1, // ID 0 is reserved for invalid/empty
		}
	})
	return GlobalSecretRegistry
}

// RegisterSecret adds a new string to the registry if it doesn't exist.
// Returns the unique uint32 ID for the string.
func (sr *SecretRegistry) RegisterSecret(text string) uint32 {
	sr.mu.RLock()
	id, exists := sr.reverse[text]
	sr.mu.RUnlock()

	if exists {
		return id
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Double-check after acquiring write lock
	if id, exists := sr.reverse[text]; exists {
		return id
	}

	id = sr.nextID
	sr.secrets[id] = text
	sr.reverse[text] = id
	sr.nextID++

	return id
}

// GetSecret retrieves a string by its ID.
// Returns the string and a boolean indicating if it was found.
func (sr *SecretRegistry) GetSecret(id uint32) (string, bool) {
	sr.mu.RLock()
	text, exists := sr.secrets[id]
	sr.mu.RUnlock()
	return text, exists
}
