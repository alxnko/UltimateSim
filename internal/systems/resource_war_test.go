package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// TestResourceWarSystem_Integration verifies that Phase 29.1 Geopolitical Resource Wars
// deterministically hooks into Phase 23.1 Blood Feud by spawning massive negative hooks
// against all citizens of the defending country.
func TestResourceWarSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	sys := NewResourceWarSystem(&world, hookGraph)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	// 1. Create Attacker Capital (Starving Country 1)
	attCap := world.NewEntity(posID, affID, capID, marketID, storageID)
	attPos := (*components.Position)(world.Get(attCap, posID))
	attPos.X, attPos.Y = 10, 10

	attAff := (*components.Affiliation)(world.Get(attCap, affID))
	attAff.CountryID = 1

	attMarket := (*components.MarketComponent)(world.Get(attCap, marketID))
	attMarket.FoodPrice = 10.0 // Extreme Famine threshold

	attStorage := (*components.StorageComponent)(world.Get(attCap, storageID))
	attStorage.Food = 0

	// 2. Create Defender Capital (Wealthy Country 2)
	defCap := world.NewEntity(posID, affID, capID, marketID, storageID)
	defPos := (*components.Position)(world.Get(defCap, posID))
	defPos.X, defPos.Y = 20, 20 // DistSq = 10*10 + 10*10 = 200 (within 2500 range)

	defAff := (*components.Affiliation)(world.Get(defCap, affID))
	defAff.CountryID = 2

	defMarket := (*components.MarketComponent)(world.Get(defCap, marketID))
	defMarket.FoodPrice = 2.0

	defStorage := (*components.StorageComponent)(world.Get(defCap, storageID))
	defStorage.Food = 5000 // Massive Surplus

	// 3. Create NPCs belonging to both Countries
	// Attacker NPC
	attNPC := world.NewEntity(npcID, identID, affID)
	attNPCIdent := (*components.Identity)(world.Get(attNPC, identID))
	attNPCIdent.ID = 101

	attNPCAff := (*components.Affiliation)(world.Get(attNPC, affID))
	attNPCAff.CountryID = 1

	// Defender NPC
	defNPC := world.NewEntity(npcID, identID, affID)
	defNPCIdent := (*components.Identity)(world.Get(defNPC, identID))
	defNPCIdent.ID = 202

	defNPCAff := (*components.Affiliation)(world.Get(defNPC, affID))
	defNPCAff.CountryID = 2

	// Neutral NPC
	neuNPC := world.NewEntity(npcID, identID, affID)
	neuNPCIdent := (*components.Identity)(world.Get(neuNPC, identID))
	neuNPCIdent.ID = 303

	neuNPCAff := (*components.Affiliation)(world.Get(neuNPC, affID))
	neuNPCAff.CountryID = 3

	// 4. Run System (Tick 0 to 499 do not evaluate)
	for i := 0; i < 499; i++ {
		sys.Update(&world)
	}

	if world.Has(attCap, warCompID) {
		t.Fatalf("WarTrackerComponent applied too early")
	}

	// Tick 500 triggers the logic
	sys.Update(&world)

	// 5. Assert Structural Component Changes
	if !world.Has(attCap, warCompID) {
		t.Fatalf("Attacker Capital did not receive WarTrackerComponent")
	}

	warComp := (*components.WarTrackerComponent)(world.Get(attCap, warCompID))
	if warComp.TargetCountryID != 2 {
		t.Fatalf("Expected TargetCountryID 2, got %d", warComp.TargetCountryID)
	}
	if !warComp.Active {
		t.Fatalf("Expected WarTrackerComponent to be active")
	}

	// 6. Assert "Butterfly Effect" SparseHookGraph changes
	grudge := hookGraph.GetHook(101, 202)
	if grudge != -100 {
		t.Fatalf("Expected attacker NPC 101 to have -100 hook against defender NPC 202, got %d", grudge)
	}

	neutralGrudge := hookGraph.GetHook(101, 303)
	if neutralGrudge != 0 {
		t.Fatalf("Expected attacker NPC 101 to have 0 hook against neutral NPC 303, got %d", neutralGrudge)
	}
}
