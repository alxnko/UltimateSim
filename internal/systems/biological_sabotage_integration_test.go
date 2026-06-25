package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 61: The Biological Sabotage Engine Integration Test
// Tests the full Butterfly Effect from Class Warfare -> Sabotage -> Ecology/Justice

func TestBiologicalSabotageSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	cwSys := NewClassWarfareSystem(&world, hooks)
	bsSys := NewBiologicalSabotageSystem(&world, hooks)

	cityID := uint32(10)
	rulerID := uint64(500)
	peasantID := uint64(501)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	villID := ecs.ComponentID[components.Village](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	adminID := ecs.ComponentID[components.AdministrationMarker](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	disID := ecs.ComponentID[components.DiseaseEntity](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	// 1. Create the Wealthy Hoarding Village
	cityE := world.NewEntity()
	world.Add(cityE, villID, posID, storID, marketID, affID)

	vPos := (*components.Position)(world.Get(cityE, posID))
	vPos.X, vPos.Y = 10.0, 10.0

	vStor := (*components.StorageComponent)(world.Get(cityE, storID))
	vStor.Food = 1000 // Hoarding!

	vMark := (*components.MarketComponent)(world.Get(cityE, marketID))
	vMark.FoodPrice = 50.0 // Expensive

	vAff := (*components.Affiliation)(world.Get(cityE, affID))
	vAff.CityID = cityID

	// 2. Create the Ruler
	rulerE := world.NewEntity()
	world.Add(rulerE, adminID, affID, idID)

	rAff := (*components.Affiliation)(world.Get(rulerE, affID))
	rAff.CityID = cityID

	rId := (*components.Identity)(world.Get(rulerE, idID))
	rId.ID = rulerID

	// 3. Create the Starving Peasant physically at the village
	peasantE := world.NewEntity()
	world.Add(peasantE, npcID, needsID, affID, idID, posID)

	pPos := (*components.Position)(world.Get(peasantE, posID))
	pPos.X, pPos.Y = 10.5, 10.5 // distance squared = 0.5 (< 2.0)

	pNeeds := (*components.Needs)(world.Get(peasantE, needsID))
	pNeeds.Food = 10.0  // Starving (< 20)
	pNeeds.Wealth = 5.0 // Too poor (< 50)

	pAff := (*components.Affiliation)(world.Get(peasantE, affID))
	pAff.CityID = cityID

	pId := (*components.Identity)(world.Get(peasantE, idID))
	pId.ID = peasantID

	// ACT - Step 1: Class Warfare generates Grudge
	// Need to run it 21 times because it updates every 50 ticks, and adds -5 per update.
	// We need <= -100, so we need 20 updates (-100).
	// We loop enough to trigger Update multiple times.

	// Fast forward Class Warfare
	for i := 0; i < 50*20; i++ {
		cwSys.Update(&world)
	}

	grudge := hooks.GetHook(peasantID, rulerID)
	if grudge > -100 {
		t.Fatalf("Expected grudge to be <= -100, got %d", grudge)
	}

	// Verify before sabotage
	if vStor.Food != 1000 {
		t.Fatalf("Food should be 1000 before sabotage, got %d", vStor.Food)
	}

	// ACT - Step 2: Biological Sabotage
	// Fast forward Biological Sabotage until its tick condition fires (mod 53 == 0)
	for i := 0; i < 53; i++ {
		bsSys.Update(&world)
	}

	// ASSERT
	// 1. Hook spent
	newGrudge := hooks.GetHook(peasantID, rulerID)
	if newGrudge != grudge+100 {
		t.Errorf("Expected grudge to be partially spent (+100), got %d", newGrudge)
	}

	// 2. Food halved
	if vStor.Food != 500 {
		t.Errorf("Expected village food to be halved to 500, got %d", vStor.Food)
	}

	// 3. Crime Marker assigned
	if !world.Has(peasantE, crimeID) {
		t.Errorf("Expected peasant to receive a CrimeMarker")
	} else {
		cm := (*components.CrimeMarker)(world.Get(peasantE, crimeID))
		if cm.CrimeLevel != 3 || cm.Bounty != 500 {
			t.Errorf("Expected CrimeLevel 3 and Bounty 500, got Level %d, Bounty %d", cm.CrimeLevel, cm.Bounty)
		}
	}

	// 4. DiseaseEntity spawned at coordinates
	diseaseQuery := world.Query(ecs.All(disID, posID))
	diseaseFound := false
	for diseaseQuery.Next() {
		dPos := (*components.Position)(diseaseQuery.Get(posID))
		if dPos.X == 10.0 && dPos.Y == 10.0 {
			diseaseFound = true
			dis := (*components.DiseaseEntity)(diseaseQuery.Get(disID))
			if dis.ID != 999 || dis.Lethality != 80 {
				t.Errorf("Expected Disease ID 999 and Lethality 80, got ID %d Lethality %d", dis.ID, dis.Lethality)
			}
		}
	}

	if !diseaseFound {
		t.Errorf("Expected a DiseaseEntity to be spawned at 10.0, 10.0")
	}
}
