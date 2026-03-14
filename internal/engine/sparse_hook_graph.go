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

// GetAllHooks retrieves all outgoing hook values mapped to target entities.
// Phase 25.1: Succession Engine
func (s *SparseHookGraph) GetAllHooks(entityA uint64) map[uint64]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[uint64]int)
	if targets, exists := s.graph[entityA]; exists {
		for target, points := range targets {
			result[target] = points
		}
	}
	return result
}

// GetAllIncomingHooks retrieves all incoming hooks (entities that have a hook on entityA) and their hook values.
// Phase 25.1: Succession Engine
func (s *SparseHookGraph) GetAllIncomingHooks(entityA uint64) map[uint64]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[uint64]int)
	for entity, targets := range s.graph {
		if points, exists := targets[entityA]; exists {
			result[entity] = points
		}
	}
	return result
}

// RemoveAllHooks deletes all incoming and outgoing hooks for an entity.
// Phase 25.1: Succession Engine
func (s *SparseHookGraph) RemoveAllHooks(entityA uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove outgoing hooks
	delete(s.graph, entityA)

	// Remove incoming hooks
	for _, targets := range s.graph {
		delete(targets, entityA)
	}
}
