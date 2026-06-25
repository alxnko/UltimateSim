package systems

// Shell Phase: Player Trade & Pickup Actions
// Pure action functions invoked by the player input bridge when the possessed
// entity trades with a village market or picks up physical objects (coins,
// legendary artifacts) from the map. They mutate the ECS world directly —
// always OUTSIDE any open query loop — and return errors instead of panicking
// so the UI layer can surface refusals as feedback text.

import (
	"errors"
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Sentinel errors returned by the trade and pickup actions.
var (
	ErrUnknownItem     = errors.New("unknown trade item")
	ErrNoMarket        = errors.New("village has no market")
	ErrNoStock         = errors.New("village lacks stock")
	ErrNotEnoughWealth = errors.New("not enough wealth")
	ErrNotEnoughGoods  = errors.New("not enough goods to sell")
	ErrNoNeeds         = errors.New("entity has no needs component")
	ErrNothingToPickUp = errors.New("nothing to pick up")
)

// EnsurePlayerStorage adds a zeroed StorageComponent to the entity if missing.
// Must be called BEFORE taking component pointers on the entity: adding a
// component moves the entity between archetype tables, invalidating pointers.
func EnsurePlayerStorage(world *ecs.World, ent ecs.Entity) {
	storID := ecs.ComponentID[components.StorageComponent](world)
	if !world.Has(ent, storID) {
		world.Add(ent, storID) // arche zero-initializes added components
	}
}

// PriceOf maps an ItemWood..ItemFood ID to the matching MarketComponent price
// field. Unknown items price at 0.
func PriceOf(m *components.MarketComponent, item uint8) float32 {
	switch item {
	case components.ItemWood:
		return m.WoodPrice
	case components.ItemStone:
		return m.StonePrice
	case components.ItemIron:
		return m.IronPrice
	case components.ItemFood:
		return m.FoodPrice
	}
	return 0
}

// StorageOf maps an ItemWood..ItemFood ID to a pointer to the matching
// StorageComponent stock field. Unknown items return nil.
func StorageOf(s *components.StorageComponent, item uint8) *uint32 {
	switch item {
	case components.ItemWood:
		return &s.Wood
	case components.ItemStone:
		return &s.Stone
	case components.ItemIron:
		return &s.Iron
	case components.ItemFood:
		return &s.Food
	}
	return nil
}

// TradeBuy purchases qty units of item from the village market.
// Cost = market price * qty paid from the player's Needs.Wealth. Fails if the
// player cannot afford it or the village storage lacks stock. On success goods
// move village -> player and the wealth lands in the village TreasuryComponent
// when present (otherwise it evaporates — untracked village economy).
func TradeBuy(world *ecs.World, player, village ecs.Entity, item uint8, qty uint32) error {
	if qty == 0 {
		return nil // buying nothing is a no-op
	}

	marketID := ecs.ComponentID[components.MarketComponent](world)
	storID := ecs.ComponentID[components.StorageComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	if !world.Has(village, marketID) || !world.Has(village, storID) {
		return ErrNoMarket
	}
	if !world.Has(player, needsID) {
		return ErrNoNeeds
	}

	// Structural mutation first: archetype moves invalidate live pointers.
	EnsurePlayerStorage(world, player)

	market := (*components.MarketComponent)(world.Get(village, marketID))
	vStor := (*components.StorageComponent)(world.Get(village, storID))
	pStor := (*components.StorageComponent)(world.Get(player, storID))
	needs := (*components.Needs)(world.Get(player, needsID))

	vStock := StorageOf(vStor, item)
	pStock := StorageOf(pStor, item)
	if vStock == nil || pStock == nil {
		return ErrUnknownItem
	}

	cost := PriceOf(market, item) * float32(qty)
	if needs.Wealth < cost {
		return ErrNotEnoughWealth
	}
	if *vStock < qty {
		return ErrNoStock
	}

	// Transfer goods and wealth.
	*vStock -= qty
	*pStock += qty
	needs.Wealth -= cost
	if world.Has(village, treasuryID) {
		treasury := (*components.TreasuryComponent)(world.Get(village, treasuryID))
		treasury.Wealth += cost
	}
	return nil
}

// TradeSell sells qty units of item from the player to the village market.
// Fails if the player lacks stock. Goods move player -> village storage (when
// present). The village TreasuryComponent pays the proceeds if it can — the
// payout is clamped to the treasury balance; without a treasury the village
// pays in full (untracked village economy, mirroring TradeBuy evaporation).
func TradeSell(world *ecs.World, player, village ecs.Entity, item uint8, qty uint32) error {
	if qty == 0 {
		return nil // selling nothing is a no-op
	}

	marketID := ecs.ComponentID[components.MarketComponent](world)
	storID := ecs.ComponentID[components.StorageComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	if !world.Has(village, marketID) {
		return ErrNoMarket
	}
	if !world.Has(player, needsID) {
		return ErrNoNeeds
	}

	// Structural mutation first: archetype moves invalidate live pointers.
	EnsurePlayerStorage(world, player)

	market := (*components.MarketComponent)(world.Get(village, marketID))
	pStor := (*components.StorageComponent)(world.Get(player, storID))
	needs := (*components.Needs)(world.Get(player, needsID))

	pStock := StorageOf(pStor, item)
	if pStock == nil {
		return ErrUnknownItem
	}
	if *pStock < qty {
		return ErrNotEnoughGoods
	}

	// Transfer goods player -> village.
	*pStock -= qty
	if world.Has(village, storID) {
		vStor := (*components.StorageComponent)(world.Get(village, storID))
		if vStock := StorageOf(vStor, item); vStock != nil {
			*vStock += qty
		}
	}

	// Payment: clamp to the treasury balance when the village tracks one.
	paid := PriceOf(market, item) * float32(qty)
	if world.Has(village, treasuryID) {
		treasury := (*components.TreasuryComponent)(world.Get(village, treasuryID))
		if treasury.Wealth < paid {
			paid = treasury.Wealth
		}
		if paid < 0 {
			paid = 0
		}
		treasury.Wealth -= paid
	}
	needs.Wealth += paid
	return nil
}

// PickUp resolves the player interacting with a physical object entity.
//   - CoinEntity + CurrencyComponent: the coin value is added to the player's
//     Needs.Wealth, consumed=true.
//   - ItemEntity + LegendComponent: equipped into the player's
//     EquipmentComponent (added if missing) when its Prestige is >= the
//     currently equipped weapon (or nothing is equipped), consumed=true.
//     A lesser-prestige artifact is refused: consumed=false, no error.
//   - Anything else: ErrNothingToPickUp.
//
// PickUp NEVER removes the object entity from the world — the caller is
// responsible for despawning it when consumed=true (deferred removal keeps
// this function safe to call while iterating cached entity lists).
func PickUp(world *ecs.World, player, item ecs.Entity) (string, bool, error) {
	coinID := ecs.ComponentID[components.CoinEntity](world)
	currencyID := ecs.ComponentID[components.CurrencyComponent](world)
	itemID := ecs.ComponentID[components.ItemEntity](world)
	legendID := ecs.ComponentID[components.LegendComponent](world)
	equipID := ecs.ComponentID[components.EquipmentComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)

	// Physical coins -> wealth.
	if world.Has(item, coinID) && world.Has(item, currencyID) {
		if !world.Has(player, needsID) {
			return "", false, ErrNoNeeds
		}
		currency := (*components.CurrencyComponent)(world.Get(item, currencyID))
		needs := (*components.Needs)(world.Get(player, needsID))
		needs.Wealth += currency.Value
		return fmt.Sprintf("Picked up %g coins", currency.Value), true, nil
	}

	// Legendary artifacts -> equipment.
	if world.Has(item, itemID) && world.Has(item, legendID) {
		// Copy the legend by value BEFORE any structural mutation: adding a
		// component to the player can swap-relocate rows and invalidate the
		// item's component pointer.
		legend := *(*components.LegendComponent)(world.Get(item, legendID))
		if !world.Has(player, equipID) {
			world.Add(player, equipID) // zero-initialized: Equipped=false
		}
		equip := (*components.EquipmentComponent)(world.Get(player, equipID))
		if equip.Equipped && legend.Prestige < equip.Weapon.Prestige {
			return "Your current weapon is finer", false, nil
		}
		equip.Weapon = legend
		equip.Equipped = true
		// LegendComponent.NameID is a SecretRegistry ID; the registry is not
		// reachable here, so describe it generically.
		return "Picked up a legendary artifact", true, nil
	}

	return "", false, ErrNothingToPickUp
}
