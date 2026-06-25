package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: secondary panels — build bar, ambitions, character sheet,
// event log, pause menu.

// DrawBuildBar shows the structure picker strip while in build mode.
func (s *StatePlaying) DrawBuildBar(screen *ebiten.Image) {
	pc := s.PC
	if pc.Mode != ModeBuild {
		return
	}
	sw := screen.Bounds().Dx()
	w := 720
	x := (sw - w) / 2
	y := 8
	DrawPanel(screen, x, y, w, 40)
	DrawText(screen, "BUILD MODE — click to place, [R] cycle, [Esc] cancel", x+10, y+4, AccentCol)

	types := []uint8{
		components.StructureHouse, components.StructureWorkshop, components.StructureStorehouse,
		components.StructureShrine, components.StructureFarm, components.StructureTavern,
	}
	bx := x + 10
	for _, t := range types {
		wood, stone := systems.BuildCosts(t)
		label := fmt.Sprintf("%s (%dW/%dS)", StructureName(uint32(t)), wood, stone)
		btnW := MeasureText(label) + 16
		sel := pc.BuildType == t
		if Button(screen, label, bx, y+20, btnW, 16) {
			pc.BuildType = t
		}
		if sel {
			DrawText(screen, ">", bx-8, y+20, AccentCol)
		}
		bx += btnW + 6
	}
}

// DrawAmbitions shows offered and active sandbox goals.
func (s *StatePlaying) DrawAmbitions(screen *ebiten.Image) {
	pc := s.PC
	if !pc.AmbitionsOpen {
		return
	}
	world := s.Status.TM.World
	player, ok := pc.PossessedEntity()
	if !ok {
		pc.AmbitionsOpen = false
		return
	}
	ambID := ecs.ComponentID[components.AmbitionsComponent](world)
	if !world.Has(player, ambID) {
		world.Add(player, ambID)
	}
	amb := (*components.AmbitionsComponent)(world.Get(player, ambID))

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 420, 360
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)
	DrawText(screen, "AMBITIONS", x+10, y+8, AccentCol)
	if Button(screen, "Close", x+w-70, y+6, 60, 18) {
		pc.AmbitionsOpen = false
	}

	cy := y + 34
	DrawText(screen, "Active:", x+10, cy, TextCol)
	cy += 18
	if len(amb.Ambitions) == 0 {
		DrawText(screen, "  none yet", x+10, cy, TextDim)
		cy += 18
	}
	for i := range amb.Ambitions {
		a := &amb.Ambitions[i]
		label := ambitionLabel(a)
		if a.Done {
			DrawText(screen, "[done] "+label, x+10, cy, BarGreen)
		} else {
			DrawText(screen, label, x+10, cy, TextCol)
			if a.Goal > 0 {
				Bar(screen, x+220, cy, 180, 12, float32(a.Progress)/float32(a.Goal), BarBlue, fmt.Sprintf("%d/%d", a.Progress, a.Goal))
			}
		}
		cy += 18
	}

	cy += 10
	DrawText(screen, "Offers:", x+10, cy, TextCol)
	cy += 18
	for i := range amb.Offers {
		a := &amb.Offers[i]
		DrawText(screen, ambitionLabel(a), x+10, cy, TextDim)
		if Button(screen, "Accept", x+250, cy-2, 70, 16) {
			systems.AcceptOffer(amb, i)
			break
		}
		if Button(screen, "Dismiss", x+324, cy-2, 76, 16) {
			systems.DismissOffer(amb, i)
			break
		}
		cy += 20
	}
	if len(amb.Offers) == 0 {
		DrawText(screen, "  (new offers arrive as you play)", x+10, cy, TextDim)
	}
}

func ambitionLabel(a *components.Ambition) string {
	switch a.Type {
	case components.AmbitionRuler:
		return fmt.Sprintf("Rule City %d", a.TargetID)
	case components.AmbitionWealth:
		return fmt.Sprintf("Amass %d gold", a.Goal)
	case components.AmbitionBuilder:
		return fmt.Sprintf("Build %d structures", a.Goal)
	case components.AmbitionFeud:
		return fmt.Sprintf("Outlive rival #%d", a.TargetID)
	case components.AmbitionHeir:
		return "Raise an heir"
	}
	return "Unknown ambition"
}

