package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase: character select — the "play as any character" entry
// point. Opens after world warmup on a fresh game; possession can also be
// switched mid-game via the NPC context menu ("Play As").

// openCharacterSelect refreshes the roster and shows the modal.
func (s *StatePlaying) openCharacterSelect() {
	pc := s.PC
	pc.SelectChoices = systems.ListStartCandidates(s.Status.TM.World)
	pc.SelectOpen = true
	pc.SelIndex = 0
}

// handleSelectKeys drives the roster with the keyboard: arrows move, Enter
// possesses, R rolls a surprise. Runs from Update while the modal is open.
func (s *StatePlaying) handleSelectKeys() {
	pc := s.PC
	n := len(pc.SelectChoices)
	if n == 0 {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		pc.SelIndex = (pc.SelIndex + 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		pc.SelIndex = (pc.SelIndex + n - 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.possessAs(pc.SelectChoices[pc.SelIndex].Entity, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		s.possessAs(pc.SelectChoices[int(s.Status.TM.Ticks)%n].Entity, true)
	}
}

// DrawCharacterSelect renders the pick-your-body modal.
func (s *StatePlaying) DrawCharacterSelect(screen *ebiten.Image) {
	pc := s.PC
	if !pc.SelectOpen {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 620, 60+len(pc.SelectChoices)*30+70
	if h > sh-40 {
		h = sh - 40
	}
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)
	DrawText(screen, "CHOOSE WHO YOU WILL BE", x+16, y+10, AccentCol)
	DrawText(screen, "Every soul below is a real, living inhabitant of this world.", x+16, y+28, TextDim)

	cy := y + 52
	for i := range pc.SelectChoices {
		c := &pc.SelectChoices[i]
		if cy+26 > y+h-40 {
			break
		}
		rowCol := TextCol
		if i == pc.SelIndex {
			DrawText(screen, ">", x+6, cy+4, AccentCol)
			rowCol = AccentCol
		}
		label := fmt.Sprintf("%s, %d — %s — %s", c.Name, c.Age, JobName(c.JobID), CityName(s.Status.TM.World, c.CityID))
		DrawText(screen, label, x+16, cy+4, rowCol)
		DrawText(screen, fmt.Sprintf("STR %d  INT %d  %dg", c.Strength, c.Intellect, int(c.Wealth)), x+340, cy+4, TextDim)
		if Button(screen, "Live as", x+w-88, cy, 74, 22) {
			s.possessAs(c.Entity, true)
			return
		}
		cy += 30
	}
	if len(pc.SelectChoices) == 0 {
		DrawText(screen, "The world is still empty... (no eligible souls)", x+16, cy, TextDim)
	}
	if Button(screen, "Surprise Me", x+16, y+h-32, 110, 24) {
		if n := len(pc.SelectChoices); n > 0 {
			idx := int(s.Status.TM.Ticks) % n
			s.possessAs(pc.SelectChoices[idx].Entity, true)
		}
		return
	}
	DrawText(screen, "Arrows+Enter pick, R random. Later: right-click anyone -> Play As.", x+140, y+h-28, TextDim)
}

// possessAs transfers control to target. fresh grants the starter kit and the
// awakening note (start of a run); a mid-game switch inherits the body as-is.
func (s *StatePlaying) possessAs(target ecs.Entity, fresh bool) {
	pc := s.PC
	world := s.Status.TM.World
	if err := systems.PossessEntity(world, target); err != nil {
		pc.PushNote("That body cannot be claimed.", s.Status.TM.Ticks)
		return
	}
	pc.SelectOpen = false
	systems.EnsurePlayerStorage(world, target)

	if fresh {
		storID := ecs.ComponentID[components.StorageComponent](world)
		st := (*components.StorageComponent)(world.Get(target, storID))
		if st.Wood == 0 && st.Stone == 0 {
			st.Wood, st.Stone, st.Food = 120, 120, 50
		}
		needsID := ecs.ComponentID[components.Needs](world)
		if world.Has(target, needsID) {
			n := (*components.Needs)(world.Get(target, needsID))
			if n.Wealth < 100 {
				n.Wealth = 100
			}
		}
	}
	ambID := ecs.ComponentID[components.AmbitionsComponent](world)
	if !world.Has(target, ambID) {
		world.Add(target, ambID)
	}
	if fresh {
		// G5: a fresh life starts with a purpose already on the tracker.
		amb := (*components.AmbitionsComponent)(world.Get(target, ambID))
		if len(amb.Ambitions) == 0 {
			amb.Offers = systems.GenerateOffers(world, s.Status.HookGraph, target, s.Status.TM.Ticks)
			amb = (*components.AmbitionsComponent)(world.Get(target, ambID))
			if len(amb.Offers) > 0 && systems.AcceptOffer(amb, 0) {
				pc.PushNote("Your first ambition is set — press G to see it.", s.Status.TM.Ticks)
			}
		}
		// Grand-strategy pace: default to 2x once you inhabit a body.
		if s.Status.TM.Speed < 2 {
			s.Status.TM.Speed = 2
		}
	}

	// Snap camera to the new body and reset hurt detection.
	posID := ecs.ComponentID[components.Position](world)
	if world.Has(target, posID) {
		pos := (*components.Position)(world.Get(target, posID))
		pc.Cam.X, pc.Cam.Y = float64(pos.X), float64(pos.Y)
	}
	pc.CamFree = false
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	if world.Has(target, vitalsID) {
		pc.lastBlood = (*components.VitalsComponent)(world.Get(target, vitalsID)).Blood
	}

	identID := ecs.ComponentID[components.Identity](world)
	name := "a stranger"
	if world.Has(target, identID) {
		name = (*components.Identity)(world.Get(target, identID)).Name
	}
	if fresh {
		pc.PushNote("You awaken as "+name+". Survive. Rise. Endure.", s.Status.TM.Ticks)
	} else {
		pc.PushNote("You now live as "+name+".", s.Status.TM.Ticks)
	}
}
