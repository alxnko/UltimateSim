package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 21.1: E2E Test ensuring Economic Desperation triggers Justice crimes.
func TestDesperationSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	needsID := ecs.ComponentID[components.Needs](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)

	// Create Village with a Market and Storage
	villageEntity := world.NewEntity(posID, affID, marketID, villageID, storageID, jurID)

	vPos := (*components.Position)(world.Get(villageEntity, posID))
	vPos.X, vPos.Y = 10.0, 10.0

	vAff := (*components.Affiliation)(world.Get(villageEntity, affID))
	vAff.CityID = 1

	vMarket := (*components.MarketComponent)(world.Get(villageEntity, marketID))
	vMarket.FoodPrice = 50.0 // Starving economy

	vStorage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	vStorage.Food = 100 // Target food

	vJur := (*components.JurisdictionComponent)(world.Get(villageEntity, jurID))
	vJur.IllegalActionIDs = 1 << components.InteractionTheft
	vJur.RadiusSquared = 100.0

	// Create Desperate NPC
	npcEntity := world.NewEntity(needsID, affID, posID, despID, memID, idID)

	nPos := (*components.Position)(world.Get(npcEntity, posID))
	nPos.X, nPos.Y = 12.0, 12.0 // Very close

	nAff := (*components.Affiliation)(world.Get(npcEntity, affID))
	nAff.CityID = 1

	nNeeds := (*components.Needs)(world.Get(npcEntity, needsID))
	nNeeds.Food = 20.0 // Starving
	nNeeds.Wealth = 10.0 // Poor (Cannot afford 50.0 food price)

	nDesp := (*components.DesperationComponent)(world.Get(npcEntity, despID))
	nDesp.Level = 49 // Set one tick away from crime

	// Systems
	despSys := NewDesperationSystem(&world)
	justiceSys := NewJusticeSystem(&world, engine.NewSparseHookGraph())

	// Update Desperation - This should push Level to 50, trigger theft, update Memory, lower Storage, raise Needs
	despSys.Update(&world)

	// Assertions
	updatedStorage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	if updatedStorage.Food != 80 { // Stole 20
		t.Errorf("Expected Village Storage Food to be 80, got %v", updatedStorage.Food)
	}

	updatedNeeds := (*components.Needs)(world.Get(npcEntity, needsID))
	if updatedNeeds.Food != 40.0 { // Gained 20
		t.Errorf("Expected NPC Needs Food to be 40.0, got %v", updatedNeeds.Food)
	}

	updatedDesp := (*components.DesperationComponent)(world.Get(npcEntity, despID))
	if updatedDesp.Level != 0 { // Reset after eating
		t.Errorf("Expected NPC Desperation to reset to 0, got %v", updatedDesp.Level)
	}

	// Now run JusticeSystem to confirm it detects the Memory buffer theft and flags the NPC
	justiceSys.Update(&world)

	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	if !world.Has(npcEntity, crimeID) {
		t.Errorf("Expected NPC to be flagged with CrimeMarker by JusticeSystem after stealing.")
	}
}
