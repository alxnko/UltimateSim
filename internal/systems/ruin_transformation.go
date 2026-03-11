package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.2: The Ruin Transformation
// Transforms abandoned settlements into Ruins instead of destroying them outright.
// Iterates PopulationComponent, if count drops to 0, removes Population and Needs
// and adds RuinComponent. This prevents "Zombie Entity" processing.

type RuinTransformationSystem struct {
	toRuin []ecs.Entity // Collect entities to ruin outside the query loop
	filter ecs.Filter
}

func NewRuinTransformationSystem(world *ecs.World) *RuinTransformationSystem {
	popID := ecs.ComponentID[components.PopulationComponent](world)
	mask := ecs.All(popID)

	return &RuinTransformationSystem{
		toRuin: make([]ecs.Entity, 0, 100),
		filter: &mask,
	}
}

func (s *RuinTransformationSystem) Update(world *ecs.World) {
	popID := ecs.ComponentID[components.PopulationComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	idID := ecs.ComponentID[components.Identity](world)

	s.toRuin = s.toRuin[:0]

	// Find entities with Population == 0
	query := world.Query(s.filter)
	for query.Next() {
		pop := (*components.PopulationComponent)(query.Get(popID))
		if pop.Count == 0 {
			s.toRuin = append(s.toRuin, query.Entity())
		}
	}

	ruinID := ecs.ComponentID[components.RuinComponent](world)

	for _, e := range s.toRuin {
		var formerName string
		if world.Has(e, idID) {
			idComp := (*components.Identity)(world.Get(e, idID))
			formerName = idComp.Name
		}

		// Instead of despawning, we convert to ruin
		world.Remove(e, popID)
		if world.Has(e, needsID) {
			world.Remove(e, needsID)
		}

		world.Add(e, ruinID)
		ruin := (*components.RuinComponent)(world.Get(e, ruinID))
		ruin.Decay = 0
		ruin.FormerName = formerName
	}
}
