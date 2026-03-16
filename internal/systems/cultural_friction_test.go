package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 33.1: Cultural Friction & Ideological Secession Engine Integration Test
func TestCulturalFrictionSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewCulturalFrictionSystem()
	// Force execution immediately
	sys.tickCounter = 49

	// Required IDs
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	cultID := ecs.ComponentID[components.CultureComponent](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)

	// Create Capital (Language 100, Belief 200)
	capital := world.NewEntity()
	world.Add(capital, capID, affID, cultID, beliefID)

	capAff := (*components.Affiliation)(world.Get(capital, affID))
	capAff.CountryID = 1

	capCult := (*components.CultureComponent)(world.Get(capital, cultID))
	capCult.LanguageID = 100

	capBelief := (*components.BeliefComponent)(world.Get(capital, beliefID))
	capBelief.Beliefs = append(capBelief.Beliefs, components.Belief{BeliefID: 200, Weight: 50})

	// Create Loyal Vassal (Matches Culture perfectly)
	loyalVassal := world.NewEntity()
	world.Add(loyalVassal, villageID, affID, loyaltyID, cultID, beliefID)

	loyalAff := (*components.Affiliation)(world.Get(loyalVassal, affID))
	loyalAff.CountryID = 1

	loyalCult := (*components.CultureComponent)(world.Get(loyalVassal, cultID))
	loyalCult.LanguageID = 100

	loyalBelief := (*components.BeliefComponent)(world.Get(loyalVassal, beliefID))
	loyalBelief.Beliefs = append(loyalBelief.Beliefs, components.Belief{BeliefID: 200, Weight: 50})

	loyalLoyalty := (*components.LoyaltyComponent)(world.Get(loyalVassal, loyaltyID))
	loyalLoyalty.Value = 100

	// Create Rebellious Vassal (Speaks Language 101, believes in 201)
	rebelVassal := world.NewEntity()
	world.Add(rebelVassal, villageID, affID, loyaltyID, cultID, beliefID)

	rebelAff := (*components.Affiliation)(world.Get(rebelVassal, affID))
	rebelAff.CountryID = 1

	rebelCult := (*components.CultureComponent)(world.Get(rebelVassal, cultID))
	rebelCult.LanguageID = 101 // Friction +2

	rebelBelief := (*components.BeliefComponent)(world.Get(rebelVassal, beliefID))
	rebelBelief.Beliefs = append(rebelBelief.Beliefs, components.Belief{BeliefID: 201, Weight: 50}) // Friction +3

	rebelLoyalty := (*components.LoyaltyComponent)(world.Get(rebelVassal, loyaltyID))
	rebelLoyalty.Value = 100

	// Tick the system
	sys.Update(&world)

	// Assertions
	loyalCheck := (*components.LoyaltyComponent)(world.Get(loyalVassal, loyaltyID))
	if loyalCheck.Value != 100 {
		t.Errorf("Expected Loyal Vassal's Loyalty to remain 100, got %d", loyalCheck.Value)
	}

	rebelCheck := (*components.LoyaltyComponent)(world.Get(rebelVassal, loyaltyID))
	if rebelCheck.Value != 95 { // 100 - (2 language friction) - (3 belief friction)
		t.Errorf("Expected Rebel Vassal's Loyalty to drop to 95 due to cultural friction, got %d", rebelCheck.Value)
	}

	// Determinism Test (Ensuring map/flat slices resolve correctly in multiple worlds)
	world2 := ecs.NewWorld()
	sys2 := NewCulturalFrictionSystem()
	sys2.tickCounter = 49

	capID2 := ecs.ComponentID[components.CapitalComponent](&world2)
	affID2 := ecs.ComponentID[components.Affiliation](&world2)
	cultID2 := ecs.ComponentID[components.CultureComponent](&world2)
	beliefID2 := ecs.ComponentID[components.BeliefComponent](&world2)
	villageID2 := ecs.ComponentID[components.Village](&world2)
	loyaltyID2 := ecs.ComponentID[components.LoyaltyComponent](&world2)

	capital2 := world2.NewEntity()
	world2.Add(capital2, capID2, affID2, cultID2, beliefID2)

	capAff2 := (*components.Affiliation)(world2.Get(capital2, affID2))
	capAff2.CountryID = 1

	capCult2 := (*components.CultureComponent)(world2.Get(capital2, cultID2))
	capCult2.LanguageID = 100

	capBelief2 := (*components.BeliefComponent)(world2.Get(capital2, beliefID2))
	capBelief2.Beliefs = append(capBelief2.Beliefs, components.Belief{BeliefID: 200, Weight: 50})

	rebelVassal2 := world2.NewEntity()
	world2.Add(rebelVassal2, villageID2, affID2, loyaltyID2, cultID2, beliefID2)

	rebelAff2 := (*components.Affiliation)(world2.Get(rebelVassal2, affID2))
	rebelAff2.CountryID = 1

	rebelCult2 := (*components.CultureComponent)(world2.Get(rebelVassal2, cultID2))
	rebelCult2.LanguageID = 101

	rebelBelief2 := (*components.BeliefComponent)(world2.Get(rebelVassal2, beliefID2))
	rebelBelief2.Beliefs = append(rebelBelief2.Beliefs, components.Belief{BeliefID: 201, Weight: 50})

	rebelLoyalty2 := (*components.LoyaltyComponent)(world2.Get(rebelVassal2, loyaltyID2))
	rebelLoyalty2.Value = 100

	sys2.Update(&world2)

	rebelCheck2 := (*components.LoyaltyComponent)(world2.Get(rebelVassal2, loyaltyID2))
	if rebelCheck2.Value != rebelCheck.Value {
		t.Errorf("Determinism failed: Rebel Loyalty in World 2 (%d) did not match World 1 (%d)", rebelCheck2.Value, rebelCheck.Value)
	}
}
