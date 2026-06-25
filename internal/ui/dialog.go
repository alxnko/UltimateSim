package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: Talk dialog — social verbs against one NPC, backed by the
// hook graph, memory rings and the secret registry.

// DrawDialog renders the conversation window and handles its actions.
func (s *StatePlaying) DrawDialog(screen *ebiten.Image) {
	pc := s.PC
	if !pc.DialogOpen {
		return
	}
	world := s.Status.TM.World
	if !world.Alive(pc.DialogTarget) {
		pc.DialogOpen = false
		return
	}
	player, okP := pc.PossessedEntity()
	if !okP {
		pc.DialogOpen = false
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 360, 230
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)

	target := pc.DialogTarget
	name := "Stranger"
	var targetID uint64
	if ident, ok := get[components.Identity](world, target); ok {
		name = ident.Name
		targetID = ident.ID
	}
	DrawText(screen, "Talking to "+name, x+10, y+8, AccentCol)

	// Standing: net hooks between player and target.
	if pid, ok := get[components.Identity](world, player); ok && s.Status.HookGraph != nil {
		standing := s.Status.HookGraph.GetHook(targetID, pid.ID) + s.Status.HookGraph.GetHook(pid.ID, targetID)
		mood := "neutral"
		if standing > 2 {
			mood = "friendly"
		} else if standing < -2 {
			mood = "hostile"
		}
		DrawText(screen, fmt.Sprintf("Standing: %+d (%s)", standing, mood), x+10, y+26, TextDim)
	}

	if pc.DialogReply != "" {
		DrawText(screen, pc.DialogReply, x+10, y+50, TextCol)
	}

	tick := s.Status.TM.Ticks
	by := y + 80
	if Button(screen, "Chat", x+10, by, 100, 24) {
		pc.DialogReply = systems.Chat(world, s.Status.HookGraph, player, target, tick)
	}
	if Button(screen, "Gift 10 gold", x+118, by, 110, 24) {
		if err := systems.GiveGift(world, s.Status.HookGraph, player, target, 10); err != nil {
			pc.DialogReply = "You can't afford that."
		} else {
			pc.DialogReply = name + " pockets the coins, grateful."
		}
	}
	if Button(screen, "Share rumor", x+236, by, 110, 24) {
		if systems.ShareRumor(world, player, target) {
			pc.DialogReply = name + " leans in, intrigued by your secret."
		} else {
			pc.DialogReply = "They don't seem to understand... or care."
		}
	}
	by += 30
	if Button(screen, "Threaten", x+10, by, 100, 24) {
		crime := systems.Threaten(world, s.Status.HookGraph, player, target, tick)
		pc.DialogReply = name + " recoils in fear."
		if crime {
			pc.DialogReply += " A guard saw that!"
			pc.PushNote("You committed assault — the law noticed", tick)
		}
	}
	if Button(screen, "Trade goods", x+118, by, 110, 24) {
		// Personal trade routes through the nearest village market.
		if v, ok := s.nearestVillage(); ok {
			pc.TradeOpen = true
			pc.TradeVillage = v
			pc.DialogOpen = false
		} else {
			pc.DialogReply = "No market nearby to price the goods."
		}
	}
	if Button(screen, "Leave", x+236, by, 110, 24) {
		pc.DialogOpen = false
		pc.DialogReply = ""
	}

	// Ruler: shortcut into order mode against this NPC.
	rank, cityID, _ := systems.GetRank(world, player)
	if rank >= systems.RankRuler {
		if a, ok := get[components.Affiliation](world, target); ok && a.CityID == cityID {
			if Button(screen, "Give Order...", x+10, by+30, 100, 24) {
				pc.DialogOpen = false
				s.openOrderMenu(target)
			}
		}
	}
}

// nearestVillage finds the closest village to the player.
func (s *StatePlaying) nearestVillage() (ecs.Entity, bool) {
	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		return ecs.Entity{}, false
	}
	ppos, ok := get[components.Position](world, player)
	if !ok {
		return ecs.Entity{}, false
	}

	villageID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	q := world.Query(ecs.All(villageID, posID))
	var best ecs.Entity
	bestD := float32(900) // within 30 tiles
	found := false
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		dx := pos.X - ppos.X
		dy := pos.Y - ppos.Y
		d := dx*dx + dy*dy
		if d < bestD {
			bestD = d
			best = q.Entity()
			found = true
		}
	}
	return best, found
}
