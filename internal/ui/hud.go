package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: bottom HUD bar — vitals, needs, identity, time, speed and tool
// buttons. Reads live component data from the possessed entity every frame.

// DrawHUD renders the bottom bar and processes its button clicks.
func (s *StatePlaying) DrawHUD(screen *ebiten.Image) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	y := sh - HUDHeight
	DrawPanel(screen, 0, y, sw, HUDHeight)

	// Detached-camera banner: the single most common way to get "lost".
	if s.PC.CamFree {
		msg := "Free camera — press F (or move) to return to your character"
		bw := MeasureText(msg) + 24
		DrawPanel(screen, (sw-bw)/2, 34, bw, 22)
		DrawText(screen, msg, (sw-bw)/2+12, 38, AccentCol)
	}

	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		DrawText(screen, "No body possessed", 10, y+8, TextDim)
		return
	}

	// --- Vitals block ---
	bx := 10
	barW, barH := 110, 12
	if world.Has(player, ecs.ComponentID[components.VitalsComponent](world)) {
		v := (*components.VitalsComponent)(world.Get(player, ecs.ComponentID[components.VitalsComponent](world)))
		Bar(screen, bx, y+6, barW, barH, v.Blood/100, BarRed, fmt.Sprintf("Blood %d", int(v.Blood)))
		Bar(screen, bx, y+22, barW, barH, v.Stamina/100, BarGreen, fmt.Sprintf("Stamina %d", int(v.Stamina)))
		Bar(screen, bx, y+38, barW, barH, v.Pain/100, BarPurple, fmt.Sprintf("Pain %d", int(v.Pain)))
	}

	bx += barW + 8
	if world.Has(player, ecs.ComponentID[components.Needs](world)) {
		n := (*components.Needs)(world.Get(player, ecs.ComponentID[components.Needs](world)))
		Bar(screen, bx, y+6, barW, barH, n.Food/100, BarGreen, fmt.Sprintf("Food %d", int(n.Food)))
		Bar(screen, bx, y+22, barW, barH, n.Rest/100, BarBlue, fmt.Sprintf("Rest %d", int(n.Rest)))
		DrawText(screen, fmt.Sprintf("Gold: %d", int(n.Wealth)), bx, y+40, AccentCol)
	}

	bx += barW + 8
	if world.Has(player, ecs.ComponentID[components.SanityComponent](world)) {
		sa := (*components.SanityComponent)(world.Get(player, ecs.ComponentID[components.SanityComponent](world)))
		maxS := sa.MaxStress
		if maxS <= 0 {
			maxS = 100
		}
		Bar(screen, bx, y+6, barW, barH, sa.Stress/maxS, BarPurple, fmt.Sprintf("Stress %d", int(sa.Stress)))
	}
	if world.Has(player, ecs.ComponentID[components.StorageComponent](world)) {
		st := (*components.StorageComponent)(world.Get(player, ecs.ComponentID[components.StorageComponent](world)))
		DrawText(screen, fmt.Sprintf("W:%d S:%d I:%d F:%d", st.Wood, st.Stone, st.Iron, st.Food), bx, y+24, TextDim)
	}

	// --- Identity block ---
	bx += barW + 12
	name, job := "Unknown", "Unemployed"
	if world.Has(player, ecs.ComponentID[components.Identity](world)) {
		id := (*components.Identity)(world.Get(player, ecs.ComponentID[components.Identity](world)))
		name = fmt.Sprintf("%s (%d)", id.Name, id.Age)
	}
	if world.Has(player, ecs.ComponentID[components.JobComponent](world)) {
		job = JobName((*components.JobComponent)(world.Get(player, ecs.ComponentID[components.JobComponent](world))).JobID)
	}
	rank, cityID, countryID := systems.GetRank(world, player)
	rankStr := systems.RankName(rank)
	if cityID != 0 {
		rankStr += " of " + CityName(world, cityID)
	}
	if countryID != 0 && rank == systems.RankSovereign {
		rankStr = fmt.Sprintf("Sovereign of Realm %d", countryID)
	}
	DrawText(screen, name, bx, y+6, TextCol)
	DrawText(screen, job, bx, y+22, TextDim)
	DrawText(screen, rankStr, bx, y+38, AccentCol)

	// --- Time + speed block ---
	tick := s.Status.TM.Ticks
	day := tick/600 + 1
	season := []string{"Spring", "Summer", "Autumn", "Winter"}[(tick/3600)%4]
	year := tick/14400 + 1
	timeStr := fmt.Sprintf("Y%d %s D%d", year, season, day%6+1)
	tx := sw - 640
	DrawText(screen, timeStr, tx, y+6, TextCol)

	pauseLabel := "Pause"
	if s.Status.TM.IsPaused {
		pauseLabel = "PAUSED"
	}
	if Button(screen, pauseLabel, tx, y+24, 64, 18) {
		s.Status.TM.TogglePause()
	}
	// Grand Strategy Phase: speed buttons mirror keys 1-4 (ticks per frame).
	speeds := [4]int{1, 2, 4, 8}
	cur := s.Status.TM.Speed
	if cur < 1 {
		cur = 1
	}
	sx := tx + 68
	for _, sp := range speeds {
		label := fmt.Sprintf("%dx", sp)
		if sp == cur {
			label = ">" + label
		}
		if Button(screen, label, sx, y+24, 34, 18) {
			s.Status.TM.Speed = sp
		}
		sx += 38
	}

	// --- Tool buttons ---
	bx2 := sw - 350
	if Button(screen, "Build (B)", bx2, y+6, 80, 22) {
		s.toggleBuildMode()
	}
	if Button(screen, "Goals (G)", bx2+84, y+6, 80, 22) {
		s.PC.AmbitionsOpen = !s.PC.AmbitionsOpen
	}
	if Button(screen, "Me (C)", bx2+168, y+6, 70, 22) {
		s.PC.CharOpen = !s.PC.CharOpen
	}
	if Button(screen, "Log (L)", bx2+242, y+6, 70, 22) {
		s.PC.EventLogOpen = !s.PC.EventLogOpen
	}
	if systems.CanClaimLeadership(world, s.Status.HookGraph, player) {
		if Button(screen, "Claim Leadership!", bx2, y+34, 154, 22) {
			if err := systems.ClaimLeadership(world, s.Status.HookGraph, player); err == nil {
				s.PC.PushNote("You seized leadership of your city!", tick)
			}
		}
	}
	if Button(screen, "Menu (Esc)", bx2+168, y+34, 144, 22) {
		s.PC.PauseOpen = true
	}
}

// toggleBuildMode flips build mode with a default structure.
func (s *StatePlaying) toggleBuildMode() {
	if s.PC.Mode == ModeBuild {
		s.PC.Mode = ModeNormal
		return
	}
	s.PC.Mode = ModeBuild
	if s.PC.BuildType == 0 {
		s.PC.BuildType = components.StructureHouse
	}
}
