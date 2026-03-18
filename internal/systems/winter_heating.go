package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 31.5: The Winter Heating Engine (Resource Depletion Crisis)
// Connects Geography (Winter) -> Economy (Wood Storage) -> Governance (Loyalty) -> Biology (Disease)
// Villages rapidly consume Wood during Winter. If they freeze, Loyalty plummets, causing
// systemic Tax Evasion (Phase 42), and hypothermia instantiates DiseaseEntities (Phase 10.3).

type WinterHeatingSystem struct {
	calendar *engine.Calendar

	// Component IDs
	villageID  ecs.ID
	posID      ecs.ID
	popID      ecs.ID
	storageID  ecs.ID
	loyaltyID  ecs.ID
	diseaseID  ecs.ID
}

// NewWinterHeatingSystem creates a new WinterHeatingSystem.
func NewWinterHeatingSystem(world *ecs.World, calendar *engine.Calendar) *WinterHeatingSystem {
	return &WinterHeatingSystem{
		calendar:  calendar,
		villageID: ecs.ComponentID[components.Village](world),
		posID:     ecs.ComponentID[components.Position](world),
		popID:     ecs.ComponentID[components.PopulationComponent](world),
		storageID: ecs.ComponentID[components.StorageComponent](world),
		loyaltyID: ecs.ComponentID[components.LoyaltyComponent](world),
		diseaseID: ecs.ComponentID[components.DiseaseEntity](world),
	}
}

// Update executes the system logic per tick.
func (s *WinterHeatingSystem) Update(world *ecs.World) {
	// If it is not Winter, or no calendar is attached, do not run the loop
	if s.calendar == nil || !s.calendar.IsWinter {
		return
	}

	filter := ecs.All(s.villageID, s.posID, s.popID, s.storageID, s.loyaltyID)
	query := world.Query(filter)

	// Local cache for structurally spawning DiseaseEntities after the iterator closes
	type freezeData struct {
		X float32
		Y float32
	}
	var freezingVillages []freezeData

	for query.Next() {
		pop := (*components.PopulationComponent)(query.Get(s.popID))
		storage := (*components.StorageComponent)(query.Get(s.storageID))
		loyalty := (*components.LoyaltyComponent)(query.Get(s.loyaltyID))
		pos := (*components.Position)(query.Get(s.posID))

		// 1 Wood unit per 10 citizens per tick (abstracted heating fuel)
		woodRequired := uint32(float32(pop.Count) * 0.1)

		if storage.Wood >= woodRequired {
			// Successfully heated
			storage.Wood -= woodRequired
		} else {
			// Freezing Crisis!
			storage.Wood = 0

			// 1. Civil Unrest: Loyalty drops rapidly due to freezing conditions
			if loyalty.Value >= 5 {
				loyalty.Value -= 5
			} else {
				loyalty.Value = 0
			}

			// 2. Biological Entrophy: 1% chance per freezing tick to spawn Hypothermia (Plague)
			if engine.GetRandomInt()%100 == 0 {
				freezingVillages = append(freezingVillages, freezeData{
					X: pos.X,
					Y: pos.Y,
				})
			}
		}
	}

	// Safely modify the ECS structure outside the query loop
	for i := 0; i < len(freezingVillages); i++ {
		e := world.NewEntity()
		world.Add(e, s.posID, s.diseaseID)

		dPos := (*components.Position)(world.Get(e, s.posID))
		dPos.X = freezingVillages[i].X
		dPos.Y = freezingVillages[i].Y

		dComp := (*components.DiseaseEntity)(world.Get(e, s.diseaseID))
		// Generate an ID deterministically using rng
		dComp.ID = uint32(engine.GetRandomInt())
		dComp.Lethality = 5 // Low lethality (Hypothermia isn't universally fatal instantly, but weakens the populace)
	}
}
