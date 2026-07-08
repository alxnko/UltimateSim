package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestEspionageSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	// Register needed components explicitly for test stability
	_ = ecs.ComponentID[components.Position](&world)
	_ = ecs.ComponentID[components.Affiliation](&world)
	_ = ecs.ComponentID[components.JurisdictionComponent](&world)
	_ = ecs.ComponentID[components.Memory](&world)
	_ = ecs.ComponentID[components.Identity](&world)
	_ = ecs.ComponentID[components.CrimeMarker](&world)
	_ = ecs.ComponentID[components.JobComponent](&world)
	_ = ecs.ComponentID[components.Path](&world)
	_ = ecs.ComponentID[components.Velocity](&world)
	_ = ecs.ComponentID[components.Needs](&world)
	_ = ecs.ComponentID[components.SecretComponent](&world)
	_ = ecs.ComponentID[components.DisguiseComponent](&world)

	// Create JusticeSystem
	js := NewJusticeSystem(&world, hooks)

	// 1. Create a City (Jurisdiction)
	cityID := uint32(101)
	cityEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
	)

	cityPos := (*components.Position)(world.Get(cityEnt, ecs.ComponentID[components.Position](&world)))
	cityPos.X, cityPos.Y = 100, 100

	cityAff := (*components.Affiliation)(world.Get(cityEnt, ecs.ComponentID[components.Affiliation](&world)))
	cityAff.CityID = cityID

	cityJur := (*components.JurisdictionComponent)(world.Get(cityEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	cityJur.RadiusSquared = 400.0 // Radius 20
	cityJur.IllegalActionIDs = 1 << components.InteractionTheft // Theft is illegal

	// 2. Create a Guard
	guardEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Path](&world),
		ecs.ComponentID[components.Velocity](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	gPos := (*components.Position)(world.Get(guardEnt, ecs.ComponentID[components.Position](&world)))
	gPos.X, gPos.Y = 105, 105

	gAff := (*components.Affiliation)(world.Get(guardEnt, ecs.ComponentID[components.Affiliation](&world)))
	gAff.CityID = cityID

	gJob := (*components.JobComponent)(world.Get(guardEnt, ecs.ComponentID[components.JobComponent](&world)))
	gJob.JobID = components.JobGuard

	gIdent := (*components.Identity)(world.Get(guardEnt, ecs.ComponentID[components.Identity](&world)))
	gIdent.ID = 1

	// 3. Create a normal criminal (no disguise)
	crim1Ent := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	c1Pos := (*components.Position)(world.Get(crim1Ent, ecs.ComponentID[components.Position](&world)))
	c1Pos.X, c1Pos.Y = 110, 110

	c1Aff := (*components.Affiliation)(world.Get(crim1Ent, ecs.ComponentID[components.Affiliation](&world)))
	c1Aff.CityID = 202 // Foreigner

	c1Mem := (*components.Memory)(world.Get(crim1Ent, ecs.ComponentID[components.Memory](&world)))
	c1Mem.Events[0] = components.MemoryEvent{
		InteractionType: components.InteractionTheft, // Committed theft
	}
	c1Mem.Head = 1

	c1Ident := (*components.Identity)(world.Get(crim1Ent, ecs.ComponentID[components.Identity](&world)))
	c1Ident.ID = 2

	// 4. Create a disguised criminal
	crim2Ent := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.DisguiseComponent](&world),
	)

	c2Pos := (*components.Position)(world.Get(crim2Ent, ecs.ComponentID[components.Position](&world)))
	c2Pos.X, c2Pos.Y = 90, 90

	c2Aff := (*components.Affiliation)(world.Get(crim2Ent, ecs.ComponentID[components.Affiliation](&world)))
	c2Aff.CityID = 202 // Foreigner

	c2Mem := (*components.Memory)(world.Get(crim2Ent, ecs.ComponentID[components.Memory](&world)))
	c2Mem.Events[0] = components.MemoryEvent{
		InteractionType: components.InteractionTheft, // Committed theft
	}
	c2Mem.Head = 1

	c2Ident := (*components.Identity)(world.Get(crim2Ent, ecs.ComponentID[components.Identity](&world)))
	c2Ident.ID = 3

	c2Disg := (*components.DisguiseComponent)(world.Get(crim2Ent, ecs.ComponentID[components.DisguiseComponent](&world)))
	c2Disg.IsActive = true
	c2Disg.SpoofedCityID = cityID // Disguised as local

	// 5. Run Justice System
	js.Update(&world)

	// Verify crim1 got a CrimeMarker
	if !world.Has(crim1Ent, ecs.ComponentID[components.CrimeMarker](&world)) {
		t.Errorf("Expected criminal 1 to be tagged with a CrimeMarker")
	}

	// Verify crim2 bypassed detection
	if world.Has(crim2Ent, ecs.ComponentID[components.CrimeMarker](&world)) {
		t.Errorf("Expected disguised criminal 2 to bypass detection, but got a CrimeMarker")
	}

	// Verify guard targeted crim1
	gPath := (*components.Path)(world.Get(guardEnt, ecs.ComponentID[components.Path](&world)))
	// Note: Because float32 equality can be tricky, check if target was changed from default 0
	if gPath.TargetX == 0 || gPath.TargetY == 0 {
		t.Errorf("Expected Guard to target criminal 1, but path target was not updated")
	} else if int(gPath.TargetX) != 110 || int(gPath.TargetY) != 110 {
		t.Errorf("Expected Guard to target criminal 1 at (110, 110), got (%f, %f)", gPath.TargetX, gPath.TargetY)
	}

	// Now tag crim2 explicitly to see if guard ignores them
	world.Add(crim2Ent, ecs.ComponentID[components.CrimeMarker](&world))
	world.Remove(crim1Ent, ecs.ComponentID[components.CrimeMarker](&world))

	// Re-run system to test Guard targeting evasion
	js.Update(&world)

	gPath2 := (*components.Path)(world.Get(guardEnt, ecs.ComponentID[components.Path](&world)))
	if int(gPath2.TargetX) == 90 && int(gPath2.TargetY) == 90 {
		t.Errorf("Expected Guard to ignore disguised criminal 2, but targeted them.")
	}
}
