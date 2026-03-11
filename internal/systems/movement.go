package systems

import (
	"math"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 01.3: ECS Core Setup - MovementSystem
// Phase 04.4: Resolving Kinematics
// Implementing a deterministic, cache-friendly iteration system over Position, Velocity, and Path components.

// MovementSystem updates the Position of entities based on their Velocity and active Paths.
type MovementSystem struct {
	mapGrid *engine.MapGrid
	filter  ecs.Filter
}

// NewMovementSystem creates a new MovementSystem.
func NewMovementSystem(world *ecs.World, mapGrid *engine.MapGrid) *MovementSystem {
	// Enforce strict 'arche-go' filter usage to query specific components and prevent 'Zombie Entity' processing.
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// ecs.All returns an ecs.Mask which implements ecs.Filter
	// We only strictly require Position and Velocity. Path component is checked optionally inside the loop.
	mask := ecs.All(posID, velID)

	return &MovementSystem{
		mapGrid: mapGrid,
		filter:  &mask,
	}
}

// Update executes the system logic per tick.
func (s *MovementSystem) Update(world *ecs.World) {
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	pathID := ecs.ComponentID[components.Path](world)

	maxX := float32(s.mapGrid.Width - 1)
	maxY := float32(s.mapGrid.Height - 1)

	// Iterate over all entities matching the filter
	query := world.Query(s.filter)
	for query.Next() {
		// Access components via flat memory pointers (arche-go handles the layout)
		pos := (*components.Position)(query.Get(posID))
		vel := (*components.Velocity)(query.Get(velID))

		// Resolve Kinematics if there is an active path
		if query.Has(pathID) {
			path := (*components.Path)(query.Get(pathID))
			if path.HasPath && len(path.Nodes) > 0 {
				target := path.Nodes[0]

				dx := target.X - pos.X
				dy := target.Y - pos.Y
				distSq := dx*dx + dy*dy

				// Small epsilon squared to consider the node reached (e.g., within 0.1 units)
				if distSq < 0.01 {
					// Reached the node, pop it
					path.Nodes = path.Nodes[1:]
					if len(path.Nodes) == 0 {
						path.HasPath = false
						vel.X = 0
						vel.Y = 0
					} else {
						// We still have nodes. Update target for this frame.
						target = path.Nodes[0]
						dx = target.X - pos.X
						dy = target.Y - pos.Y
						distSq = dx*dx + dy*dy
					}
				}

				// If not reached (or just switched node), calculate velocity
				if len(path.Nodes) > 0 {
					// Move towards the node
					dist := float32(math.Sqrt(float64(distSq)))
					// Normalize and set a constant speed (e.g., 1.0 units per tick)
					// For simplicity, we just use a speed of 1.0, or cap at distance
					speed := float32(1.0)
					if dist < speed {
						speed = dist
					}
					vel.X = (dx / dist) * speed
					vel.Y = (dy / dist) * speed
				}
			} else if !path.HasPath {
				// Ensure velocity is 0 if no path is active, to prevent drifting
				vel.X = 0
				vel.Y = 0
			}
		}

		// Apply velocity to position
		pos.X += vel.X
		pos.Y += vel.Y

		// Process bounds verifications so values do not map outside arrays
		if pos.X < 0 {
			pos.X = 0
		} else if pos.X > maxX {
			pos.X = maxX
		}

		if pos.Y < 0 {
			pos.Y = 0
		} else if pos.Y > maxY {
			pos.Y = maxY
		}
	}
}
