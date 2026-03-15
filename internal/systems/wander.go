package systems

import (
	"math"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 04.3: Trait & Need Driven Targeting - WanderSystem Evaluator

type WanderSystem struct {
	mapGrid     *engine.MapGrid
	pathQueue   *engine.PathRequestQueue
	filter      ecs.Filter
	pendingReqs map[uint64]ecs.Entity
	tickCounter uint64
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *WanderSystem) IsExpensive() bool {
	return true
}

// NewWanderSystem creates a new WanderSystem.
func NewWanderSystem(world *ecs.World, mapGrid *engine.MapGrid, pathQueue *engine.PathRequestQueue) *WanderSystem {
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)
	needsID := ecs.ComponentID[components.Needs](world)
	pathID := ecs.ComponentID[components.Path](world)

	possessedID := ecs.ComponentID[components.Possessed](world)

	// Phase 11.2: Override the standard WanderSystem AI state-processor for the Possessed target
	// We skip entities that are Possessed so input cleanly controls movement
	mask := ecs.All(posID, idID, needsID, pathID).Without(possessedID)

	return &WanderSystem{
		mapGrid:     mapGrid,
		pathQueue:   pathQueue,
		filter:      &mask,
		pendingReqs: make(map[uint64]ecs.Entity),
		tickCounter: 0,
	}
}

// Update executes the system logic per tick.
func (s *WanderSystem) Update(world *ecs.World) {
	s.tickCounter++
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)
	needsID := ecs.ComponentID[components.Needs](world)
	pathID := ecs.ComponentID[components.Path](world)

	// 1. Drain incoming path results from the asynchronous queue
DrainLoop:
	for {
		select {
		case res := <-s.pathQueue.GetResultsChannel():
			// Re-fetch entity from pending map
			if entity, exists := s.pendingReqs[res.EntityID]; exists {
				// Ensure entity still exists in world
				if world.Alive(entity) {
					path := (*components.Path)(world.Get(entity, pathID))
					if res.Success && len(res.Path) > 0 {
						// We successfully found a path
						path.Nodes = make([]components.Position, len(res.Path))
						for i, node := range res.Path {
							path.Nodes[i] = components.Position{X: node.X, Y: node.Y}
						}
						// Next target logic would go here in Phase 4.4 Resolving Kinematics
					} else {
						// Path failed, allow retry
						path.HasPath = false
					}
				}
				delete(s.pendingReqs, res.EntityID)
			}
		default:
			// No pending results
			break DrainLoop
		}
	}

	// 2. Iterate entities to generate new path requests
	query := world.Query(s.filter)
	npcIndex := 0
	for query.Next() {
		npcIndex++
		path := (*components.Path)(query.Get(pathID))

		// Only process entities that do not currently have a path pending or active
		if path.HasPath {
			continue
		}

		// Phase 31.1: Throttle AI evaluations
		// Only evaluate a fraction of entities each tick to maintain 60 TPS during heavy simulation.
		if (s.tickCounter + uint64(npcIndex)) % 30 != 0 {
			continue
		}

		needs := (*components.Needs)(query.Get(needsID))
		id := (*components.Identity)(query.Get(idID))
		pos := (*components.Position)(query.Get(posID))

		// Is Food dominant missing need? Let's say < 50 threshold for hunger.
		if needs.Food < 50.0 {
			// Find nearest ResourceDepot(Food) using flat-memory sequential iteration.
			var bestX, bestY int
			var bestScore float32 = math.MaxFloat32
			foundTarget := false

			// Optimized with FoodCache and sampling (check every 8th food source)
			step := 8
			for i := 0; i < len(s.mapGrid.FoodCache); i += step {
				idx := s.mapGrid.FoodCache[i]
				x := idx % s.mapGrid.Width
				y := idx / s.mapGrid.Width

				// Base euclidean distance
				dx := float32(x) - pos.X
				dy := float32(y) - pos.Y
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

				// Apply Traits
				variance := float32((x*17 + y*31) % 15)

				if (id.BaseTraits & components.TraitCautious) != 0 {
					dist += variance
				} else if (id.BaseTraits & components.TraitRiskTaker) != 0 {
					dist -= variance
				}

				if dist < bestScore {
					bestScore = dist
					bestX = x
					bestY = y
					foundTarget = true
				}
			}

			if foundTarget {
				// We found food, dispatch pathfinding request
				req := engine.PathRequest{
					EntityID: id.ID,
					StartX:   pos.X,
					StartY:   pos.Y,
					TargetX:  float32(bestX),
					TargetY:  float32(bestY),
				}
				s.pathQueue.Enqueue(req)
				s.pendingReqs[id.ID] = query.Entity()

				path.HasPath = true
				path.TargetX = float32(bestX)
				path.TargetY = float32(bestY)
			}
		}
	}
}
