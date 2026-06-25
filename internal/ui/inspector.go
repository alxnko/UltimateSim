package ui

import (
	"fmt"
	"strings"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: RimWorld-style entity inspector — right-side panel with tabs
// exposing the simulation's hidden depth for whatever is selected.

// DrawInspector renders the panel for the current selection.
func (s *StatePlaying) DrawInspector(screen *ebiten.Image) {
	pc := s.PC
	if !pc.SelectedValid {
		return
	}
	world := s.Status.TM.World
	if !world.Alive(pc.Selected) {
		pc.ClearSelection()
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	x := sw - InspectorWidth
	h := sh - HUDHeight
	DrawPanel(screen, x, 0, InspectorWidth, h)

	if Button(screen, "x", sw-24, 4, 18, 16) {
		pc.ClearSelection()
		return
	}

	switch pc.SelectedKind {
	case TargetNPC:
		s.drawNPCInspector(screen, x, h)
	case TargetVillage:
		s.drawVillageInspector(screen, x, h)
	case TargetStructure, TargetSite, TargetWorkbench:
		s.drawStructureInspector(screen, x)
	default:
		DrawText(screen, "Object", x+10, 10, TextCol)
	}
}

func get[T any](world *ecs.World, e ecs.Entity) (*T, bool) {
	id := ecs.ComponentID[T](world)
	if !world.Has(e, id) {
		return nil, false
	}
	return (*T)(world.Get(e, id)), true
}

func (s *StatePlaying) drawNPCInspector(screen *ebiten.Image, x, h int) {
	pc := s.PC
	world := s.Status.TM.World
	e := pc.Selected

	tabs := []string{"Bio", "Needs", "Social", "Politics"}
	for i, t := range tabs {
		w := 68
		bx := x + 6 + i*(w+2)
		if i == pc.InspectorTab {
			DrawText(screen, "["+t+"]", bx, 8, AccentCol)
		} else if Button(screen, t, bx, 4, w, 18) {
			pc.InspectorTab = i
		}
	}
	y := 30

	line := func(txt string) {
		DrawText(screen, txt, x+10, y, TextCol)
		y += 16
	}
	dim := func(txt string) {
		DrawText(screen, txt, x+10, y, TextDim)
		y += 16
	}

	ident, _ := get[components.Identity](world, e)
	switch pc.InspectorTab {
	case 0: // Bio
		if ident != nil {
			line(fmt.Sprintf("%s, age %d", ident.Name, ident.Age))
			traits := TraitNames(ident.BaseTraits)
			if len(traits) > 0 {
				dim("Traits: " + strings.Join(traits, ", "))
			} else {
				dim("Traits: none notable")
			}
		}
		if g, ok := get[components.GenomeComponent](world, e); ok {
			Bar(screen, x+10, y, 130, 11, float32(g.Strength)/255, BarRed, fmt.Sprintf("STR %d", g.Strength))
			y += 14
			Bar(screen, x+10, y, 130, 11, float32(g.Health)/255, BarGreen, fmt.Sprintf("HLT %d", g.Health))
			y += 14
			Bar(screen, x+10, y, 130, 11, float32(g.Intellect)/255, BarBlue, fmt.Sprintf("INT %d", g.Intellect))
			y += 14
			Bar(screen, x+10, y, 130, 11, float32(g.Beauty)/255, BarPurple, fmt.Sprintf("BEA %d", g.Beauty))
			y += 18
			dim(fmt.Sprintf("Generation %d", g.Generation))
		}
		if j, ok := get[components.JobComponent](world, e); ok {
			line("Job: " + JobName(j.JobID))
		}
		if l, ok := get[components.Legacy](world, e); ok {
			dim(fmt.Sprintf("Prestige %d | Debt %d", l.Prestige, l.InheritedDebt))
		}
		if eq, ok := get[components.EquipmentComponent](world, e); ok && eq.Equipped {
			line(fmt.Sprintf("Wields artifact (prestige %d)", eq.Weapon.Prestige))
		}
	case 1: // Needs
		if n, ok := get[components.Needs](world, e); ok {
			Bar(screen, x+10, y, 200, 12, n.Food/100, BarGreen, fmt.Sprintf("Food %d", int(n.Food)))
			y += 16
			Bar(screen, x+10, y, 200, 12, n.Rest/100, BarBlue, fmt.Sprintf("Rest %d", int(n.Rest)))
			y += 16
			Bar(screen, x+10, y, 200, 12, n.Safety/100, BarPurple, fmt.Sprintf("Safety %d", int(n.Safety)))
			y += 16
			line(fmt.Sprintf("Wealth: %d", int(n.Wealth)))
		}
		if v, ok := get[components.VitalsComponent](world, e); ok {
			y += 6
			Bar(screen, x+10, y, 200, 12, v.Blood/100, BarRed, fmt.Sprintf("Blood %d", int(v.Blood)))
			y += 16
			Bar(screen, x+10, y, 200, 12, v.Stamina/100, BarGreen, fmt.Sprintf("Stamina %d", int(v.Stamina)))
			y += 16
			Bar(screen, x+10, y, 200, 12, v.Pain/100, BarPurple, fmt.Sprintf("Pain %d", int(v.Pain)))
			y += 16
		}
		if sa, ok := get[components.SanityComponent](world, e); ok {
			maxS := sa.MaxStress
			if maxS <= 0 {
				maxS = 100
			}
			Bar(screen, x+10, y, 200, 12, sa.Stress/maxS, BarPurple, fmt.Sprintf("Stress %d", int(sa.Stress)))
			y += 16
			if sa.BreakState == components.BreakBerserk {
				line("BERSERK!")
			} else if sa.BreakState == components.BreakCatatonic {
				line("Catatonic")
			}
		}
		if d, ok := get[components.DesperationComponent](world, e); ok && d.Level > 0 {
			dim(fmt.Sprintf("Desperation: %d/100", d.Level))
		}
	case 2: // Social
		if mem, ok := get[components.Memory](world, e); ok {
			line("Recent memories:")
			shown := 0
			for i := 0; i < 50 && shown < 8; i++ {
				idx := (int(mem.Head) - 1 - i + 100) % 50
				ev := mem.Events[idx]
				if ev.TickStamp == 0 {
					continue
				}
				dim(fmt.Sprintf("t%d %s -> #%d", ev.TickStamp, InteractionName(ev.InteractionType), ev.TargetID))
				shown++
			}
			if shown == 0 {
				dim("  nothing memorable")
			}
		}
		if sec, ok := get[components.SecretComponent](world, e); ok {
			line(fmt.Sprintf("Knows %d secrets", len(sec.Secrets)))
		}
		if ident != nil && s.Status.HookGraph != nil {
			in := s.Status.HookGraph.GetAllIncomingHooks(ident.ID)
			out := s.Status.HookGraph.GetAllHooks(ident.ID)
			pos, neg := 0, 0
			for _, v := range in {
				if v > 0 {
					pos += v
				} else {
					neg += v
				}
			}
			line(fmt.Sprintf("Influence: +%d / %d", pos, neg))
			dim(fmt.Sprintf("Bonds: %d in, %d out", len(in), len(out)))
		}
		if c, ok := get[components.CultureComponent](world, e); ok {
			dim(fmt.Sprintf("Speaks language %d", c.LanguageID))
		}
	case 3: // Politics
		if a, ok := get[components.Affiliation](world, e); ok {
			line(fmt.Sprintf("Family %d  Clan %d", a.FamilyID, a.ClanID))
			line(fmt.Sprintf("City %d  Country %d", a.CityID, a.CountryID))
		}
		rank, _, _ := systems.GetRank(world, e)
		line("Rank: " + systems.RankName(rank))
		if lg, ok := get[components.LegitimacyComponent](world, e); ok {
			Bar(screen, x+10, y, 200, 12, float32(lg.Score)/100, BarBlue, fmt.Sprintf("Legitimacy %d", lg.Score))
			y += 16
		}
		if cm, ok := get[components.CrimeMarker](world, e); ok {
			line(fmt.Sprintf("WANTED: crime lvl %d, bounty %d", cm.CrimeLevel, cm.Bounty))
		}
		if o, ok := get[components.PlayerOrderComponent](world, e); ok {
			dim(fmt.Sprintf("Under orders (type %d)", o.OrderType))
		}
		if _, ok := get[components.StrikeMarker](world, e); ok {
			dim("On strike")
		}
	}
}

func (s *StatePlaying) drawVillageInspector(screen *ebiten.Image, x, h int) {
	pc := s.PC
	world := s.Status.TM.World
	e := pc.Selected
	y := 30

	line := func(txt string) { DrawText(screen, txt, x+10, y, TextCol); y += 16 }
	dim := func(txt string) { DrawText(screen, txt, x+10, y, TextDim); y += 16 }

	name := "Settlement"
	if ident, ok := get[components.Identity](world, e); ok && ident.Name != "" {
		name = ident.Name
	}
	if _, ok := get[components.CapitalComponent](world, e); ok {
		name = "★ " + name + " (Capital)"
	}
	line(name)
	if a, ok := get[components.Affiliation](world, e); ok {
		dim(fmt.Sprintf("City %d, Country %d", a.CityID, a.CountryID))
	}
	if p, ok := get[components.PopulationComponent](world, e); ok {
		line(fmt.Sprintf("Population: %d", p.Count))
	}
	if d, ok := get[components.DemographicsComponent](world, e); ok {
		dim(fmt.Sprintf("Capacity: %d", d.PeakPopulation))
		if d.LaborCrisisActive {
			line("LABOR CRISIS")
		}
	}
	if st, ok := get[components.StorageComponent](world, e); ok {
		y += 4
		line("Stores:")
		dim(fmt.Sprintf("  Wood %d  Stone %d", st.Wood, st.Stone))
		dim(fmt.Sprintf("  Iron %d  Food %d", st.Iron, st.Food))
	}
	if t, ok := get[components.TreasuryComponent](world, e); ok {
		line(fmt.Sprintf("Treasury: %d", int(t.Wealth)))
	}
	if m, ok := get[components.MarketComponent](world, e); ok {
		y += 4
		line("Market prices:")
		dim(fmt.Sprintf("  Wood %.1f  Stone %.1f", m.WoodPrice, m.StonePrice))
		dim(fmt.Sprintf("  Iron %.1f  Food %.1f", m.IronPrice, m.FoodPrice))
		dim(fmt.Sprintf("  Wages %.1f", m.WageRate))
	}
	if jur, ok := get[components.JurisdictionComponent](world, e); ok {
		y += 4
		line("Laws forbid:")
		laws := []string{}
		for _, it := range []uint8{components.InteractionAssault, components.InteractionTheft, components.InteractionMurder, components.InteractionEsoteric, components.InteractionGossip} {
			if systems.IsIllegal(jur, it) {
				laws = append(laws, InteractionName(it))
			}
		}
		if len(laws) == 0 {
			dim("  nothing (lawless)")
		} else {
			dim("  " + strings.Join(laws, ", "))
		}
		if jur.Corruption > 0 {
			dim(fmt.Sprintf("Corruption: %d", jur.Corruption))
		}
		if jur.Trauma > 0 {
			dim(fmt.Sprintf("Trauma: %d", jur.Trauma))
		}
	}
	if _, ok := get[components.PortComponent](world, e); ok {
		dim("Harbor town")
	}
	if r, ok := get[components.RuinComponent](world, e); ok {
		line(fmt.Sprintf("RUINS of %s (decay %d)", r.FormerName, r.Decay))
	}

	// Ruler action.
	player, okP := pc.PossessedEntity()
	if okP {
		rank, cityID, _ := systems.GetRank(world, player)
		if a, ok := get[components.Affiliation](world, e); ok && rank >= systems.RankRuler && a.CityID == cityID {
			if Button(screen, "Edit Laws", x+10, y+8, 120, 22) {
				pc.LawsOpen = true
				pc.LawsVillage = e
			}
		}
	}
}

func (s *StatePlaying) drawStructureInspector(screen *ebiten.Image, x int) {
	world := s.Status.TM.World
	e := s.PC.Selected
	y := 30
	line := func(txt string) { DrawText(screen, txt, x+10, y, TextCol); y += 16 }
	dim := func(txt string) { DrawText(screen, txt, x+10, y, TextDim); y += 16 }

	if st, ok := get[components.StructureComponent](world, e); ok {
		line(StructureName(st.StructureType))
		dim(fmt.Sprintf("Integrity: %.0f%%", st.Integrity))
		if uint8(st.StructureType) == components.StructureShrine && st.DataA != 0 {
			dim(fmt.Sprintf("Devoted to belief %d", st.DataA))
		}
	}
	if site, ok := get[components.ConstructionSiteComponent](world, e); ok {
		line("Construction: " + StructureName(site.SiteType))
		Bar(screen, x+10, y, 200, 12, float32(site.Progress)/float32(max32(site.MaxProgress, 1)), BarGreen,
			fmt.Sprintf("%d/%d", site.Progress, site.MaxProgress))
		y += 18
		dim(fmt.Sprintf("Wood %d/%d  Stone %d/%d", site.WoodGathered, site.WoodRequired, site.StoneGathered, site.StoneRequired))
		if site.BuilderID != 0 {
			dim(fmt.Sprintf("Builder #%d on the job", site.BuilderID))
		} else {
			dim("No builder assigned")
		}
		dim("Stand close and press E to hammer")
	}
	if _, ok := get[components.WorkbenchComponent](world, e); ok {
		line("Workbench")
		dim("Artisans craft goods here")
	}
}

func max32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}
