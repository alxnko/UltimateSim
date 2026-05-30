package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine (End-to-End Test)
// Demonstrates the "Butterfly Effect":
// 1. A war starts between Country A and B.
// 2. Hostile NPCs from A surround a Village in B.
// 3. SiegeSystem applies a SiegeMarker, spiking FoodPrice and draining Loyalty.
// 4. VassalRebellionSystem detects the massive starvation and 0 loyalty, causing the Village to secede.
// 5. The citizens naturally spawn -100 hooks against their former ruler via BloodFeudSystem.
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration
	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	desperationID := ecs.ComponentID[components.DesperationComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)

	hooks := engine.NewSparseHookGraph()

	// 2. Initialize Systems
	siegeSys := NewSiegeSystem(&world)
	vassalRebellionSys := NewVassalRebellionSystem(&world, hooks)

	// 3. Spawn Entities
	// Capital B (Target of War)
	eCapitalB := world.NewEntity(identID, affID, capitalID, countryID)
	cIdentB := (*components.Identity)(world.Get(eCapitalB, identID))
	cIdentB.ID = 101
	cAffilB := (*components.Affiliation)(world.Get(eCapitalB, affID))
	cAffilB.CountryID = 2

	// Capital A (Attacker)
	eCapitalA := world.NewEntity(identID, affID, capitalID, countryID, warCompID)
	cAffilA := (*components.Affiliation)(world.Get(eCapitalA, affID))
	cAffilA.CountryID = 1
	cWarA := (*components.WarTrackerComponent)(world.Get(eCapitalA, warCompID))
	cWarA.Active = true
	cWarA.TargetCountryID = 2

	// Village in Country B
	eVillage := world.NewEntity(posID, affID, villageID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(eVillage, posID))
	vPos.X, vPos.Y = 50.0, 50.0
	vAffil := (*components.Affiliation)(world.Get(eVillage, affID))
	vAffil.CityID = 202
	vAffil.CountryID = 2
	vMarket := (*components.MarketComponent)(world.Get(eVillage, marketID))
	vMarket.FoodPrice = 1.0
	vLoyalty := (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))
	vLoyalty.Value = 100 // High loyalty initially

	// Friendly NPC in Village (Country B)
	eFriendly := world.NewEntity(posID, identID, affID, npcID, desperationID)
	fPos := (*components.Position)(world.Get(eFriendly, posID))
	fPos.X, fPos.Y = 50.0, 51.0
	fAffil := (*components.Affiliation)(world.Get(eFriendly, affID))
	fAffil.CountryID = 2
	fAffil.CityID = 202
	fIdent := (*components.Identity)(world.Get(eFriendly, identID))
	fIdent.ID = 303
	fDesp := (*components.DesperationComponent)(world.Get(eFriendly, desperationID))
	fDesp.Level = 80 // Desperate citizen

	// Hostile NPCs from Country A (2 attackers vs 1 friendly)
	for i := 0; i < 2; i++ {
		eHostile := world.NewEntity(posID, affID, npcID)
		hPos := (*components.Position)(world.Get(eHostile, posID))
		hPos.X, hPos.Y = 49.0 + float32(i), 49.0
		hAffil := (*components.Affiliation)(world.Get(eHostile, affID))
		hAffil.CountryID = 1
	}

	// ---------------------------------------------------------
	// TICK 100: Siege System detects siege and applies marker
	// ---------------------------------------------------------
	siegeSys.tickCounter = 99 // Manually skip to right before mod
	siegeSys.Update(&world)

	if !world.Has(eVillage, siegeID) {
		t.Fatalf("Tick 100: Village was not tagged with SiegeMarker")
	}

	vSiege := (*components.SiegeMarker)(world.Get(eVillage, siegeID))
	if vSiege.BesiegerCountryID != 1 {
		t.Fatalf("Tick 100: Expected BesiegerCountryID 1, got %d", vSiege.BesiegerCountryID)
	}

	// ---------------------------------------------------------
	// TICK 200: Siege System applies starvation & loyalty penalties
	// ---------------------------------------------------------
	siegeSys.tickCounter = 199
	siegeSys.Update(&world)

	// Fetch pointers safely again
	vMarket = (*components.MarketComponent)(world.Get(eVillage, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))

	if vMarket.FoodPrice <= 1.0 {
		t.Fatalf("Tick 200: FoodPrice should have spiked due to siege")
	}

	if vLoyalty.Value >= 100 {
		t.Fatalf("Tick 200: Loyalty should have drained due to siege")
	}

	// ---------------------------------------------------------
	// FAST FORWARD to force Loyalty to 0
	// ---------------------------------------------------------
	for i := 0; i < 20; i++ {
		siegeSys.tickCounter = uint64(299 + (i * 100))
		siegeSys.Update(&world)
	}

	vLoyalty = (*components.LoyaltyComponent)(world.Get(eVillage, loyaltyID))
	if vLoyalty.Value != 0 {
		t.Fatalf("Fast Forward: Loyalty should be exactly 0, got %d", vLoyalty.Value)
	}

	// ---------------------------------------------------------
	// TRIGGER VASSAL REBELLION
	// ---------------------------------------------------------
	vassalRebellionSys.tickStamp = 99
	vassalRebellionSys.Update(&world)

	vAffil = (*components.Affiliation)(world.Get(eVillage, affID))
	if vAffil.CountryID != 0 {
		t.Fatalf("Rebellion: Village should have seceded (CountryID 0), got %d", vAffil.CountryID)
	}

	// Assert that the friendly citizen spawned a -100 Blood Feud against their former ruler
	hookValue := hooks.GetHook(fIdent.ID, cIdentB.ID)
	if hookValue != -100 {
		t.Fatalf("Rebellion: Expected -100 Blood Feud hook from citizen against former ruler, got %d", hookValue)
	}
}
