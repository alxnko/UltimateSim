package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// newTestCountryCapital builds a country capital entity (CountryComponent +
// CapitalComponent + Affiliation + Treasury + Jurisdiction + Identity) for
// countryID, mirroring the taxation_test.go setup.
func newTestCountryCapital(t *testing.T, world *ecs.World, countryID uint32) ecs.Entity {
	t.Helper()
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	identID := ecs.ComponentID[components.Identity](world)

	capital := world.NewEntity(countryCompID, capitalID, affilID, treasuryID, jurID, identID)
	affil := (*components.Affiliation)(world.Get(capital, affilID))
	affil.CountryID = countryID
	ident := (*components.Identity)(world.Get(capital, identID))
	ident.ID = 900 + uint64(countryID)
	return capital
}

// newTestTaxVillage builds a taxable village (Village + Affiliation + Market +
// Treasury + Loyalty) inside countryID with the given wealth and a price sum
// of 10.0 (2+3+1+4), i.e. a base collection of exactly 10.0 per window.
func newTestTaxVillage(t *testing.T, world *ecs.World, countryID uint32, wealth float32) ecs.Entity {
	t.Helper()
	villageID := ecs.ComponentID[components.Village](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)

	village := world.NewEntity(villageID, affilID, marketID, treasuryID, loyaltyID)
	affil := (*components.Affiliation)(world.Get(village, affilID))
	affil.CountryID = countryID
	market := (*components.MarketComponent)(world.Get(village, marketID))
	market.FoodPrice = 2.0
	market.WoodPrice = 3.0
	market.StonePrice = 1.0
	market.IronPrice = 4.0
	treasury := (*components.TreasuryComponent)(world.Get(village, treasuryID))
	treasury.Wealth = wealth
	loyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	loyalty.Value = 100
	return village
}

// wealthOf re-fetches an entity's treasury wealth (pointers may be
// invalidated by archetype moves, e.g. SetTaxRate adding a component).
func wealthOf(world *ecs.World, ent ecs.Entity) float32 {
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	return (*components.TreasuryComponent)(world.Get(ent, treasuryID)).Wealth
}

func TestSetGetTaxRate(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.NPC](&world)
	newTestCountryCapital(t, &world, 5)

	// Default rate before any policy is set.
	if got := GetTaxRate(&world, 5); got != DefaultTaxRatePercent {
		t.Fatalf("Expected default rate %d, got %d", DefaultTaxRatePercent, got)
	}
	// Unknown countries also fall back to the default.
	if got := GetTaxRate(&world, 99); got != DefaultTaxRatePercent {
		t.Fatalf("Expected default rate for unknown country, got %d", got)
	}

	if err := SetTaxRate(&world, 5, 20); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	if got := GetTaxRate(&world, 5); got != 20 {
		t.Fatalf("Expected rate 20, got %d", got)
	}

	// Clamp above MaxTaxRatePercent.
	if err := SetTaxRate(&world, 5, 200); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	if got := GetTaxRate(&world, 5); got != components.MaxTaxRatePercent {
		t.Fatalf("Expected clamped rate %d, got %d", components.MaxTaxRatePercent, got)
	}

	// Zero is a valid rate.
	if err := SetTaxRate(&world, 5, 0); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	if got := GetTaxRate(&world, 5); got != 0 {
		t.Fatalf("Expected rate 0, got %d", got)
	}

	// No capital for the country -> error.
	if err := SetTaxRate(&world, 99, 10); err != ErrNoSuchCountry {
		t.Fatalf("Expected ErrNoSuchCountry, got %v", err)
	}
}

// TestTaxPolicySystem_RateScalesCollection verifies that changing the tax
// rate measurably changes the treasury delta of a full collection window.
// Base collection per window is exactly 10.0 (price sum) at the default rate.
func TestTaxPolicySystem_RateScalesCollection(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	ecs.ComponentID[components.NPC](&world)

	sys := NewTaxPolicySystem(NewTaxationSystem(&world, hooks))

	capital := newTestCountryCapital(t, &world, 5)
	village := newTestTaxVillage(t, &world, 5, 1000.0)

	runWindow := func() {
		for i := 0; i < 100; i++ {
			sys.Update(&world)
		}
	}

	// Window 1: default rate -> baseline 1.0x collection.
	runWindow()
	if got := wealthOf(&world, capital); got != 10.0 {
		t.Fatalf("Default rate: expected capital 10.0, got %v", got)
	}
	if got := wealthOf(&world, village); got != 990.0 {
		t.Fatalf("Default rate: expected village 990.0, got %v", got)
	}

	// Window 2: rate 20 (2x the default) -> collection doubles.
	if err := SetTaxRate(&world, 5, 20); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	runWindow()
	if got := wealthOf(&world, capital); got != 30.0 {
		t.Fatalf("Rate 20: expected capital 30.0 (+20), got %v", got)
	}
	if got := wealthOf(&world, village); got != 970.0 {
		t.Fatalf("Rate 20: expected village 970.0 (-20), got %v", got)
	}

	// Window 3: rate 5 (0.5x) -> half collection via rebate.
	if err := SetTaxRate(&world, 5, 5); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	runWindow()
	if got := wealthOf(&world, capital); got != 35.0 {
		t.Fatalf("Rate 5: expected capital 35.0 (+5), got %v", got)
	}
	if got := wealthOf(&world, village); got != 965.0 {
		t.Fatalf("Rate 5: expected village 965.0 (-5), got %v", got)
	}

	// Window 4: rate 0 -> full rebate, no net collection.
	if err := SetTaxRate(&world, 5, 0); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	runWindow()
	if got := wealthOf(&world, capital); got != 35.0 {
		t.Fatalf("Rate 0: expected capital unchanged at 35.0, got %v", got)
	}
	if got := wealthOf(&world, village); got != 965.0 {
		t.Fatalf("Rate 0: expected village unchanged at 965.0, got %v", got)
	}
}

