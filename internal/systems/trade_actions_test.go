package systems

import (
	"errors"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: Player Trade & Pickup Actions E2E tests.
// Deterministic and headless: go test ./internal/systems/ -run TestTradeActions -count=2

// newTradeWorld builds a player (Needs) and a village (Market+Storage+Treasury).
func newTradeWorld(t *testing.T) (*ecs.World, ecs.Entity, ecs.Entity) {
	t.Helper()
	world := ecs.NewWorld()

	needsID := ecs.ComponentID[components.Needs](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)

	player := world.NewEntity(needsID)
	needs := (*components.Needs)(world.Get(player, needsID))
	needs.Wealth = 100

	village := world.NewEntity(villageID, marketID, storID, treasuryID)
	market := (*components.MarketComponent)(world.Get(village, marketID))
	market.WoodPrice = 2
	market.StonePrice = 3
	market.IronPrice = 500
	market.FoodPrice = 1

	vStor := (*components.StorageComponent)(world.Get(village, storID))
	vStor.Wood = 10
	vStor.Iron = 5

	treasury := (*components.TreasuryComponent)(world.Get(village, treasuryID))
	treasury.Wealth = 5

	return &world, player, village
}

func TestTradeActions_BuyAndSell(t *testing.T) {
	world, player, village := newTradeWorld(t)

	needsID := ecs.ComponentID[components.Needs](world)
	storID := ecs.ComponentID[components.StorageComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	// 1. Buy 3 wood at price 2: cost 6.
	if err := TradeBuy(world, player, village, components.ItemWood, 3); err != nil {
		t.Fatalf("TradeBuy failed: %v", err)
	}
	needs := (*components.Needs)(world.Get(player, needsID))
	if needs.Wealth != 94 {
		t.Errorf("Expected player wealth 94 after buy, got %f", needs.Wealth)
	}
	if !world.Has(player, storID) {
		t.Fatalf("Expected EnsurePlayerStorage to add StorageComponent to player")
	}
	pStor := (*components.StorageComponent)(world.Get(player, storID))
	if pStor.Wood != 3 {
		t.Errorf("Expected player wood 3 after buy, got %d", pStor.Wood)
	}
	vStor := (*components.StorageComponent)(world.Get(village, storID))
	if vStor.Wood != 7 {
		t.Errorf("Expected village wood 7 after buy, got %d", vStor.Wood)
	}
	treasury := (*components.TreasuryComponent)(world.Get(village, treasuryID))
	if treasury.Wealth != 11 {
		t.Errorf("Expected village treasury 11 (5+6) after buy, got %f", treasury.Wealth)
	}

	// 2. Buy more stock than the village has (cost 40 is affordable,
	// stock is only 7) -> ErrNoStock, no mutation.
	if err := TradeBuy(world, player, village, components.ItemWood, 20); !errors.Is(err, ErrNoStock) {
		t.Errorf("Expected ErrNoStock, got %v", err)
	}
	// 3. Buy beyond player wealth (iron at 500) -> ErrNotEnoughWealth.
	if err := TradeBuy(world, player, village, components.ItemIron, 1); !errors.Is(err, ErrNotEnoughWealth) {
		t.Errorf("Expected ErrNotEnoughWealth, got %v", err)
	}
	// 4. Unknown item -> ErrUnknownItem.
	if err := TradeBuy(world, player, village, 99, 1); !errors.Is(err, ErrUnknownItem) {
		t.Errorf("Expected ErrUnknownItem, got %v", err)
	}
	// Failed trades must not mutate state.
	needs = (*components.Needs)(world.Get(player, needsID))
	vStor = (*components.StorageComponent)(world.Get(village, storID))
	if needs.Wealth != 94 || vStor.Wood != 7 {
		t.Errorf("Failed trades mutated state: wealth=%f villageWood=%d", needs.Wealth, vStor.Wood)
	}

	// 5. Sell 2 wood back at price 2: proceeds 4, treasury 11 can pay in full.
	if err := TradeSell(world, player, village, components.ItemWood, 2); err != nil {
		t.Fatalf("TradeSell failed: %v", err)
	}
	needs = (*components.Needs)(world.Get(player, needsID))
	if needs.Wealth != 98 {
		t.Errorf("Expected player wealth 98 after sell, got %f", needs.Wealth)
	}
	pStor = (*components.StorageComponent)(world.Get(player, storID))
	if pStor.Wood != 1 {
		t.Errorf("Expected player wood 1 after sell, got %d", pStor.Wood)
	}
	vStor = (*components.StorageComponent)(world.Get(village, storID))
	if vStor.Wood != 9 {
		t.Errorf("Expected village wood 9 after sell, got %d", vStor.Wood)
	}
	treasury = (*components.TreasuryComponent)(world.Get(village, treasuryID))
	if treasury.Wealth != 7 {
		t.Errorf("Expected village treasury 7 (11-4) after sell, got %f", treasury.Wealth)
	}

	// 6. Sell more than the player owns -> ErrNotEnoughGoods.
	if err := TradeSell(world, player, village, components.ItemWood, 50); !errors.Is(err, ErrNotEnoughGoods) {
		t.Errorf("Expected ErrNotEnoughGoods, got %v", err)
	}
	// 7. Unknown item -> ErrUnknownItem.
	if err := TradeSell(world, player, village, 99, 1); !errors.Is(err, ErrUnknownItem) {
		t.Errorf("Expected ErrUnknownItem on sell, got %v", err)
	}
}

func TestTradeActions_TreasuryClamp(t *testing.T) {
	world, player, village := newTradeWorld(t)

	needsID := ecs.ComponentID[components.Needs](world)
	storID := ecs.ComponentID[components.StorageComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	// Give the player goods directly and drain the treasury below the proceeds.
	EnsurePlayerStorage(world, player)
	pStor := (*components.StorageComponent)(world.Get(player, storID))
	pStor.Wood = 4
	treasury := (*components.TreasuryComponent)(world.Get(village, treasuryID))
	treasury.Wealth = 0.5

	// Sell 1 wood at price 2 -> proceeds 2, clamped to 0.5.
	if err := TradeSell(world, player, village, components.ItemWood, 1); err != nil {
		t.Fatalf("TradeSell failed: %v", err)
	}
	needs := (*components.Needs)(world.Get(player, needsID))
	if needs.Wealth != 100.5 {
		t.Errorf("Expected player wealth 100.5 (clamped payout), got %f", needs.Wealth)
	}
	treasury = (*components.TreasuryComponent)(world.Get(village, treasuryID))
	if treasury.Wealth != 0 {
		t.Errorf("Expected village treasury drained to 0, got %f", treasury.Wealth)
	}
	// Goods still transferred despite the clamp.
	pStor = (*components.StorageComponent)(world.Get(player, storID))
	if pStor.Wood != 3 {
		t.Errorf("Expected player wood 3 after clamped sell, got %d", pStor.Wood)
	}
	vStor := (*components.StorageComponent)(world.Get(village, storID))
	if vStor.Wood != 11 {
		t.Errorf("Expected village wood 11 after clamped sell, got %d", vStor.Wood)
	}

	// Empty treasury pays nothing but the trade still succeeds.
	if err := TradeSell(world, player, village, components.ItemWood, 1); err != nil {
		t.Fatalf("TradeSell with empty treasury failed: %v", err)
	}
	needs = (*components.Needs)(world.Get(player, needsID))
	if needs.Wealth != 100.5 {
		t.Errorf("Expected wealth unchanged at 100.5 with empty treasury, got %f", needs.Wealth)
	}
}

func TestTradeActions_PickUpCoin(t *testing.T) {
	world := ecs.NewWorld()

	needsID := ecs.ComponentID[components.Needs](&world)
	coinID := ecs.ComponentID[components.CoinEntity](&world)
	currencyID := ecs.ComponentID[components.CurrencyComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)

	player := world.NewEntity(needsID)
	needs := (*components.Needs)(world.Get(player, needsID))
	needs.Wealth = 10

	coin := world.NewEntity(coinID, currencyID)
	currency := (*components.CurrencyComponent)(world.Get(coin, currencyID))
	currency.Value = 25

	desc, consumed, err := PickUp(&world, player, coin)
	if err != nil {
		t.Fatalf("PickUp coin failed: %v", err)
	}
	if !consumed {
		t.Errorf("Expected coin pickup to be consumed")
	}
	if desc != "Picked up 25 coins" {
		t.Errorf("Expected desc 'Picked up 25 coins', got %q", desc)
	}
	needs = (*components.Needs)(world.Get(player, needsID))
	if needs.Wealth != 35 {
		t.Errorf("Expected player wealth 35 after coin pickup, got %f", needs.Wealth)
	}
	// PickUp must never despawn the entity — the caller does.
	if !world.Alive(coin) {
		t.Errorf("Expected coin entity to remain alive (caller removes it)")
	}

	// Non-pickup entity -> ErrNothingToPickUp.
	rock := world.NewEntity(posID)
	if _, _, err := PickUp(&world, player, rock); !errors.Is(err, ErrNothingToPickUp) {
		t.Errorf("Expected ErrNothingToPickUp, got %v", err)
	}
}

func TestTradeActions_PickUpLegend(t *testing.T) {
	world := ecs.NewWorld()

	needsID := ecs.ComponentID[components.Needs](&world)
	itemID := ecs.ComponentID[components.ItemEntity](&world)
	legendID := ecs.ComponentID[components.LegendComponent](&world)
	equipID := ecs.ComponentID[components.EquipmentComponent](&world)

	player := world.NewEntity(needsID)

	makeLegend := func(prestige uint32) ecs.Entity {
		ent := world.NewEntity(itemID, legendID)
		legend := (*components.LegendComponent)(world.Get(ent, legendID))
		legend.NameID = prestige // SecretRegistry ID; any value works headless
		legend.Prestige = prestige
		return ent
	}

	// 1. First artifact equips even with EquipmentComponent missing.
	first := makeLegend(50)
	desc, consumed, err := PickUp(&world, player, first)
	if err != nil {
		t.Fatalf("PickUp legend failed: %v", err)
	}
	if !consumed {
		t.Errorf("Expected first legend pickup to be consumed")
	}
	if desc != "Picked up a legendary artifact" {
		t.Errorf("Expected legendary artifact desc, got %q", desc)
	}
	if !world.Has(player, equipID) {
		t.Fatalf("Expected EquipmentComponent added to player")
	}
	equip := (*components.EquipmentComponent)(world.Get(player, equipID))
	if !equip.Equipped || equip.Weapon.Prestige != 50 {
		t.Errorf("Expected equipped weapon prestige 50, got equipped=%v prestige=%d", equip.Equipped, equip.Weapon.Prestige)
	}
	if !world.Alive(first) {
		t.Errorf("Expected legend entity to remain alive (caller removes it)")
	}

	// 2. Lesser prestige is refused: not consumed, equipment unchanged.
	lesser := makeLegend(10)
	desc, consumed, err = PickUp(&world, player, lesser)
	if err != nil {
		t.Fatalf("PickUp lesser legend errored: %v", err)
	}
	if consumed {
		t.Errorf("Expected lesser legend to NOT be consumed")
	}
	if desc != "Your current weapon is finer" {
		t.Errorf("Expected refusal desc, got %q", desc)
	}
	equip = (*components.EquipmentComponent)(world.Get(player, equipID))
	if equip.Weapon.Prestige != 50 {
		t.Errorf("Expected equipped prestige to remain 50, got %d", equip.Weapon.Prestige)
	}

	// 3. Equal prestige re-equips (new Prestige >= current).
	equal := makeLegend(50)
	_, consumed, err = PickUp(&world, player, equal)
	if err != nil || !consumed {
		t.Errorf("Expected equal-prestige legend to equip, consumed=%v err=%v", consumed, err)
	}

	// 4. Greater prestige upgrades the weapon.
	greater := makeLegend(120)
	_, consumed, err = PickUp(&world, player, greater)
	if err != nil || !consumed {
		t.Fatalf("Expected greater-prestige legend to equip, consumed=%v err=%v", consumed, err)
	}
	equip = (*components.EquipmentComponent)(world.Get(player, equipID))
	if equip.Weapon.Prestige != 120 {
		t.Errorf("Expected equipped prestige 120 after upgrade, got %d", equip.Weapon.Prestige)
	}
}
