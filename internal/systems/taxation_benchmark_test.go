package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func BenchmarkTaxationSystem_Update_Evasion(b *testing.B) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// Pre-register components
	ecs.ComponentID[components.CountryComponent](&world)
	ecs.ComponentID[components.CapitalComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.TreasuryComponent](&world)
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.MarketComponent](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.LoyaltyComponent](&world)
	ecs.ComponentID[components.NPC](&world)

	sys := NewTaxationSystem(&world, hookGraph)

	// Create 10 Countries
	for i := 1; i <= 10; i++ {
		e := world.NewEntity(
			ecs.ComponentID[components.CountryComponent](&world),
			ecs.ComponentID[components.CapitalComponent](&world),
			ecs.ComponentID[components.Affiliation](&world),
			ecs.ComponentID[components.TreasuryComponent](&world),
			ecs.ComponentID[components.JurisdictionComponent](&world),
			ecs.ComponentID[components.Identity](&world),
		)
		aff := (*components.Affiliation)(world.Get(e, ecs.ComponentID[components.Affiliation](&world)))
		aff.CountryID = uint32(i)
		aff.CityID = uint32(i)

		jur := (*components.JurisdictionComponent)(world.Get(e, ecs.ComponentID[components.JurisdictionComponent](&world)))
		jur.Corruption = 100 // Maximum corruption to trigger evasion

		ident := (*components.Identity)(world.Get(e, ecs.ComponentID[components.Identity](&world)))
		ident.ID = uint64(i)
	}

	// Create 100 Villages per Country (1000 total)
	for i := 1; i <= 10; i++ {
		for j := 0; j < 100; j++ {
			cityID := uint32(100 + i*100 + j)
			e := world.NewEntity(
				ecs.ComponentID[components.Village](&world),
				ecs.ComponentID[components.Affiliation](&world),
				ecs.ComponentID[components.MarketComponent](&world),
				ecs.ComponentID[components.TreasuryComponent](&world),
				ecs.ComponentID[components.LoyaltyComponent](&world),
			)
			aff := (*components.Affiliation)(world.Get(e, ecs.ComponentID[components.Affiliation](&world)))
			aff.CountryID = uint32(i)
			aff.CityID = cityID

			loy := (*components.LoyaltyComponent)(world.Get(e, ecs.ComponentID[components.LoyaltyComponent](&world)))
			loy.Value = 0 // Minimum loyalty to trigger evasion
		}
	}

	// Create 10 NPCs per Village (10,000 total)
	for i := 1; i <= 10; i++ {
		for j := 0; j < 100; j++ {
			cityID := uint32(100 + i*100 + j)
			for k := 0; k < 10; k++ {
				e := world.NewEntity(
					ecs.ComponentID[components.NPC](&world),
					ecs.ComponentID[components.Identity](&world),
					ecs.ComponentID[components.Affiliation](&world),
				)
				ident := (*components.Identity)(world.Get(e, ecs.ComponentID[components.Identity](&world)))
				ident.ID = uint64(10000 + cityID*10 + uint32(k))
				aff := (*components.Affiliation)(world.Get(e, ecs.ComponentID[components.Affiliation](&world)))
				aff.CityID = cityID
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We need to make sure Update actually runs its logic.
		// It runs every 100 ticks.
		sys.tickStamp = 99
		sys.Update(&world)
	}
}
