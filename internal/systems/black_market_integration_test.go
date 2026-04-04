package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestBlackMarketSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	contraID := ecs.ComponentID[components.ContrabandComponent](&world)
	villID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	// Create Capital with Jurisdiction & Contraband (Iron is illegal)
	capEnt := world.NewEntity(posID, jurID, contraID, affID)
	capPos := (*components.Position)(world.Get(capEnt, posID))
	capPos.X, capPos.Y = 10, 10

	jur := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	jur.RadiusSquared = 100.0

	contra := (*components.ContrabandComponent)(world.Get(capEnt, contraID))
	contra.Contraband = 1 << components.ItemIron // Ban Iron

	// Create Village inside the Jurisdiction
	villEnt := world.NewEntity(posID, villID, marketID, popID, storageID)
	villPos := (*components.Position)(world.Get(villEnt, posID))
	villPos.X, villPos.Y = 12, 12 // Dist Sq = 8.0, inside radius

	pop := (*components.PopulationComponent)(world.Get(villEnt, popID))
	pop.Count = 10 // Setting population to generate demand

	storage := (*components.StorageComponent)(world.Get(villEnt, storageID))
	storage.Iron = 10 // Supply
	storage.Wood = 10 // Unbanned item

	// 1. Run PriceDiscoverySystem (Base Pricing)
	pds := NewPriceDiscoverySystem()
	pds.Update(&world)

	market := (*components.MarketComponent)(world.Get(villEnt, marketID))
	baseIronPrice := market.IronPrice
	baseWoodPrice := market.WoodPrice

	if baseIronPrice <= 0.0 {
		t.Fatalf("PriceDiscoverySystem failed to set base IronPrice, got %f", baseIronPrice)
	}

	// 2. Run BlackMarketSystem (The Spike)
	bms := NewBlackMarketSystem(&world)
	bms.Update(&world)

	// Check if Iron spiked 5x
	expectedIronPrice := baseIronPrice * 5.0
	if market.IronPrice != expectedIronPrice {
		t.Errorf("BlackMarketSystem failed to spike IronPrice. Expected %f, got %f", expectedIronPrice, market.IronPrice)
	}

	// Check if Wood remained normal
	if market.WoodPrice != baseWoodPrice {
		t.Errorf("BlackMarketSystem improperly spiked WoodPrice. Expected %f, got %f", baseWoodPrice, market.WoodPrice)
	}

	// 3. Butterfly Effect Test: JusticeSystem catching Smugglers
	hooks := engine.NewSparseHookGraph()
	justice := NewJusticeSystem(&world, hooks)

	// Create Smuggler (NPC holding Iron)
	smugglerEnt := world.NewEntity(posID, memID, storageID, affID)
	smugglerPos := (*components.Position)(world.Get(smugglerEnt, posID))
	smugglerPos.X, smugglerPos.Y = 15, 15 // Inside Jurisdiction (DistSq 50.0)

	smugglerStore := (*components.StorageComponent)(world.Get(smugglerEnt, storageID))
	smugglerStore.Iron = 5 // Holding contraband

	// Update JusticeSystem
	justice.Update(&world)

	// Smuggler should now have a CrimeMarker for Contraband
	if !world.Has(smugglerEnt, crimeID) {
		t.Errorf("JusticeSystem failed to catch Smuggler carrying Contraband Iron.")
	}
}
