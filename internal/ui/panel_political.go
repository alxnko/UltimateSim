package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: ruler tools — laws panel (city jurisdiction + contraband, and
// sovereign-only currency/war controls) and the order status bar.

// DrawLaws renders the law-editing modal for a ruled settlement.
func (s *StatePlaying) DrawLaws(screen *ebiten.Image) {
	pc := s.PC
	if !pc.LawsOpen {
		return
	}
	world := s.Status.TM.World
	if !world.Alive(pc.LawsVillage) {
		pc.LawsOpen = false
		return
	}
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	if !world.Has(pc.LawsVillage, jurID) {
		world.Add(pc.LawsVillage, jurID)
		j := (*components.JurisdictionComponent)(world.Get(pc.LawsVillage, jurID))
		j.RadiusSquared = 400
	}
	jur := (*components.JurisdictionComponent)(world.Get(pc.LawsVillage, jurID))

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 360, 340
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)
	DrawText(screen, "LAWS & DECREES", x+10, y+8, AccentCol)
	if Button(screen, "Close", x+w-70, y+6, 60, 18) {
		pc.LawsOpen = false
	}

	cy := y + 34
	DrawText(screen, "Outlaw acts (click to toggle):", x+10, cy, TextCol)
	cy += 20
	laws := []struct {
		bit  uint8
		name string
	}{
		{components.InteractionAssault, "Assault"},
		{components.InteractionTheft, "Theft"},
		{components.InteractionMurder, "Murder"},
		{components.InteractionEsoteric, "Witchcraft"},
		{components.InteractionGossip, "Gossip"},
	}
	for _, l := range laws {
		if Checkbox(screen, l.name, x+16, cy, systems.IsIllegal(jur, l.bit)) {
			systems.ToggleLaw(jur, l.bit)
		}
		cy += 20
	}

	cy += 6
	DrawText(screen, fmt.Sprintf("Corruption %d   Trauma %d", jur.Corruption, jur.Trauma), x+10, cy, TextDim)
	cy += 22

	// Sovereign-only controls.
	player, ok := pc.PossessedEntity()
	if !ok {
		return
	}
	rank, _, countryID := systems.GetRank(world, player)
	if rank == systems.RankSovereign && countryID != 0 {
		DrawText(screen, "SOVEREIGN POWERS", x+10, cy, AccentCol)
		cy += 20
		if capital, found := systems.FindCapitalOf(world, countryID); found {
			ccID := ecs.ComponentID[components.CountryComponent](world)
			if world.Has(capital, ccID) {
				cc := (*components.CountryComponent)(world.Get(capital, ccID))
				DrawText(screen, fmt.Sprintf("Currency debasement: %.2f", cc.Debasement), x+16, cy, TextCol)
				if Button(screen, "+", x+260, cy-2, 24, 16) {
					systems.SetDebasement(cc, 0.05)
				}
				if Button(screen, "-", x+288, cy-2, 24, 16) {
					systems.SetDebasement(cc, -0.05)
				}
				cy += 22
			}
			// War declarations.
			countries := systems.ListCountries(world)
			DrawText(screen, "Declare war:", x+16, cy, TextCol)
			cy += 18
			shown := 0
			for _, c := range countries {
				if c == countryID || shown >= 3 {
					continue
				}
				if Button(screen, fmt.Sprintf("Country %d", c), x+16, cy, 110, 16) {
					if err := systems.DeclareWar(world, capital, c); err == nil {
						pc.PushNote(fmt.Sprintf("War declared on Country %d!", c), s.Status.TM.Ticks)
					}
				}
				shown++
				cy += 18
			}
			if Button(screen, "Make Peace (all)", x+16, cy, 140, 16) {
				systems.MakePeace(world, capital)
				pc.PushNote("Peace declared.", s.Status.TM.Ticks)
			}
		}
	}
}

// DrawOrderBar shows a hint while an order awaits a target click.
func (s *StatePlaying) DrawOrderBar(screen *ebiten.Image) {
	pc := s.PC
	if pc.Mode != ModeOrder {
		return
	}
	sw := screen.Bounds().Dx()
	w := 360
	x := (sw - w) / 2
	DrawPanel(screen, x, 8, w, 28)
	msg := "Click a destination [Esc cancels]"
	if pc.OrderType == components.OrderAttack {
		msg = "Click an enemy to attack [Esc cancels]"
	}
	DrawText(screen, msg, x+10, 15, AccentCol)
}
