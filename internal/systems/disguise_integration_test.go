package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestDisguiseSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	_ = ecs.ComponentID[components.Position](&world)
	_ = ecs.ComponentID[components.Affiliation](&world)
	_ = ecs.ComponentID[components.JurisdictionComponent](&world)
	_ = ecs.ComponentID[components.Memory](&world)
	_ = ecs.ComponentID[components.Identity](&world)
	_ = ecs.ComponentID[components.Path](&world)
	_ = ecs.ComponentID[components.CrimeMarker](&world)
	_ = ecs.ComponentID[components.StorageComponent](&world)
	_ = ecs.ComponentID[components.ContrabandComponent](&world)
	_ = ecs.ComponentID[components.BeliefComponent](&world)
	_ = ecs.ComponentID[components.JobComponent](&world)
	_ = ecs.ComponentID[components.EsotericMarker](&world)
	_ = ecs.ComponentID[components.DisguiseComponent](&world)
	_ = ecs.ComponentID[components.Velocity](&world)

	// Create Justice System
	justiceSys := NewJusticeSystem(&world, engine.NewSparseHookGraph())

	// Set up City 1 (Jurisdiction)
	city1 := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
	)

	city1Pos := (*components.Position)(world.Get(city1, ecs.ComponentID[components.Position](&world)))
	city1Pos.X = 10.0
	city1Pos.Y = 10.0

	city1Aff := (*components.Affiliation)(world.Get(city1, ecs.ComponentID[components.Affiliation](&world)))
	city1Aff.CityID = 1

	city1Jur := (*components.JurisdictionComponent)(world.Get(city1, ecs.ComponentID[components.JurisdictionComponent](&world)))
	city1Jur.RadiusSquared = 100.0 // Covering (0,0) to (20,20)
	city1Jur.IllegalActionIDs = 1 << components.InteractionAssault // Assault is illegal here

	// Create Criminal (Spy) - commits assault, but disguised as City 1 citizen
	spy := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.DisguiseComponent](&world),
	)

	spyPos := (*components.Position)(world.Get(spy, ecs.ComponentID[components.Position](&world)))
	spyPos.X = 12.0
	spyPos.Y = 12.0 // Inside jurisdiction

	spyAff := (*components.Affiliation)(world.Get(spy, ecs.ComponentID[components.Affiliation](&world)))
	spyAff.CityID = 2 // Actually from City 2

	spyMem := (*components.Memory)(world.Get(spy, ecs.ComponentID[components.Memory](&world)))
	// Log illegal assault
	spyMem.Events[0] = components.MemoryEvent{
		InteractionType: components.InteractionAssault,
		TickStamp:       10,
	}
	spyMem.Head = 1

	spyDisguise := (*components.DisguiseComponent)(world.Get(spy, ecs.ComponentID[components.DisguiseComponent](&world)))
	spyDisguise.SpoofedCityID = 1
	spyDisguise.IsActive = true

	// Create regular Criminal - commits assault, no disguise
	thug := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Memory](&world),
	)

	thugPos := (*components.Position)(world.Get(thug, ecs.ComponentID[components.Position](&world)))
	thugPos.X = 14.0
	thugPos.Y = 14.0 // Inside jurisdiction

	thugAff := (*components.Affiliation)(world.Get(thug, ecs.ComponentID[components.Affiliation](&world)))
	thugAff.CityID = 2

	thugMem := (*components.Memory)(world.Get(thug, ecs.ComponentID[components.Memory](&world)))
	// Log illegal assault
	thugMem.Events[0] = components.MemoryEvent{
		InteractionType: components.InteractionAssault,
		TickStamp:       10,
	}
	thugMem.Head = 1

	// Run Justice System
	justiceSys.Update(&world)

	// Check if CrimeMarkers were applied appropriately
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	if world.Has(spy, crimeID) {
		t.Errorf("Spy with active DisguiseComponent spoofing City 1 should not have been tagged with a CrimeMarker")
	}

	if !world.Has(thug, crimeID) {
		t.Errorf("Thug without DisguiseComponent should have been tagged with a CrimeMarker for Assault")
	}
}
