package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.2: Oceanic Pathfinding
// NavalRoutingSystem is a specialized pathfinder that traces vectors across deep water grids,
// avoiding shallows or seasonal ice caps calculated by the CalendarSystem.
// This calculates a path using engine.PathRequestQueue targeting the generated oceanic nav-mesh.

type pendingNavalReq struct {
	EntityID uint64
	Entity   ecs.Entity
	StartX   float32
	StartY   float32
	TargetX  float32
	TargetY  float32
}

type NavalRoutingSystem struct {
	mapGrid     *engine.MapGrid
	pathQueue   *engine.PathRequestQueue
	calendar    *engine.Calendar
	pendingReqs []pendingNavalReq
	filter      ecs.Filter
}

func NewNavalRoutingSystem(world *ecs.World, mapGrid *engine.MapGrid, pathQueue *engine.PathRequestQueue, calendar *engine.Calendar) *NavalRoutingSystem {
	shipID := ecs.ComponentID[components.ShipComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)
	idID := ecs.ComponentID[components.Identity](world)

	mask := ecs.All(shipID, posID, pathID, idID)

	return &NavalRoutingSystem{
		mapGrid:     mapGrid,
		pathQueue:   pathQueue,
		calendar:    calendar,
		pendingReqs: make([]pendingNavalReq, 0, 100),
		filter:      &mask,
	}
}

func (s *NavalRoutingSystem) Update(world *ecs.World) {
	pathID := ecs.ComponentID[components.Path](world)
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)

	// Step 1: Process ALL completed pathfinding results deterministically.
	// Iterating over a pre-allocated DOD slice ensures identical order across parallel seeds.
	for i := 0; i < len(s.pendingReqs); i++ {
		req := s.pendingReqs[i]
		if world.Alive(req.Entity) && world.Has(req.Entity, pathID) {
			path := (*components.Path)(world.Get(req.Entity, pathID))

			// Run exact deterministic pathing logic natively synchronously
			pathReq := engine.PathRequest{
				EntityID: req.EntityID,
				StartX:   req.StartX,
				StartY:   req.StartY,
				TargetX:  req.TargetX,
				TargetY:  req.TargetY,
				IsNaval:  true,
			}

			res := s.pathQueue.WorkerProcessSync(pathReq, s.mapGrid)

			if res.Success && len(res.Path) > 0 {
				path.Nodes = make([]components.Position, len(res.Path))
				for i, p := range res.Path {
					path.Nodes[i] = components.Position{X: p.X, Y: p.Y}
				}
				path.HasPath = true
			} else {
				path.HasPath = false
				// Prevent infinite CPU spin loop re-attempting an impossible path
				path.TargetX = 0
				path.TargetY = 0
			}
		}
	}

	// Clear flat array DOD style
	s.pendingReqs = s.pendingReqs[:0]

	// Step 2: Queue new pathfinding requests for Ships that need routing
	query := world.Query(s.filter)

	isWinter := false
	if s.calendar != nil {
		isWinter = s.calendar.IsWinter
	}

	for query.Next() {
		entity := query.Entity()
		ident := (*components.Identity)(query.Get(idID))
		entityID := ident.ID

		path := (*components.Path)(query.Get(pathID))

		// If ship has no path but needs one (Target != current Pos), request a route
		// In a real implementation, TargetX/Y would be set by a trade AI.
		// For Phase 17.2, we assume TargetX and TargetY are set on the Path component.
		if !path.HasPath && (path.TargetX != 0 || path.TargetY != 0) {

			// Avoid routing if frozen (Ice Caps logic)
			// (Assuming winter prevents naval routing in certain zones, simplistically blocked here)
			if isWinter {
				continue
			}

			pos := (*components.Position)(query.Get(posID))

			s.pendingReqs = append(s.pendingReqs, pendingNavalReq{
				EntityID: entityID,
				Entity:   entity,
				StartX:   pos.X,
				StartY:   pos.Y,
				TargetX:  path.TargetX,
				TargetY:  path.TargetY,
			})
		}
	}
}
