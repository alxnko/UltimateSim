package engine

import (
	"sync"
	"testing"
)

// Phase 06.3: Sparse Hook Graph Test

func TestSparseHookGraph(t *testing.T) {
	graph := NewSparseHookGraph()

	// Initial check
	if graph.GetHook(1, 2) != 0 {
		t.Errorf("Expected hook to be 0, got %d", graph.GetHook(1, 2))
	}

	// Add hooks
	graph.AddHook(1, 2, 10)
	if graph.GetHook(1, 2) != 10 {
		t.Errorf("Expected hook to be 10, got %d", graph.GetHook(1, 2))
	}

	graph.AddHook(1, 2, 5)
	if graph.GetHook(1, 2) != 15 {
		t.Errorf("Expected hook to be 15, got %d", graph.GetHook(1, 2))
	}

	// Spend hooks
	graph.SpendHook(1, 2, 8)
	if graph.GetHook(1, 2) != 7 {
		t.Errorf("Expected hook to be 7, got %d", graph.GetHook(1, 2))
	}

	// Spend hooks when none exist
	graph.SpendHook(2, 1, 5)
	if graph.GetHook(2, 1) != 0 {
		t.Errorf("Expected hook to be 0, got %d", graph.GetHook(2, 1))
	}
}

func TestSparseHookGraphThreadSafety(t *testing.T) {
	graph := NewSparseHookGraph()
	var wg sync.WaitGroup

	numGoroutines := 100
	incrementsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				graph.AddHook(1, 2, 1)
			}
		}(uint64(i))
	}

	wg.Wait()

	expected := numGoroutines * incrementsPerGoroutine
	if graph.GetHook(1, 2) != expected {
		t.Errorf("Expected %d hooks, got %d", expected, graph.GetHook(1, 2))
	}
}
