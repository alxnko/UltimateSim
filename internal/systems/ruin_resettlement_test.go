package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 40.2: The Ruins Resettlement Engine (Butterfly Effect E2E Test)
// Proves the integration where homeless NPCs (CityID=0) naturally seek and claim
// abandoned RuinComponent entities, removing the Ruin and restoring the Village natively.

func TestRuinResettlementSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize System
	sys := NewRuinResettlementSystem(&world)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	ruinID := ecs.ComponentID[components.RuinComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	slID := ecs.ComponentID[components.SettlementLogic](&world)

	// Create a Ruin Entity
	eRuin := world.NewEntity(posID, ruinID)
	rPos := (*components.Position)(world.Get(eRuin, posID))
	rPos.X = 10.0
	rPos.Y = 10.0

	// Create a Homeless NPC wandering entity
	eNPC := world.NewEntity(npcID, posID, slID, affID)
	npcPos := (*components.Position)(world.Get(eNPC, posID))
	npcPos.X = 10.0
	npcPos.Y = 10.0 // Same location as Ruin

	npcAff := (*components.Affiliation)(world.Get(eNPC, affID))
	npcAff.CityID = 0 // Homeless

	npcSl := (*components.SettlementLogic)(world.Get(eNPC, slID))
	npcSl.TicksAtZeroVelocity = 499 // Almost ready to settle

	// Advance ticks to trigger the update logic
	sys.tickCounter = 49

	sys.Update(&world)

	// Check state before resettlement trigger (should not have triggered yet since TicksAtZeroVelocity is 499)
	if !world.Has(eRuin, ruinID) {
		t.Fatalf("Expected Ruin entity to still have RuinComponent at tick 50 with SL=499")
	}

	// Advance SL to trigger
	npcSl.TicksAtZeroVelocity = 500
	sys.tickCounter = 99

	sys.Update(&world)

	// Assertions
	if world.Has(eRuin, ruinID) {
		t.Fatalf("Expected RuinComponent to be removed from Ruin entity upon resettlement")
	}

	if !world.Has(eRuin, villageID) {
		t.Fatalf("Expected Village component to be added back to Ruin entity upon resettlement")
	}

	if !world.Has(eRuin, storageID) {
		t.Fatalf("Expected StorageComponent to be added to resettled Village")
	}

	storage := (*components.StorageComponent)(world.Get(eRuin, storageID))
	if storage.Stone != 100 || storage.Food != 50 || storage.Wood != 50 {
		t.Errorf("Expected reclaimed storage bonuses (Stone: 100, Food: 50, Wood: 50), got Stone: %d, Food: %d, Wood: %d", storage.Stone, storage.Food, storage.Wood)
	}

	if !world.Has(eRuin, popID) {
		t.Fatalf("Expected PopulationComponent to be added to resettled Village")
	}

	pop := (*components.PopulationComponent)(world.Get(eRuin, popID))
	if pop.Count != 1 {
		t.Errorf("Expected resettled Village to have Population Count 1, got %d", pop.Count)
	}

	if !world.Has(eRuin, needsID) {
		t.Fatalf("Expected Needs component to be restored to resettled Village")
	}

	if !world.Has(eRuin, marketID) {
		t.Fatalf("Expected MarketComponent to be restored to resettled Village")
	}

	market := (*components.MarketComponent)(world.Get(eRuin, marketID))
	if market.FoodPrice != 1.0 {
		t.Errorf("Expected MarketComponent.FoodPrice to be initialized to 1.0, got %f", market.FoodPrice)
	}

	// Validate Affiliation updating (closing the cycle)
	if !world.Has(eRuin, affID) {
		t.Fatalf("Expected Affiliation component to be added to resettled Village")
	}

	ruinAff := (*components.Affiliation)(world.Get(eRuin, affID))

	// Check NPC CityID
	updatedNpcAff := (*components.Affiliation)(world.Get(eNPC, affID))
	if updatedNpcAff.CityID == 0 {
		t.Fatalf("Expected homeless NPC to have CityID updated upon resettling, got 0")
	}

	if updatedNpcAff.CityID != ruinAff.CityID {
		t.Errorf("Expected NPC CityID (%d) to match reborn Village CityID (%d)", updatedNpcAff.CityID, ruinAff.CityID)
	}

	// Check NPC SL Reset
	updatedNpcSl := (*components.SettlementLogic)(world.Get(eNPC, slID))
	if updatedNpcSl.TicksAtZeroVelocity != 0 {
		t.Errorf("Expected NPC SettlementLogic to be reset to 0 upon settling, got %d", updatedNpcSl.TicksAtZeroVelocity)
	}
}
