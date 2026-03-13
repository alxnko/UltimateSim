package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.4: Administrative Reach & Friction
// FractureLogicSystem Tests

func TestFractureLogicSystem(t *testing.T) {
	world := ecs.NewWorld()

	sys := NewFractureLogicSystem(&world)

	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	villageID := ecs.ComponentID[components.Village](&world)

	// Create Capital City for Country 1 at (0, 0)
	capital1 := world.NewEntity()
	world.Add(capital1, countryID, capitalID, affilID, posID, villageID)

	affilCap := (*components.Affiliation)(world.Get(capital1, affilID))
	affilCap.CountryID = 1

	posCap := (*components.Position)(world.Get(capital1, posID))
	posCap.X = 0
	posCap.Y = 0

	// Create nearby Sub-City for Country 1 at (10, 10)
	city1 := world.NewEntity()
	world.Add(city1, villageID, affilID, posID)

	affilCity1 := (*components.Affiliation)(world.Get(city1, affilID))
	affilCity1.CountryID = 1

	posCity1 := (*components.Position)(world.Get(city1, posID))
	posCity1.X = 10
	posCity1.Y = 10

	// Create far-away Sub-City for Country 1 at (300, 300)
	city2 := world.NewEntity()
	world.Add(city2, villageID, affilID, posID)

	affilCity2 := (*components.Affiliation)(world.Get(city2, affilID))
	affilCity2.CountryID = 1

	posCity2 := (*components.Position)(world.Get(city2, posID))
	posCity2.X = 300
	posCity2.Y = 300

	// Run ticks until fracture evaluation occurs
	for i := 0; i < FractureLogicTickRate; i++ {
		sys.Update(&world)
	}

	// Verify nearby City 1 remains in the Country
	if affilCity1.CountryID != 1 {
		t.Errorf("Nearby city unilaterally withdrew from the Country, expected CountryID to remain 1")
	}

	// Verify far-away City 2 fractured out of the Country
	if affilCity2.CountryID != 0 {
		t.Errorf("Far-away city failed to unilaterally withdraw from the Country, expected CountryID to be 0")
	}
}

// Test Deterministic execution
func TestFractureLogicSystem_Deterministic(t *testing.T) {
	world1 := ecs.NewWorld()
	sys1 := NewFractureLogicSystem(&world1)

	world2 := ecs.NewWorld()
	sys2 := NewFractureLogicSystem(&world2)

	// Both worlds identical setup
	setupWorld := func(w *ecs.World, sys *FractureLogicSystem) {
		countryID := ecs.ComponentID[components.CountryComponent](w)
		capitalID := ecs.ComponentID[components.CapitalComponent](w)
		affilID := ecs.ComponentID[components.Affiliation](w)
		posID := ecs.ComponentID[components.Position](w)
		villageID := ecs.ComponentID[components.Village](w)

		// Create Capital City for Country 1
		capital1 := w.NewEntity()
		w.Add(capital1, countryID, capitalID, affilID, posID, villageID)

		affilCap := (*components.Affiliation)(w.Get(capital1, affilID))
		affilCap.CountryID = 1

		posCap := (*components.Position)(w.Get(capital1, posID))
		posCap.X = 50
		posCap.Y = 50

		// Create 100 cities with deterministic scatter
		for i := 0; i < 100; i++ {
			c := w.NewEntity()
			w.Add(c, villageID, affilID, posID)

			affil := (*components.Affiliation)(w.Get(c, affilID))
			affil.CountryID = 1

			pos := (*components.Position)(w.Get(c, posID))
			pos.X = float32(i * 3) // Linearly spreading outwards
			pos.Y = float32(i * 3)
		}

		// Run for exactly 1 iteration
		for i := 0; i < FractureLogicTickRate; i++ {
			sys.Update(w)
		}
	}

	setupWorld(&world1, sys1)
	setupWorld(&world2, sys2)

	// Compare active country affiliations
	affilID1 := ecs.ComponentID[components.Affiliation](&world1)
	affilID2 := ecs.ComponentID[components.Affiliation](&world2)
	villageID1 := ecs.ComponentID[components.Village](&world1)
	villageID2 := ecs.ComponentID[components.Village](&world2)
	capitalID1 := ecs.ComponentID[components.CapitalComponent](&world1)
	capitalID2 := ecs.ComponentID[components.CapitalComponent](&world2)

	villageFilter1 := filter.All(villageID1, affilID1).Without(capitalID1)
	q1 := world1.Query(&villageFilter1)
	count1 := 0
	for q1.Next() {
		affil := (*components.Affiliation)(q1.Get(affilID1))
		if affil.CountryID == 1 {
			count1++
		}
	}

	villageFilter2 := filter.All(villageID2, affilID2).Without(capitalID2)
	q2 := world2.Query(&villageFilter2)
	count2 := 0
	for q2.Next() {
		affil := (*components.Affiliation)(q2.Get(affilID2))
		if affil.CountryID == 1 {
			count2++
		}
	}

	if count1 != count2 {
		t.Errorf("Deterministic check failed: World1 sub-cities remaining = %d, World2 sub-cities remaining = %d", count1, count2)
	}

	// Log expected deterministic split (informational test validation)
	t.Logf("Deterministic test split: %d sub-cities retained their CountryID out of 100", count1)
}
