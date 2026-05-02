package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine (End-to-End Test)
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	siegeSys := systems.NewSiegeSystem(&world)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	// Create Defender Country (CountryID = 10)
	eDefenderCap := world.NewEntity(capID, affID)
	defAff := (*components.Affiliation)(world.Get(eDefenderCap, affID))
	defAff.CountryID = 10

	// Create Defending Village (Belongs to Country 10)
	eVillage := world.NewEntity(villageID, affID, posID, marketID, loyaltyID)
	vAff := (*components.Affiliation)(world.Get(eVillage, affID))
	vAff.CountryID = 10
	vPos := (*components.Position)(world.Get(eVillage, posID))
	vPos.X, vPos.Y = 50.0, 50.0
	vMarket := (*components.MarketComponent)(world.Get(eVillage, marketID))
	vMarket.FoodPrice = 5.0
	vLoyalty := (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))
	vLoyalty.Value = 100

	// Create Attacker Country (CountryID = 20) with Active War
	eAttackerCap := world.NewEntity(capID, affID, warCompID)
	attAff := (*components.Affiliation)(world.Get(eAttackerCap, affID))
	attAff.CountryID = 20
	warComp := (*components.WarTrackerComponent)(world.Get(eAttackerCap, warCompID))
	warComp.TargetCountryID = 10
	warComp.Active = true

	// Create 1 Defending Guard
	eDefGuard := world.NewEntity(npcID, posID, affID, jobID, vitalsID)
	dgAff := (*components.Affiliation)(world.Get(eDefGuard, affID))
	dgAff.CountryID = 10
	dgPos := (*components.Position)(world.Get(eDefGuard, posID))
	dgPos.X, dgPos.Y = 51.0, 51.0 // Within range
	dgJob := (*components.JobComponent)(world.Get(eDefGuard, jobID))
	dgJob.JobID = components.JobGuard
	dgVitals := (*components.VitalsComponent)(world.Get(eDefGuard, vitalsID))
	dgVitals.Blood = 100.0
	dgVitals.Consciousness = 100.0

	// Create 3 Attacking Guards (To satisfy 9 > 6)
	eAttGuard1 := world.NewEntity(npcID, posID, affID, jobID, vitalsID)
	ag1Aff := (*components.Affiliation)(world.Get(eAttGuard1, affID))
	ag1Aff.CountryID = 20
	ag1Pos := (*components.Position)(world.Get(eAttGuard1, posID))
	ag1Pos.X, ag1Pos.Y = 49.0, 49.0 // Within range
	ag1Job := (*components.JobComponent)(world.Get(eAttGuard1, jobID))
	ag1Job.JobID = components.JobGuard
	ag1Vitals := (*components.VitalsComponent)(world.Get(eAttGuard1, vitalsID))
	ag1Vitals.Blood = 100.0
	ag1Vitals.Consciousness = 100.0

	eAttGuard2 := world.NewEntity(npcID, posID, affID, jobID, vitalsID)
	ag2Aff := (*components.Affiliation)(world.Get(eAttGuard2, affID))
	ag2Aff.CountryID = 20
	ag2Pos := (*components.Position)(world.Get(eAttGuard2, posID))
	ag2Pos.X, ag2Pos.Y = 49.0, 50.0 // Within range
	ag2Job := (*components.JobComponent)(world.Get(eAttGuard2, jobID))
	ag2Job.JobID = components.JobGuard
	ag2Vitals := (*components.VitalsComponent)(world.Get(eAttGuard2, vitalsID))
	ag2Vitals.Blood = 100.0
	ag2Vitals.Consciousness = 100.0

	eAttGuard3 := world.NewEntity(npcID, posID, affID, jobID, vitalsID)
	ag3Aff := (*components.Affiliation)(world.Get(eAttGuard3, affID))
	ag3Aff.CountryID = 20
	ag3Pos := (*components.Position)(world.Get(eAttGuard3, posID))
	ag3Pos.X, ag3Pos.Y = 50.0, 49.0 // Within range
	ag3Job := (*components.JobComponent)(world.Get(eAttGuard3, jobID))
	ag3Job.JobID = components.JobGuard
	ag3Vitals := (*components.VitalsComponent)(world.Get(eAttGuard3, vitalsID))
	ag3Vitals.Blood = 100.0
	ag3Vitals.Consciousness = 100.0


	// Force system to trigger
	for i := 0; i < 100; i++ {
		siegeSys.Update(&world)
	}

	// Verify siege was applied
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	if !world.Has(eVillage, siegeID) {
		t.Fatalf("Village was not placed under siege despite being outnumbered")
	}

	marker := (*components.SiegeMarker)(world.Get(eVillage, siegeID))
	if marker.AttackerCountryID != 20 {
		t.Errorf("Expected SiegeMarker.AttackerCountryID 20, got %d", marker.AttackerCountryID)
	}

	// Verify consequences
	vMarketAfter := (*components.MarketComponent)(world.Get(eVillage, marketID))
	if vMarketAfter.FoodPrice <= 5.0 {
		t.Errorf("Expected FoodPrice to spike due to siege, got %f", vMarketAfter.FoodPrice)
	}

	vLoyaltyAfter := (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))
	if vLoyaltyAfter.Value >= 100 {
		t.Errorf("Expected Loyalty to drain due to siege, got %d", vLoyaltyAfter.Value)
	}

	// Test lifting the siege (Attacker dies)
	ag2Vitals.Blood = 0 // Dead
	ag3Vitals.Blood = 0 // Dead

	for i := 0; i < 100; i++ {
		siegeSys.Update(&world)
	}

	if world.Has(eVillage, siegeID) {
		t.Errorf("Village should no longer be under siege since attackers don't outnumber 2:1")
	}
}