// DrawCharPanel shows the player's own character sheet.
func (s *StatePlaying) DrawCharPanel(screen *ebiten.Image) {
	pc := s.PC
	if !pc.CharOpen {
		return
	}
	world := s.Status.TM.World
	player, ok := pc.PossessedEntity()
	if !ok {
		pc.CharOpen = false
		return
	}

	sw := screen.Bounds().Dx()
	w, h := 300, 340
	x, y := (sw-w)/2, 80
	DrawPanel(screen, x, y, w, h)
	DrawText(screen, "CHARACTER", x+10, y+8, AccentCol)
	if Button(screen, "Close", x+w-70, y+6, 60, 18) {
		pc.CharOpen = false
	}
	cy := y + 32
	put := func(t string) {
		DrawText(screen, t, x+12, cy, TextCol)
		cy += 16
	}
	dim := func(t string) {
		DrawText(screen, t, x+12, cy, TextDim)
		cy += 16
	}

	if id, okk := get[components.Identity](world, player); okk {
		put(fmt.Sprintf("%s, age %d", id.Name, id.Age))
		traits := TraitNames(id.BaseTraits)
		if len(traits) > 0 {
			dim("Traits: " + joinStrings(traits))
		}
	}
	rank, cityID, countryID := systems.GetRank(world, player)
	put("Rank: " + systems.RankName(rank))
	dim(fmt.Sprintf("City %d  Country %d", cityID, countryID))
	if j, okk := get[components.JobComponent](world, player); okk {
		put("Job: " + JobName(j.JobID))
	}
	if l, okk := get[components.Legacy](world, player); okk {
		put(fmt.Sprintf("Prestige: %d", l.Prestige))
		if l.InheritedDebt > 0 {
			dim(fmt.Sprintf("Inherited debt: %d", l.InheritedDebt))
		}
	}
	if g, okk := get[components.GenomeComponent](world, player); okk {
		cy += 6
		Bar(screen, x+12, cy, 180, 12, float32(g.Strength)/255, BarRed, fmt.Sprintf("STR %d", g.Strength))
		cy += 16
		Bar(screen, x+12, cy, 180, 12, float32(g.Health)/255, BarGreen, fmt.Sprintf("HLT %d", g.Health))
		cy += 16
		Bar(screen, x+12, cy, 180, 12, float32(g.Intellect)/255, BarBlue, fmt.Sprintf("INT %d", g.Intellect))
		cy += 16
		Bar(screen, x+12, cy, 180, 12, float32(g.Beauty)/255, BarPurple, fmt.Sprintf("BEA %d", g.Beauty))
		cy += 20
	}
	if eq, okk := get[components.EquipmentComponent](world, player); okk && eq.Equipped {
		put(fmt.Sprintf("Wielding artifact (prestige %d)", eq.Weapon.Prestige))
	}
}

// DrawEventLog shows the rolling history of notifications.
func (s *StatePlaying) DrawEventLog(screen *ebiten.Image) {
	pc := s.PC
	if !pc.EventLogOpen {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 460, 360
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)
	DrawText(screen, "CHRONICLE", x+10, y+8, AccentCol)
	if Button(screen, "Close", x+w-70, y+6, 60, 18) {
		pc.EventLogOpen = false
	}
	cy := y + 32
	start := 0
	if len(pc.Log) > 19 {
		start = len(pc.Log) - 19
	}
	for i := len(pc.Log) - 1; i >= start; i-- {
		n := pc.Log[i]
		DrawText(screen, fmt.Sprintf("t%d  %s", n.Tick, n.Text), x+10, cy, TextCol)
		cy += 17
	}
	if len(pc.Log) == 0 {
		DrawText(screen, "Nothing has happened yet.", x+10, cy, TextDim)
	}
}

// DrawPauseMenu shows the pause/save/load/quit modal.
func (s *StatePlaying) DrawPauseMenu(screen *ebiten.Image) {
	pc := s.PC
	if !pc.PauseOpen {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	w, h := 280, 280
	x, y := (sw-w)/2, (sh-h)/2
	DrawPanel(screen, x, y, w, h)
	DrawText(screen, "PAUSED", x+w/2-20, y+12, AccentCol)

	by := y + 44
	if Button(screen, "Resume", x+40, by, w-80, 28) {
		pc.PauseOpen = false
	}
	by += 36
	if Button(screen, "Save Game", x+40, by, w-80, 28) {
		s.doSave()
	}
	by += 36
	if Button(screen, "Load Game", x+40, by, w-80, 28) {
		s.doLoad()
	}
	by += 36
	DrawText(screen, "WASD move, click attack, right-click menu", x+20, by+4, TextDim)
	DrawText(screen, "B build, G goals, C char, L log, Tab lens", x+20, by+18, TextDim)
	by += 40
	if Button(screen, "Quit to Desktop", x+40, by, w-80, 24) {
		// Signal quit by closing; main loop handles os exit via ebiten.
		quitRequested = true
	}
}

// quitRequested is read by the main loop to terminate the game.
var quitRequested bool

// QuitRequested reports whether the player asked to quit.
func QuitRequested() bool { return quitRequested }

func (s *StatePlaying) doSave() {
	if err := engineSave(s.Status); err != nil {
		s.PC.PushNote("Save failed: "+err.Error(), s.Status.TM.Ticks)
	} else {
		s.PC.PushNote("Game saved.", s.Status.TM.Ticks)
	}
	s.PC.PauseOpen = false
}

func (s *StatePlaying) doLoad() {
	if err := engineLoad(s.Status); err != nil {
		s.PC.PushNote("Load failed: "+err.Error(), s.Status.TM.Ticks)
	} else {
		s.PC.PushNote("Game loaded.", s.Status.TM.Ticks)
		s.PC.Initialized = false // re-bind sprites/bridge against loaded world
	}
	s.PC.PauseOpen = false
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