// TestTaxPolicySystem_SurplusCappedByVillageWealth verifies a raised rate can
// only take what the village still has after the base collection.
func TestTaxPolicySystem_SurplusCappedByVillageWealth(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	ecs.ComponentID[components.NPC](&world)

	sys := NewTaxPolicySystem(NewTaxationSystem(&world, hooks))

	capital := newTestCountryCapital(t, &world, 5)
	village := newTestTaxVillage(t, &world, 5, 15.0)

	// Rate 50 -> multiplier 5.0 -> desired 50, but the village only holds 15.
	if err := SetTaxRate(&world, 5, 50); err != nil {
		t.Fatalf("SetTaxRate failed: %v", err)
	}
	for i := 0; i < 100; i++ {
		sys.Update(&world)
	}

	if got := wealthOf(&world, village); got != 0.0 {
		t.Fatalf("Expected village drained to 0.0, got %v", got)
	}
	if got := wealthOf(&world, capital); got != 15.0 {
		t.Fatalf("Expected capital 15.0 (everything the village had), got %v", got)
	}
}

// newTestTradeCity builds a full trading city (Village + Identity +
// Affiliation + Storage + Market + Treasury) with the given city and country
// IDs. All prices start at 1.0; adjust per test.
func newTestTradeCity(t *testing.T, world *ecs.World, cityID, countryID uint32) ecs.Entity {
	t.Helper()
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	city := world.NewEntity(villageID, identID, affilID, storageID, marketID, treasuryID)
	ident := (*components.Identity)(world.Get(city, identID))
	ident.ID = uint64(cityID)
	affil := (*components.Affiliation)(world.Get(city, affilID))
	affil.CityID = cityID
	affil.CountryID = countryID
	market := (*components.MarketComponent)(world.Get(city, marketID))
	market.WoodPrice = 1.0
	market.StonePrice = 1.0
	market.IronPrice = 1.0
	market.FoodPrice = 1.0
	return city
}

func TestEstablishTradeRoute_Validation(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.RuinComponent](&world)

	newTestTradeCity(t, &world, 1, 5)
	newTestTradeCity(t, &world, 2, 5)

	if _, err := EstablishTradeRoute(&world, 1, 1); err != ErrSameCity {
		t.Fatalf("Expected ErrSameCity, got %v", err)
	}
	if _, err := EstablishTradeRoute(&world, 1, 77); err != ErrNoSuchCity {
		t.Fatalf("Expected ErrNoSuchCity for unknown city, got %v", err)
	}
	if _, err := EstablishTradeRoute(&world, 0, 1); err != ErrNoSuchCity {
		t.Fatalf("Expected ErrNoSuchCity for city 0, got %v", err)
	}

	if _, err := EstablishTradeRoute(&world, 1, 2); err != nil {
		t.Fatalf("EstablishTradeRoute failed: %v", err)
	}
	// Dedupe in both orders.
	if _, err := EstablishTradeRoute(&world, 1, 2); err != ErrRouteExists {
		t.Fatalf("Expected ErrRouteExists (same order), got %v", err)
	}
	if _, err := EstablishTradeRoute(&world, 2, 1); err != ErrRouteExists {
		t.Fatalf("Expected ErrRouteExists (reversed order), got %v", err)
	}

	routes := ListTradeRoutes(&world)
	if len(routes) != 1 {
		t.Fatalf("Expected exactly 1 route, got %d", len(routes))
	}
	want := TradeRouteInfo{FromCity: 1, ToCity: 2, Volume: DefaultTradeRouteVolume}
	if routes[0] != want {
		t.Fatalf("Expected route %+v, got %+v", want, routes[0])
	}
}

