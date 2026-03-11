package engine

import (
	"sync"
)

// Phase 06.3: The Sparse Hook Graph implementation
// The Sparse Solution: map[EntityID]map[TargetID]int

type SparseHookGraph struct {
	mu    sync.RWMutex
	graph map[uint64]map[uint64]int
}

// NewSparseHookGraph initializes the sparse hook graph.
func NewSparseHookGraph() *SparseHookGraph {
	return &SparseHookGraph{
		graph: make(map[uint64]map[uint64]int),
	}
}

// AddHook appends hook points dynamically.
func (s *SparseHookGraph) AddHook(entityA, entityB uint64, points int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.graph[entityA] == nil {
		s.graph[entityA] = make(map[uint64]int)
	}

	s.graph[entityA][entityB] += points
}

// SpendHook decrements hook points.
func (s *SparseHookGraph) SpendHook(entityA, entityB uint64, points int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.graph[entityA] == nil {
		return
	}

	s.graph[entityA][entityB] -= points
}

// GetHook retrieves the current hook points.
func (s *SparseHookGraph) GetHook(entityA, entityB uint64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if targets, exists := s.graph[entityA]; exists {
		return targets[entityB]
	}

	return 0
}
