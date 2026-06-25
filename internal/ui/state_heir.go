package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: death → heir succession. On the player's death we present the
// dynasty's living members; choosing one re-possesses them (DeathSystem already
// migrated Legacy/debt/artifacts). With no family, a legacy/reincarnate screen.

func causeName(c uint8) string {
	switch c {
	case engine.DeathCauseBleedOut:
		return "bled out in combat"
	case engine.DeathCauseStarvation:
		return "starved"
	}
	return "died"
}

// openHeirFlow populates and opens the heir-selection or game-over modal.
func (s *StatePlaying) openHeirFlow() {
	pc := s.PC
	world := s.Status.TM.World
	ev := pc.Events
	pc.deathCause = ev.DeathCause
	pc.HeirChoices = systems.FindHeirs(world, ev.DeadFamilyID, ev.DeadID)
	if len(pc.HeirChoices) > 0 {
		pc.HeirOpen = true
	} else {
		pc.GameOverOpen = true
	}
}

// DrawHeir renders the heir-selection modal and the game-over screen.
func (s *StatePlaying) DrawHeir(screen *ebiten.Image) {
	pc := s.PC
	world := s.Status.TM.World
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	if pc.HeirOpen {
		w, h := 420, 320
		x, y := (sw-w)/2, (sh-h)/2
		DrawPanel(screen, x, y, w, h)
		DrawText(screen, fmt.Sprintf("You %s. Choose your heir:", causeName(pc.deathCause)), x+10, y+10, AccentCol)
		cy := y + 36
		for i := range pc.HeirChoices {
			if i >= 10 {
				break
			}
			hr := pc.HeirChoices[i]
			label := fmt.Sprintf("%s, age %d, %s (STR %d INT %d)", hr.Name, hr.Age, JobName(hr.JobID), hr.Strength, hr.Intellect)
			if Button(screen, label, x+10, cy, w-20, 22) {
				if err := systems.PossessEntity(world, hr.Entity); err == nil {
					systems.EnsurePlayerStorage(world, hr.Entity)
					ambID := ecs.ComponentID[components.AmbitionsComponent](world)
					if !world.Has(hr.Entity, ambID) {
						world.Add(hr.Entity, ambID)
					}
					pc.PushNote("Your bloodline endures. You live on as "+hr.Name+".", s.Status.TM.Ticks)
					pc.HeirOpen = false
					pc.HeirChoices = nil
				}
			}
			cy += 26
		}
		return
	}

	if pc.GameOverOpen {
		w, h := 380, 220
		x, y := (sw-w)/2, (sh-h)/2
		DrawPanel(screen, x, y, w, h)
		DrawText(screen, "YOUR DYNASTY HAS ENDED", x+w/2-90, y+16, AccentCol)
		DrawText(screen, fmt.Sprintf("You %s with no living heir.", causeName(pc.deathCause)), x+20, y+46, TextCol)
		DrawText(screen, "The world continues without you.", x+20, y+64, TextDim)

		if Button(screen, "Reincarnate (possess a newcomer)", x+30, y+100, w-60, 28) {
			if cand, found := systems.FindStartCandidate(world); found {
				systems.PossessEntity(world, cand)
				systems.EnsurePlayerStorage(world, cand)
				ambID := ecs.ComponentID[components.AmbitionsComponent](world)
				if !world.Has(cand, ambID) {
					world.Add(cand, ambID)
				}
				pc.PushNote("You are reborn into a new life.", s.Status.TM.Ticks)
				pc.GameOverOpen = false
			} else {
				pc.PushNote("No soul to inhabit — the world is empty.", s.Status.TM.Ticks)
			}
		}
		if Button(screen, "Watch the world (spectate)", x+30, y+136, w-60, 24) {
			pc.GameOverOpen = false
		}
	}
}
