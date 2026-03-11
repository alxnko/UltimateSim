package engine

import (
	"sync"
	"testing"
)

// Phase 04.2: Async Path Queue Pool
// E2E Test verifying goroutine load distribution without blocking the ECS Loop.

func TestPathQueue_E2E(t *testing.T) {
	// Initialize the queue with buffer size 100, 4 workers
	pq := NewPathRequestQueue(100, 4)
	pq.StartWorkers()

	// WaitGroup to sync the consumer test routine
	var wg sync.WaitGroup
	wg.Add(1)

	totalRequests := 100
	resultsReceived := 0

	// Launch consumer goroutine to monitor results channel
	go func() {
		defer wg.Done()
		for result := range pq.GetResultsChannel() {
			if !result.Success || len(result.Path) == 0 {
				t.Errorf("Expected successful path generation, got fail for Entity %d", result.EntityID)
			}
			resultsReceived++
			if resultsReceived == totalRequests {
				return
			}
		}
	}()

	// Produce 100 deterministic path requests
	for i := 0; i < totalRequests; i++ {
		pq.Enqueue(PathRequest{
			EntityID: uint64(i + 1),
			StartX:   float32(i * 10),
			StartY:   float32(i * 10),
			TargetX:  float32((i + 1) * 10),
			TargetY:  float32((i + 1) * 10),
		})
	}

	// Wait for consumer to process all exactly
	wg.Wait()
	pq.Close()

	if resultsReceived != totalRequests {
		t.Fatalf("Expected exactly %d results, got %d", totalRequests, resultsReceived)
	}
}

func TestPathQueue_Deterministic(t *testing.T) {
	// Start two separate isolated queues to confirm identical worker processing
	pq1 := NewPathRequestQueue(10, 2)
	pq2 := NewPathRequestQueue(10, 2)

	pq1.StartWorkers()
	pq2.StartWorkers()

	req := PathRequest{
		EntityID: 999,
		StartX:   0,
		StartY:   0,
		TargetX:  100,
		TargetY:  100,
	}

	pq1.Enqueue(req)
	pq2.Enqueue(req)

	result1 := <-pq1.GetResultsChannel()
	result2 := <-pq2.GetResultsChannel()

	pq1.Close()
	pq2.Close()

	if result1.EntityID != result2.EntityID || result1.Success != result2.Success {
		t.Fatalf("Results metadata mismatch: %+v != %+v", result1, result2)
	}

	if len(result1.Path) != len(result2.Path) {
		t.Fatalf("Path length mismatch: %d != %d", len(result1.Path), len(result2.Path))
	}

	for i := range result1.Path {
		if result1.Path[i].X != result2.Path[i].X || result1.Path[i].Y != result2.Path[i].Y {
			t.Errorf("Path node mismatch at %d: %v != %v", i, result1.Path[i], result2.Path[i])
		}
	}
}
