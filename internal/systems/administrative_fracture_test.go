package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 16.4: Administrative Reach & Friction
// AdministrativeFractureSystem Tests

// TestAdministrativeFractureSystem_Deterministic verifies Phase 16.4 DOD constraint matching and range limits
func TestAdministrativeFractureSystem_Deterministic(t *testing.T) {
	world := ecs.NewWorld()

	sys := NewAdministrativeFractureSystem(&world)

	villageID := ecs.ComponentID[components.Village](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)

	// 1. Create a Country Capital entity
	capitalEntity := world.NewEntity(countryID, capitalID, affilID, posID)

	capAffil := (*components.Affiliation)(world.Get(capitalEntity, affilID))
	capAffil.CountryID = 5 // Represents Country 5

	capPos := (*components.Position)(world.Get(capitalEntity, posID))
	capPos.X = 100.0
	capPos.Y = 100.0

	// 2. Create a Sub-City (Village) entity that belongs to Country 5, close to Capital
	villageEntityClose := world.NewEntity(villageID, affilID, posID)

	vilCloseAffil := (*components.Affiliation)(world.Get(villageEntityClose, affilID))
	vilCloseAffil.CountryID = 5 // Village is inside Country 5

	vilClosePos := (*components.Position)(world.Get(villageEntityClose, posID))
	vilClosePos.X = 150.0 // Distance = 50.0, which is < 150.0
	vilClosePos.Y = 100.0

	// 3. Create a Sub-City (Village) entity that belongs to Country 5, far from Capital
	villageEntityFar := world.NewEntity(villageID, affilID, posID)

	vilFarAffil := (*components.Affiliation)(world.Get(villageEntityFar, affilID))
	vilFarAffil.CountryID = 5 // Village is inside Country 5

	vilFarPos := (*components.Position)(world.Get(villageEntityFar, posID))
	vilFarPos.X = 300.0 // Distance = 200.0, which is > 150.0 threshold
	vilFarPos.Y = 100.0

	// 4. Run system for 999 ticks. No fracture should occur yet.
	for i := 0; i < 999; i++ {
		sys.Update(&world)
	}

	if vilCloseAffil.CountryID != 5 {
		t.Fatalf("Expected Close Village to remain in Country 5 before tick 1000, got %v", vilCloseAffil.CountryID)
	}

	if vilFarAffil.CountryID != 5 {
		t.Fatalf("Expected Far Village to remain in Country 5 before tick 1000, got %v", vilFarAffil.CountryID)
	}

	// 5. Run tick 1000. Fracture logic should occur.
	sys.Update(&world)

	if vilCloseAffil.CountryID != 5 {
		t.Fatalf("Expected Close Village to remain in Country 5 after tick 1000, got %v", vilCloseAffil.CountryID)
	}

	if vilFarAffil.CountryID != 0 {
		t.Fatalf("Expected Far Village to fracture and leave Country 5 (CountryID=0) after tick 1000, got %v", vilFarAffil.CountryID)
	}

	// 6. Test missing Capital behavior. Remove Capital and add a village assigned to Country 5
	world.RemoveEntity(capitalEntity)

	villageEntityOrphan := world.NewEntity(villageID, affilID, posID)

	vilOrphanAffil := (*components.Affiliation)(world.Get(villageEntityOrphan, affilID))
	vilOrphanAffil.CountryID = 5 // Village is inside Country 5

	vilOrphanPos := (*components.Position)(world.Get(villageEntityOrphan, posID))
	vilOrphanPos.X = 100.0
	vilOrphanPos.Y = 100.0

	// Run another 1000 ticks
	for i := 0; i < 1000; i++ {
		sys.Update(&world)
	}

	// The orphan village should fracture because its capital no longer exists
	if vilOrphanAffil.CountryID != 0 {
		t.Fatalf("Expected Orphan Village to fracture due to missing Capital after tick 2000, got %v", vilOrphanAffil.CountryID)
	}
}
