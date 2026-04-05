package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 53: The Black Market Smuggling Engine
// The "Butterfly Effect" proving BlackMarketSystem organically spikes prices
// of illegal goods, naturally incentivizing JobArtisan NPCs to become Smugglers
// (represented by adopting the base gathering job of the contraband, e.g. JobLumberjack).

func TestBlackMarketSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.MarketComponent](&world)
	ecs.ComponentID[components.StorageComponent](&world)
	ecs.ComponentID[components.PopulationComponent](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.ContrabandComponent](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)

	// Systems
	priceSystem := NewPriceDiscoverySystem()
	blackMarketSystem := NewBlackMarketSystem(&world)
	careerSystem := NewCareerChangeSystem()

	// 1. Create a Capital Jurisdiction with Contraband laws (Wood is illegal)
	capEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
		ecs.ComponentID[components.ContrabandComponent](&world),
	)

	capPos := (*components.Position)(world.Get(capEnt, ecs.ComponentID[components.Position](&world)))
	capPos.X = 50.0
	capPos.Y = 50.0

	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	capJur.RadiusSquared = 1000.0 // Large radius

	capContra := (*components.ContrabandComponent)(world.Get(capEnt, ecs.ComponentID[components.ContrabandComponent](&world)))
	capContra.Contraband = 1 << components.ItemWood // Wood is contraband

	// 2. Create a Village inside the jurisdiction
	vilEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.PopulationComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	vilIdent := (*components.Identity)(world.Get(vilEnt, ecs.ComponentID[components.Identity](&world)))
	vilIdent.ID = 1 // CityID = 1

	vilPos := (*components.Position)(world.Get(vilEnt, ecs.ComponentID[components.Position](&world)))
	vilPos.X = 55.0 // Inside radius
	vilPos.Y = 55.0

	vilPop := (*components.PopulationComponent)(world.Get(vilEnt, ecs.ComponentID[components.PopulationComponent](&world)))
	vilPop.Count = 10

	vilStore := (*components.StorageComponent)(world.Get(vilEnt, ecs.ComponentID[components.StorageComponent](&world)))
	vilStore.Wood = 20 // Base supply
	vilStore.Food = 20 // Added Base supply to prevent Farmer career change

	vilMarket := (*components.MarketComponent)(world.Get(vilEnt, ecs.ComponentID[components.MarketComponent](&world)))

	// 3. Create an Artisan NPC inside the Village
	npcEnt := world.NewEntity(
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)

	npcJob := (*components.JobComponent)(world.Get(npcEnt, ecs.ComponentID[components.JobComponent](&world)))
	npcJob.JobID = components.JobArtisan

	npcAff := (*components.Affiliation)(world.Get(npcEnt, ecs.ComponentID[components.Affiliation](&world)))
	npcAff.CityID = 1 // Bound to Village 1

	// Execution Step 1: Price Discovery
	// Base Price = (Demand / Supply+1) = (10 * 5) / (20+1) = 50 / 21 = ~2.38
	priceSystem.Update(&world)
	baseWoodPrice := vilMarket.WoodPrice

	if baseWoodPrice > 10.0 {
		t.Fatalf("Base WoodPrice was unexpectedly high: %f", baseWoodPrice)
	}

	// Execution Step 2: Career Change
	// Since WoodPrice <= 10.0, the Artisan should retain their job
	careerSystem.Update(&world)
	if npcJob.JobID != components.JobArtisan {
		t.Fatalf("Expected NPC to retain JobArtisan under normal prices, changed to %d", npcJob.JobID)
	}

	// Execution Step 3: Black Market System
	// Applies 5x Risk Premium to Wood because it's contraband in this jurisdiction
	blackMarketSystem.Update()

	expectedPremiumPrice := baseWoodPrice * 5.0
	if vilMarket.WoodPrice != expectedPremiumPrice {
		t.Fatalf("Expected WoodPrice to hit risk premium %f, got %f", expectedPremiumPrice, vilMarket.WoodPrice)
	}

	// The premium price should now be > 10.0
	if vilMarket.WoodPrice <= 10.0 {
		t.Fatalf("Risk premium did not push WoodPrice over the career change threshold: %f", vilMarket.WoodPrice)
	}

	// Execution Step 4: The Butterfly Effect (Career Change due to Risk Premium)
	// The massive artificial price spike caused by the contraband law causes the Artisan
	// to see massive profit margins in Wood, natively dropping their job to become a "Smuggler" (JobLumberjack).
	careerSystem.Update(&world)

	if npcJob.JobID != components.JobLumberjack {
		t.Fatalf("Expected NPC to organically change to JobLumberjack due to contraband risk premium, got %d", npcJob.JobID)
	}
}
