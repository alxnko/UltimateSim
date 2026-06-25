package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// Shell Phase: trade window — buy/sell goods against a village market at its
// live MarketComponent prices. Wealth flows into the village treasury.

// DrawTrade renders the trade window and processes transactions.
func (s *StatePlaying) DrawTrade(screen *ebiten.Image) {
	pc := s.PC
	if !pc.TradeOpen {
		return
	}
	world := s.Status.TM.World
	if !world.Alive(pc.TradeVillage) {
		pc.TradeOpen = false
		return
	}
	player, okP := pc.PossessedEntity()
	if !okP {
		pc.TradeOpen = false
		return
	}
	systems.EnsurePlayerStorage(world, player)

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 430, 240
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)

	name := "Market"
	if ident, ok := get[components.Identity](world, pc.TradeVillage); ok && ident.Name != "" {
		name = ident.Name + " Market"
	}
	DrawText(screen, name, x+10, y+8, AccentCol)

	if n, ok := get[components.Needs](world, player); ok {
		DrawText(screen, fmt.Sprintf("Your gold: %d", int(n.Wealth)), x+250, y+8, AccentCol)
	}

	market, hasMarket := get[components.MarketComponent](world, pc.TradeVillage)
	vStore, hasStore := get[components.StorageComponent](world, pc.TradeVillage)
	pStore, _ := get[components.StorageComponent](world, player)
	if !hasMarket || !hasStore || pStore == nil {
		DrawText(screen, "This settlement has no functioning market.", x+10, y+40, TextDim)
		if Button(screen, "Close", x+w-90, y+h-32, 80, 24) {
			pc.TradeOpen = false
		}
		return
	}

	DrawText(screen, "Good      Price   Town   You", x+10, y+32, TextDim)
	items := []uint8{components.ItemWood, components.ItemStone, components.ItemIron, components.ItemFood}
	ry := y + 52
	for _, item := range items {
		price := systems.PriceOf(market, item)
		town := *systems.StorageOf(vStore, item)
		mine := *systems.StorageOf(pStore, item)
		DrawText(screen, fmt.Sprintf("%-9s %6.1f %6d %6d", ItemName(item), price, town, mine), x+10, ry, TextCol)

		if Button(screen, "Buy 1", x+250, ry-3, 50, 20) {
			s.tradeFeedback(systems.TradeBuy(world, player, pc.TradeVillage, item, 1))
		}
		if Button(screen, "x10", x+304, ry-3, 36, 20) {
			s.tradeFeedback(systems.TradeBuy(world, player, pc.TradeVillage, item, 10))
		}
		if Button(screen, "Sell 1", x+344, ry-3, 50, 20) {
			s.tradeFeedback(systems.TradeSell(world, player, pc.TradeVillage, item, 1))
		}
		ry += 28
	}

	DrawText(screen, "Buying funds the town treasury; selling drains it.", x+10, ry+6, TextDim)
	if Button(screen, "Close", x+w-90, y+h-32, 80, 24) {
		pc.TradeOpen = false
	}
}

func (s *StatePlaying) tradeFeedback(err error) {
	if err != nil {
		s.PC.PushNote("Trade failed: "+err.Error(), s.Status.TM.Ticks)
	}
}
