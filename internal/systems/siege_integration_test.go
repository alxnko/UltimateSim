package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine (End-to-End Test)
// Demonstrates the "Butterfly Effect": Geopolitical war intent evaluates spatial NPC physical presence,
// outnumbering triggers the SiegeMarker, which immediately bridges into the Economy
// by hyperinflating FoodPrice and draining the target's Loyalty.
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	villID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	// 2. Initialize System
	siegeSys := NewSiegeSystem(&world)

	// 3. Spawn Entities
	// Target Village (Country 1)
	eVillage := world.NewEntity(villID, posID, affID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(eVillage, posID))
	vPos.X, vPos.Y = 10.0, 10.0
	vAff := (*components.Affiliation)(world.Get(eVillage, affID))
	vAff.CountryID = 1
	vMarket := (*components.MarketComponent)(world.Get(eVillage, marketID))
	vMarket.FoodPrice = 5.0
	vLoyalty := (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))
	vLoyalty.Value = 100

	// Attacker Country (Country 2 attacking Country 1)
	eAttackerCapital := world.NewEntity(capID, warID, affID)
	aAff := (*components.Affiliation)(world.Get(eAttackerCapital, affID))
	aAff.CountryID = 2
	aWar := (*components.WarTrackerComponent)(world.Get(eAttackerCapital, warID))
	aWar.Active = true
	aWar.TargetCountryID = 1

	// NPCs
	// 1 Defender
	eDef := world.NewEntity(npcID, posID, affID, vitalsID)
	dPos := (*components.Position)(world.Get(eDef, posID))
	dPos.X, dPos.Y = 10.0, 10.0 // Exactly at village
	dAff := (*components.Affiliation)(world.Get(eDef, affID))
	dAff.CountryID = 1

	// 2 Attackers (outnumbering defender)
	for i := 0; i < 2; i++ {
		eAtt := world.NewEntity(npcID, posID, affID, vitalsID)
		aPos := (*components.Position)(world.Get(eAtt, posID))
		aPos.X, aPos.Y = 12.0, 12.0 // Close to village, distSq = 8 <= 100
		aNpcAff := (*components.Affiliation)(world.Get(eAtt, affID))
		aNpcAff.CountryID = 2
	}

	// 4. Tick the System (Tick 10 evaluates and applies SiegeMarker)
	for i := 0; i < 10; i++ {
		siegeSys.Update(&world)
	}

	// Re-fetch pointers after potential structural change
	if !world.Has(eVillage, siegeID) {
		t.Fatalf("Expected Village to receive SiegeMarker due to outnumbering attackers")
	}

	vSiege := (*components.SiegeMarker)(world.Get(eVillage, siegeID))
	if vSiege.BesiegerCountryID != 2 {
		t.Errorf("Expected BesiegerCountryID 2, got %d", vSiege.BesiegerCountryID)
	}

	// 5. Tick System again to trigger emergent starvation effects
	for i := 0; i < 10; i++ {
		siegeSys.Update(&world)
	}

	vMarket = (*components.MarketComponent)(world.Get(eVillage, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))

	if vMarket.FoodPrice <= 5.0 {
		t.Errorf("Expected FoodPrice to hyperinflate during siege, got %f", vMarket.FoodPrice)
	}
	if vLoyalty.Value >= 100 {
		t.Errorf("Expected Loyalty to drain during siege, got %d", vLoyalty.Value)
	}

	// 6. Break the Siege (Attackers run away out of range)
	// Iterate through NPC query to move attackers away
	filter := ecs.All(npcID, affID, posID)
	query := world.Query(filter)
	for query.Next() {
		aff := (*components.Affiliation)(query.Get(affID))
		if aff.CountryID == 2 {
			pos := (*components.Position)(query.Get(posID))
			pos.X, pos.Y = 1000.0, 1000.0 // Move far away
		}
	}

	// Tick to remove SiegeMarker
	for i := 0; i < 10; i++ {
		siegeSys.Update(&world)
	}

	if world.Has(eVillage, siegeID) {
		t.Errorf("Expected SiegeMarker to be removed after attackers retreated")
	}
}
