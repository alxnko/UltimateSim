package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 32: Espionage & Disguises Engine - Integration Test
func TestDisguiseSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register all relevant components
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Path](&world)
	ecs.ComponentID[components.Velocity](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.Memory](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.CrimeMarker](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.DisguiseComponent](&world)
	ecs.ComponentID[components.Needs](&world)
	ecs.ComponentID[components.TreasuryComponent](&world)
	ecs.ComponentID[components.StorageComponent](&world)

	hooks := engine.NewSparseHookGraph()
	sys := NewJusticeSystem(&world, hooks)

	cityID := uint32(42)

	// Create Jurisdiction
	jurEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
	)
	jurPos := (*components.Position)(world.Get(jurEnt, ecs.ComponentID[components.Position](&world)))
	jurPos.X, jurPos.Y = 0, 0
	jurAff := (*components.Affiliation)(world.Get(jurEnt, ecs.ComponentID[components.Affiliation](&world)))
	jurAff.CityID = cityID
	jurComp := (*components.JurisdictionComponent)(world.Get(jurEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	jurComp.RadiusSquared = 100.0
	jurComp.IllegalActionIDs = uint32(1 << components.InteractionTheft) // Theft is illegal

	// Create a Guard
	guardEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Path](&world),
		ecs.ComponentID[components.Velocity](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Identity](&world),
	)
	guardPos := (*components.Position)(world.Get(guardEnt, ecs.ComponentID[components.Position](&world)))
	guardPos.X, guardPos.Y = 5, 5
	guardJob := (*components.JobComponent)(world.Get(guardEnt, ecs.ComponentID[components.JobComponent](&world)))
	guardJob.JobID = components.JobGuard
	guardAff := (*components.Affiliation)(world.Get(guardEnt, ecs.ComponentID[components.Affiliation](&world)))
	guardAff.CityID = cityID

	// Create normal Criminal (no disguise)
	crim1Ent := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Needs](&world),
	)
	crim1Pos := (*components.Position)(world.Get(crim1Ent, ecs.ComponentID[components.Position](&world)))
	crim1Pos.X, crim1Pos.Y = 2, 2
	crim1Mem := (*components.Memory)(world.Get(crim1Ent, ecs.ComponentID[components.Memory](&world)))
	crim1Mem.Events[0] = components.MemoryEvent{InteractionType: components.InteractionTheft}
	crim1Mem.Head = 1

	// Create disguised Criminal
	crim2Ent := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.DisguiseComponent](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Needs](&world),
	)
	crim2Pos := (*components.Position)(world.Get(crim2Ent, ecs.ComponentID[components.Position](&world)))
	crim2Pos.X, crim2Pos.Y = 3, 3 // Close to guard
	crim2Mem := (*components.Memory)(world.Get(crim2Ent, ecs.ComponentID[components.Memory](&world)))
	crim2Mem.Events[0] = components.MemoryEvent{InteractionType: components.InteractionTheft}
	crim2Mem.Head = 1
	crim2Disg := (*components.DisguiseComponent)(world.Get(crim2Ent, ecs.ComponentID[components.DisguiseComponent](&world)))
	crim2Disg.IsActive = true
	crim2Disg.SpoofedCityID = cityID // Spoofing as same city

	// Run system
	sys.Update(&world)

	// Verify crim1 got marked as criminal
	if !world.Has(crim1Ent, ecs.ComponentID[components.CrimeMarker](&world)) {
		t.Errorf("Expected normal criminal to be given CrimeMarker")
	}

	// Verify crim2 bypassed detection
	if world.Has(crim2Ent, ecs.ComponentID[components.CrimeMarker](&world)) {
		t.Errorf("Expected disguised criminal to bypass CrimeMarker detection")
	}

	// Add CrimeMarker manually to disguised criminal to test Guard bypassing
	world.Add(crim2Ent, ecs.ComponentID[components.CrimeMarker](&world))
	sys.Update(&world)

	// Since Guard targets closest criminal (distSq to crim2 is 8, distSq to crim1 is 18)
	// Guard normally would target crim2, but should bypass due to disguise.
	// Guard will try to move to crim1.
	guardPath := (*components.Path)(world.Get(guardEnt, ecs.ComponentID[components.Path](&world)))

	// Wait, Guard just sets Path.TargetX to criminal's X
	// In the system, if it targets crim1, TargetX will be 2
	if guardPath.TargetX != 2 || guardPath.TargetY != 2 {
		t.Errorf("Guard should have targeted crim1 at (2,2) bypassing disguised crim2 at (3,3). Guard path target: (%f, %f)", guardPath.TargetX, guardPath.TargetY)
	}
}