// TestTradeRouteSystem_MovesGoodsTowardPricierCity verifies goods flow along
// the price gradient and that the route pays income to both city treasuries
// plus the sovereign cut to the country treasury.
func TestTradeRouteSystem_MovesGoodsTowardPricierCity(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.RuinComponent](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)

	capital := newTestCountryCapital(t, &world, 5)
	cityA := newTestTradeCity(t, &world, 1, 5)
	cityB := newTestTradeCity(t, &world, 2, 5)

	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)

	// Food is cheap and plentiful in A, expensive and absent in B.
	aStorage := (*components.StorageComponent)(world.Get(cityA, storageID))
	aStorage.Food = 100
	aMarket := (*components.MarketComponent)(world.Get(cityA, marketID))
	aMarket.FoodPrice = 2.0
	bMarket := (*components.MarketComponent)(world.Get(cityB, marketID))
	bMarket.FoodPrice = 6.0

	if _, err := EstablishTradeRoute(&world, 1, 2); err != nil {
		t.Fatalf("EstablishTradeRoute failed: %v", err)
	}

	sys := NewTradeRouteSystem()

	// No transfers before the window closes.
	for i := 0; i < int(TradeRouteInterval)-1; i++ {
		sys.Update(&world)
	}
	if aStorage.Food != 100 {
		t.Fatalf("Expected no transfer before tick %d, A food = %d", TradeRouteInterval, aStorage.Food)
	}

	// Window tick: Volume (10) food units move A -> B (B is pricier).
	sys.Update(&world)

	bStorage := (*components.StorageComponent)(world.Get(cityB, storageID))
	if aStorage.Food != 90 {
		t.Fatalf("Expected A food 90, got %d", aStorage.Food)
	}
	if bStorage.Food != 10 {
		t.Fatalf("Expected B food 10, got %d", bStorage.Food)
	}

	// Profit = (6-2)*10 = 40. Cities earn 40*0.25 = 10 each; the shared
	// country treasury earns 40*0.125 per endpoint = 10 total.
	if got := wealthOf(&world, cityA); got != 10.0 {
		t.Fatalf("Expected city A treasury 10.0, got %v", got)
	}
	if got := wealthOf(&world, cityB); got != 10.0 {
		t.Fatalf("Expected city B treasury 10.0, got %v", got)
	}
	if got := wealthOf(&world, capital); got != 10.0 {
		t.Fatalf("Expected country treasury 10.0, got %v", got)
	}
}

// TestTradeRouteSystem_RouteRemovedWhenCityDies verifies a route whose
// endpoint city was ruined vanishes at the next window without trading.
func TestTradeRouteSystem_RouteRemovedWhenCityDies(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.JurisdictionComponent](&world)

	newTestCountryCapital(t, &world, 5)
	cityA := newTestTradeCity(t, &world, 1, 5)
	cityB := newTestTradeCity(t, &world, 2, 5)

	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	ruinID := ecs.ComponentID[components.RuinComponent](&world)

	aStorage := (*components.StorageComponent)(world.Get(cityA, storageID))
	aStorage.Food = 100
	aMarket := (*components.MarketComponent)(world.Get(cityA, marketID))
	aMarket.FoodPrice = 2.0
	bMarket := (*components.MarketComponent)(world.Get(cityB, marketID))
	bMarket.FoodPrice = 6.0

	if _, err := EstablishTradeRoute(&world, 1, 2); err != nil {
		t.Fatalf("EstablishTradeRoute failed: %v", err)
	}

	// City B dies (ruin transformation path).
	world.Add(cityB, ruinID)

	sys := NewTradeRouteSystem()
	for i := 0; i < int(TradeRouteInterval); i++ {
		sys.Update(&world)
	}

	// Ruined endpoint: nothing traded, route removed.
	aStorage = (*components.StorageComponent)(world.Get(cityA, storageID))
	if aStorage.Food != 100 {
		t.Fatalf("Expected no trade with a dead city, A food = %d", aStorage.Food)
	}
	if routes := ListTradeRoutes(&world); len(routes) != 0 {
		t.Fatalf("Expected route to vanish, still have %d", len(routes))
	}
}

func TestMarketSnapshot_Deterministic(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.RuinComponent](&world)

	// Created out of CityID order on purpose.
	for _, cityID := range []uint32{30, 10, 20} {
		city := newTestTradeCity(t, &world, cityID, 5)
		marketID := ecs.ComponentID[components.MarketComponent](&world)
		market := (*components.MarketComponent)(world.Get(city, marketID))
		market.FoodPrice = float32(cityID) + 0.5
		market.WoodPrice = float32(cityID) + 1.5
		market.StonePrice = float32(cityID) + 2.5
		market.IronPrice = float32(cityID) + 3.5
	}

	snap := MarketSnapshot(&world)
	if len(snap) != 3 {
		t.Fatalf("Expected 3 cities, got %d", len(snap))
	}
	for i, wantID := range []uint32{10, 20, 30} {
		if snap[i].CityID != wantID {
			t.Fatalf("Expected sorted CityID %d at index %d, got %d", wantID, i, snap[i].CityID)
		}
		if snap[i].Food != float32(wantID)+0.5 || snap[i].Wood != float32(wantID)+1.5 ||
			snap[i].Stone != float32(wantID)+2.5 || snap[i].Iron != float32(wantID)+3.5 {
			t.Fatalf("Price mismatch for city %d: %+v", wantID, snap[i])
		}
	}

	// Snapshot is a pure read: calling it again yields identical rows.
	again := MarketSnapshot(&world)
	for i := range snap {
		if snap[i] != again[i] {
			t.Fatalf("Snapshot not deterministic at row %d: %+v vs %+v", i, snap[i], again[i])
		}
	}
}
