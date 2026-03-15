package systems

import (
	"math"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 01.3: ECS Core Setup - MovementSystem
// Phase 04.4: Resolving Kinematics
// Phase 09.3: Infrastructure Wear System (Desire Paths)
// Implementing a deterministic, cache-friendly iteration system over Position, Velocity, and Path components.

// MovementSystem updates the Position of entities based on their Velocity and active Paths.
type MovementSystem struct {
	mapGrid  *engine.MapGrid
	filter   ecs.Filter
	calendar *engine.Calendar
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *MovementSystem) IsExpensive() bool {
	return true
}

// NewMovementSystem creates a new MovementSystem.
func NewMovementSystem(world *ecs.World, mapGrid *engine.MapGrid, calendar *engine.Calendar) *MovementSystem {
	// Enforce strict 'arche-go' filter usage to query specific components and prevent 'Zombie Entity' processing.
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	mask := ecs.All(posID, velID)

	return &MovementSystem{
		mapGrid:  mapGrid,
		filter:   &mask,
		calendar: calendar,
	}
}

// Update executes the system logic per tick.
func (s *MovementSystem) Update(world *ecs.World) {
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	pathID := ecs.ComponentID[components.Path](world)

	maxX := float32(s.mapGrid.Width - 1)
	maxY := float32(s.mapGrid.Height - 1)

	// Phase 11.2: Check if possessed (to skip AI pathing)
	possessedID := ecs.ComponentID[components.Possessed](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)

	// Iterate over all entities matching the filter
	query := world.Query(s.filter)
	for query.Next() {
		// Access components via flat memory pointers (arche-go handles the layout)
		pos := (*components.Position)(query.Get(posID))
		vel := (*components.Velocity)(query.Get(velID))

		// Skip pathing logic if possessed by user
		isPossessed := query.Has(possessedID)

		// Phase 09.3: Infrastructure Wear System (Desire Paths)
		// Calculate movement cost dynamically based on the current tile's biome and foot traffic.
		currentX, currentY := int(pos.X), int(pos.Y)

		// Ensure currentX and currentY are within map bounds
		if currentX < 0 { currentX = 0 } else if currentX >= s.mapGrid.Width { currentX = s.mapGrid.Width - 1 }
		if currentY < 0 { currentY = 0 } else if currentY >= s.mapGrid.Height { currentY = s.mapGrid.Height - 1 }

		tileIndex := currentY*s.mapGrid.Width + currentX
		tile := s.mapGrid.Tiles[tileIndex]
		state := s.mapGrid.TileStates[tileIndex]

		isWinter := false
		if s.calendar != nil {
			isWinter = s.calendar.IsWinter
		}

		movementCost := engine.GetEffectiveMovementCost(tile.BiomeID, state.FootTraffic, isWinter)

		// Resolve Kinematics if there is an active path and NOT possessed
		if !isPossessed && query.Has(pathID) {
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

					// Phase 09.3: Adjust speed inversely proportional to movement cost.
					// Base speed is 1.0 units per tick.
					speed := 1.0 / movementCost

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

		// Phase 19.4: Advanced Biology (Vitals integration)
		if query.Has(vitalsID) {
			vitals := (*components.VitalsComponent)(query.Get(vitalsID))

			if vitals.Consciousness <= 0 {
				vel.X = 0
				vel.Y = 0
			} else {
				// Stamina drain if moving
				if vel.X != 0 || vel.Y != 0 {
					vitals.Stamina -= 0.1
					if vitals.Stamina < 0 {
						vitals.Stamina = 0
					}
				}

				// Apply movement penalty if low stamina
				if vitals.Stamina < 10.0 {
					vel.X *= 0.5
					vel.Y *= 0.5
				}
			}
		}

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

		// Phase 09.3: Infrastructure Wear System (Desire Paths)
		// Detect tile transitions by comparing integer grids.
		newX, newY := int(pos.X), int(pos.Y)
		// Safe bounding (pos should be bounded above, but safeguard array indices)
		if newX < 0 { newX = 0 } else if newX >= s.mapGrid.Width { newX = s.mapGrid.Width - 1 }
		if newY < 0 { newY = 0 } else if newY >= s.mapGrid.Height { newY = s.mapGrid.Height - 1 }

		if newX != currentX || newY != currentY {
			// Entity transitioned to a new tile. Increment FootTraffic of the new tile.
			newTileIndex := newY*s.mapGrid.Width + newX
			s.mapGrid.TileStates[newTileIndex].FootTraffic++
		}
	}
}
