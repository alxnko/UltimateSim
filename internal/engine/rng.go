package engine

import (
	"math/rand/v2"
	"sync"
)

// Phase 01.2: Deterministic Simulation Focus
// A global singleton RNG seed to handle all stochastic events securely and deterministically.
// We are using ChaCha8 with math/rand/v2 to maintain absolute determinism across all components.

var (
	rngInstance *rand.Rand
	mu          sync.Mutex
)

// InitializeRNG initializes the global random number generator with a given seed.
// This must be called at the start of the simulation to ensure determinism.
func InitializeRNG(seed [32]byte) {
	mu.Lock()
	defer mu.Unlock()

	src := rand.NewChaCha8(seed)
	rngInstance = rand.New(src)
}

// GetRandomInt returns a deterministic pseudo-random integer.
func GetRandomInt() int {
	mu.Lock()
	defer mu.Unlock()

	if rngInstance == nil {
		panic("RNG not initialized")
	}
	return rngInstance.Int()
}

// GetRandomFloat32 returns a deterministic pseudo-random float32 in [0.0, 1.0).
func GetRandomFloat32() float32 {
	mu.Lock()
	defer mu.Unlock()

	if rngInstance == nil {
		panic("RNG not initialized")
	}
	return rngInstance.Float32()
}

// GetRandomFloat64 returns a deterministic pseudo-random float64 in [0.0, 1.0).
func GetRandomFloat64() float64 {
	mu.Lock()
	defer mu.Unlock()

	if rngInstance == nil {
		panic("RNG not initialized")
	}
	return rngInstance.Float64()
}
