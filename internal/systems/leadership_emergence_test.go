package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 43: Organic Administration Engine Integration Test
func TestLeadershipEmergenceSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	sys := NewLeadershipEmergenceSystem(hookGraph)

	npcID := ecs.ComponentID[components.NPC](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](&world)

	// Create a city with two NPCs
	cityID := uint32(1)

	npc1 := world.NewEntity(npcID, identID, affilID)
	ident1 := (*components.Identity)(world.Get(npc1, identID))
	ident1.ID = 101
	affil1 := (*components.Affiliation)(world.Get(npc1, affilID))
	affil1.CityID = cityID

	npc2 := world.NewEntity(npcID, identID, affilID)
	ident2 := (*components.Identity)(world.Get(npc2, identID))
	ident2.ID = 102
	affil2 := (*components.Affiliation)(world.Get(npc2, affilID))
	affil2.CityID = cityID

	// Scenario 1: NPC 1 has positive hooks
	hookGraph.AddHook(201, ident1.ID, 50)
	hookGraph.AddHook(202, ident1.ID, 50)
	// NPC 2 has none

	for i := 0; i < LeadershipEmergenceTickRate; i++ {
		sys.Update(&world)
	}

	// Verify NPC 1 became the leader
	if !world.Has(npc1, adminMarkerID) {
		t.Errorf("NPC 1 should have gained the AdministrationMarker")
	}
	if world.Has(npc2, adminMarkerID) {
		t.Errorf("NPC 2 should not have the AdministrationMarker")
	}

	// Scenario 2: NPC 2 gets MASSIVE positive hooks, dethroning NPC 1
	hookGraph.AddHook(203, 102, 500)

	for i := 0; i < LeadershipEmergenceTickRate; i++ {
		sys.Update(&world)
	}

	// Verify NPC 2 became the leader and NPC 1 lost the marker
	if world.Has(npc1, adminMarkerID) {
		t.Errorf("NPC 1 should have lost the AdministrationMarker, score1: %v, score2: %v", hookGraph.GetAllIncomingHooks(101), hookGraph.GetAllIncomingHooks(102))
	}
	if !world.Has(npc2, adminMarkerID) {
		t.Errorf("NPC 2 should have gained the AdministrationMarker, score1: %v, score2: %v", hookGraph.GetAllIncomingHooks(101), hookGraph.GetAllIncomingHooks(102))
	}

	// Verify integration with TaxationSystem
	taxSys := NewTaxationSystem(&world, hookGraph)

	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)

	// Create Capital (Needs Identity as per existing code fallback though it prefers the marker now)
	capital := world.NewEntity(countryID, capitalID, affilID, treasuryID, jurID, identID)
	capAffil := (*components.Affiliation)(world.Get(capital, affilID))
	capAffil.CountryID = 10
	capAffil.CityID = cityID
	capJur := (*components.JurisdictionComponent)(world.Get(capital, jurID))
	capJur.Corruption = 50 // High corruption

	// Create Village that will evade taxes
	villageCityID := uint32(2)
	village := world.NewEntity(villageID, affilID, marketID, treasuryID, loyaltyID)
	villAffil := (*components.Affiliation)(world.Get(village, affilID))
	villAffil.CountryID = 10
	villAffil.CityID = villageCityID
	villLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	villLoyalty.Value = 10 // Low loyalty, evades taxes

	// Add NPC in the village to be target of grudges
	npc3 := world.NewEntity(npcID, identID, affilID)
	ident3 := (*components.Identity)(world.Get(npc3, identID))
	ident3.ID = 103
	affil3 := (*components.Affiliation)(world.Get(npc3, affilID))
	affil3.CityID = villageCityID

	taxSys.tickStamp = 99
	taxSys.Update(&world)

	// The ruler of the capital is NPC 2. The village NPC (103) should get a -50 hook against the ruler (102).
	grudge := hookGraph.GetHook(103, 102)
	if grudge != -50 {
		t.Errorf("Expected village NPC (103) to get -50 grudge against emergent Ruler (102), got %d", grudge)
	}
}
