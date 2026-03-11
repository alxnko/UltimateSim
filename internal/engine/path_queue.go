package engine

import (
	"math"
)

// Phase 04.2: Async Path Queue Pool
// Structs are designed to maintain minimal memory overhead and fast sequential iteration.

// Vec2 represents a 2D float32 vector, used for path coordinates.
// Defined here to avoid cyclic dependencies and keep engine logic contained.
type Vec2 struct {
	X float32
	Y float32
}

// PathRequest represents a query from an entity (WanderSystem) wanting to travel from Start to Target.
type PathRequest struct {
	EntityID uint64
	StartX   float32
	StartY   float32
	TargetX  float32
	TargetY  float32
}

// PathResult contains the calculated path nodes returned from the worker pool.
type PathResult struct {
	EntityID uint64
	Path     []Vec2
	Success  bool
}

// PathRequestQueue manages the goroutine worker pool for asynchronous HPA* math execution.
type PathRequestQueue struct {
	requests chan PathRequest
	results  chan PathResult
	workers  int
}

// NewPathRequestQueue initializes the channels and structure, taking a buffer size for channels and worker count.
func NewPathRequestQueue(bufferSize int, workers int) *PathRequestQueue {
	return &PathRequestQueue{
		requests: make(chan PathRequest, bufferSize),
		results:  make(chan PathResult, bufferSize),
		workers:  workers,
	}
}

// StartWorkers launches the dedicated persistent Goroutines that will process incoming requests.
func (pq *PathRequestQueue) StartWorkers() {
	for i := 0; i < pq.workers; i++ {
		go pq.workerProcess()
	}
}

// Enqueue sends a PathRequest into the worker pool. Non-blocking up to channel buffer limit.
func (pq *PathRequestQueue) Enqueue(req PathRequest) {
	pq.requests <- req
}

// GetResultsChannel returns the channel to read completed PathResults from.
func (pq *PathRequestQueue) GetResultsChannel() <-chan PathResult {
	return pq.results
}

// workerProcess is the actual loop run by each persistent goroutine.
func (pq *PathRequestQueue) workerProcess() {
	for req := range pq.requests {
		// Mock calculation for HPA* to ensure determinism and baseline logic.
		// In a real implementation, this would query pkg/math/hpa/AbstractGrid.

		// For deterministic check, we generate a direct line with fixed steps.
		// Distance formula
		dx := req.TargetX - req.StartX
		dy := req.TargetY - req.StartY
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

		var path []Vec2
		success := false

		if dist > 0 {
			success = true
			steps := int(math.Ceil(float64(dist)))

			// Deterministically allocate the slice size
			path = make([]Vec2, 0, steps+1)

			// Add steps along the vector
			stepX := dx / float32(steps)
			stepY := dy / float32(steps)

			for i := 0; i <= steps; i++ {
				path = append(path, Vec2{
					X: req.StartX + stepX*float32(i),
					Y: req.StartY + stepY*float32(i),
				})
			}

			// Ensure exactly hitting target at the end (preventing float drift)
			path[len(path)-1] = Vec2{X: req.TargetX, Y: req.TargetY}
		}

		// Pass the constructed path back
		pq.results <- PathResult{
			EntityID: req.EntityID,
			Path:     path,
			Success:  success,
		}
	}
}

// Close gracefully shuts down the queue by closing the requests channel,
// which will naturally terminate the worker loops.
func (pq *PathRequestQueue) Close() {
	close(pq.requests)
}
