package systems

import (
	"sort"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase: genesis civilization seeding. A grand-strategy start
// needs an already-political world — cities, countries, rulers, jobs — like an
// EU/CK start date, instead of 100 wanderers who might found a village an hour
// in. SeedCivilization runs ONCE from the engine build, after map generation
// and before the first tick. Fully deterministic through the seeded GlobalRNG.

// ID namespaces: keep genesis IDs disjoint from NPCSpawnerSystem (starts at 1)
// and organically-founded villages (inherit NPC identity IDs).
const (
	genesisNPCIDBase    uint64 = 100000
	genesisCityIDBase   uint64 = 900000
	genesisFamilyIDBase uint32 = 1000
)

// GenesisConfig sizes the seeded civilization.
type GenesisConfig struct {
	Villages          int
	Countries         int
	FamiliesPerCity   int
	NPCsPerFamily     int
	MinSiteSeparation float32
	// NameFn names people (id, culture). Nil falls back to "NPC-<id>".
	NameFn func(id uint64, culture uint8) string
}

// DefaultGenesis is the standard 1024x1024 civilization.
func DefaultGenesis() GenesisConfig {
	return GenesisConfig{
		Villages:          12,
		Countries:         4,
		FamiliesPerCity:   6,
		NPCsPerFamily:     4,
		MinSiteSeparation: 80,
	}
}

type genesisSite struct {
	x, y  int
	score int
}

// SeedCivilization plants villages, countries, capitals, rulers and employed,
// city-bound families. Returns the number of villages and NPCs created.
// hooks may be nil (marriage affection hooks are then skipped by Marry's own
// nil handling — see dynasty.go).
func SeedCivilization(world *ecs.World, grid *engine.MapGrid, hooks *engine.SparseHookGraph, cfg GenesisConfig) (int, int) {
	sites := pickGenesisSites(grid, cfg)
	if len(sites) == 0 {
		return 0, 0
	}
	countries := cfg.Countries
	if countries > len(sites) {
		countries = len(sites)
	}
	if countries < 1 {
		countries = 1
	}

	// Component IDs.
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	legID := ecs.ComponentID[components.Legacy](world)
	needsID := ecs.ComponentID[components.Needs](world)
	pathID := ecs.ComponentID[components.Path](world)
	npcID := ecs.ComponentID[components.NPC](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)

	// The first `countries` sites (highest scored, spaced) become capitals.
	// Every other village joins the nearest capital's country.
	countryOf := func(i int) uint32 {
		if i < countries {
			return uint32(i + 1)
		}
		best, bestD := 0, float32(0)
		for c := 0; c < countries; c++ {
			dx := float32(sites[i].x - sites[c].x)
			dy := float32(sites[i].y - sites[c].y)
			d := dx*dx + dy*dy
			if c == 0 || d < bestD {
				best, bestD = c, d
			}
		}
		return uint32(best + 1)
	}

	name := cfg.NameFn
	if name == nil {
		name = Name // deterministic syllable names (names.go)
	}

	nextNPC := genesisNPCIDBase
	nextFam := genesisFamilyIDBase
	npcCount := 0

	jobs := []uint8{
		components.JobFarmer, components.JobLumberjack, components.JobArtisan,
		components.JobGuard, components.JobBuilder, components.JobMiner,
	}

	for i, site := range sites {
		cityID := genesisCityIDBase + uint64(i)
		country := countryOf(i)
		culture := uint8(country)

		// Village entity (mirrors settlement_rule.go recipe + political layer).
		v := world.NewEntity(villageID, posID, storageID, popID, marketID, idID, genID, legID, affID, treasuryID)
		vp := (*components.Position)(world.Get(v, posID))
		vp.X, vp.Y = float32(site.x), float32(site.y)
		vid := (*components.Identity)(world.Get(v, idID))
		vid.ID = cityID
		vid.Name = Name(cityID, culture)
		vid.Age = 0
		st := (*components.StorageComponent)(world.Get(v, storageID))
		st.Wood, st.Food, st.Stone, st.Iron = 300, 500, 150, 50
		pop := (*components.PopulationComponent)(world.Get(v, popID))
		pop.Count = uint32(cfg.FamiliesPerCity * cfg.NPCsPerFamily)
		mk := (*components.MarketComponent)(world.Get(v, marketID))
		mk.FoodPrice, mk.WoodPrice, mk.StonePrice, mk.IronPrice = 1.0, 1.0, 1.0, 1.0
		va := (*components.Affiliation)(world.Get(v, affID))
		va.CityID = uint32(cityID)
		va.CountryID = country
		tr := (*components.TreasuryComponent)(world.Get(v, treasuryID))
		tr.Wealth = 500
		if i < countries {
			// A capital is CapitalComponent + CountryComponent + Affiliation
			// (the convention FindCapitalOf and the macro law systems expect).
			world.Add(v, capitalID)
			world.Add(v, countryCompID)
			ctry := (*components.CountryComponent)(world.Get(v, countryCompID))
			ctry.StandardCurrencyID = country
			ctry.Debasement = 0
		}

		// Families around the village.
		type genesisAdult struct {
			ent    ecs.Entity
			id     uint64
			intel  uint8
			famIdx int
		}
		var adults []genesisAdult
		var bestRuler ecs.Entity
		bestScore := -1
		for f := 0; f < cfg.FamiliesPerCity; f++ {
			famID := nextFam
			nextFam++
			clan := uint32(engine.GetRandomInt() % 100)
			for m := 0; m < cfg.NPCsPerFamily; m++ {
				e := world.NewEntity(posID, velID, idID, genID, legID, needsID, pathID, npcID, slID, affID)
				p := (*components.Position)(world.Get(e, posID))
				jx := site.x + engine.GetRandomInt()%13 - 6
				jy := site.y + engine.GetRandomInt()%13 - 6
				if jx < 0 || jx >= grid.Width || jy < 0 || jy >= grid.Height ||
					grid.GetTile(jx, jy).BiomeID == engine.BiomeOcean {
					jx, jy = site.x, site.y
				}
				p.X, p.Y = float32(jx), float32(jy)

				ident := (*components.Identity)(world.Get(e, idID))
				ident.ID = nextNPC
				nextNPC++
				ident.BaseTraits = uint32(engine.GetRandomInt())
				if m == 0 || engine.GetRandomInt()%4 != 0 {
					ident.Age = uint16(18 + engine.GetRandomInt()%40) // adult
				} else {
					ident.Age = uint16(6 + engine.GetRandomInt()%10) // child
				}
				ident.Name = name(ident.ID, culture)

				g := (*components.GenomeComponent)(world.Get(e, genID))
				g.Strength = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
				g.Beauty = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
				g.Health = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
				g.Intellect = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
				g.Dominant = uint32(engine.GetRandomInt())
				g.Recessive = uint32(engine.GetRandomInt())

				n := (*components.Needs)(world.Get(e, needsID))
				n.Food, n.Rest, n.Safety, n.Wealth = 1000, 100, 100, float32(50+engine.GetRandomInt()%150)

				a := (*components.Affiliation)(world.Get(e, affID))
				a.FamilyID = famID
				a.ClanID = clan
				a.CityID = uint32(cityID)
				a.CountryID = country

				// Copy values BEFORE any structural Add below: world.Add moves
				// the archetype row and invalidates ident/g pointers.
				age := ident.Age
				identityID := ident.ID
				strength, intellect := g.Strength, g.Intellect

				// Adults get trades; the eldest family member stays flexible.
				if age >= 18 && m > 0 {
					world.Add(e, jobID)
					j := (*components.JobComponent)(world.Get(e, jobID))
					j.JobID = jobs[(f+m)%len(jobs)]
				}
				npcCount++

				if age >= 18 {
					adults = append(adults, genesisAdult{ent: e, id: identityID, intel: intellect, famIdx: f})
				}
				if score := int(strength) + int(intellect); age >= 18 && score > bestScore {
					bestScore = score
					bestRuler = e
				}
			}
		}

		// Marry the two eldest-created adults of each family so dynasties
		// exist from tick 1 (creation order is deterministic).
		firstOfFam := map[int]genesisAdult{}
		for _, a := range adults {
			if head, ok := firstOfFam[a.famIdx]; ok {
				if head.ent != (ecs.Entity{}) {
					_ = Marry(world, hooks, head.ent, a.ent)
					firstOfFam[a.famIdx] = genesisAdult{} // family sealed
				}
			} else {
				firstOfFam[a.famIdx] = a
			}
		}

		// Crown the strongest adult as city ruler (capital ruler = sovereign).
		if bestScore >= 0 {
			world.Add(bestRuler, adminID)
			world.Add(bestRuler, legitID)
			lg := (*components.LegitimacyComponent)(world.Get(bestRuler, legitID))
			lg.Score = 50
		}

		// Capitals start with a seated council: the four brightest adults
		// (excluding the ruler) take Steward/Marshal/Diplomat/Spymaster.
		if i < countries {
			sort.SliceStable(adults, func(x, y int) bool {
				if adults[x].intel != adults[y].intel {
					return adults[x].intel > adults[y].intel
				}
				return adults[x].id < adults[y].id
			})
			seat := components.SeatSteward
			for _, a := range adults {
				if seat > components.SeatSpymaster {
					break
				}
				if a.ent == bestRuler {
					continue
				}
				if err := Appoint(world, country, seat, a.id); err == nil {
					seat++
				}
			}
		}

		seedVillageStructures(world, grid, site, uint32(cityID), country)
	}
	return len(sites), npcCount
}

// seedVillageStructures plants a town-square layout so villages LOOK like
// villages from tick 1: a ring of houses plus workshop, shrine, tavern and an
// outlying farm. Offsets are fixed (deterministic); tiles that fall in water
// are skipped.
func seedVillageStructures(world *ecs.World, grid *engine.MapGrid, site genesisSite, cityID, country uint32) {
	structID := ecs.ComponentID[components.StructureComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	benchID := ecs.ComponentID[components.WorkbenchComponent](world)

	place := func(stype uint8, dx, dy int) {
		x, y := site.x+dx, site.y+dy
		if x < 0 || x >= grid.Width || y < 0 || y >= grid.Height ||
			grid.GetTile(x, y).BiomeID == engine.BiomeOcean {
			return
		}
		e := world.NewEntity(structID, posID, affID)
		p := (*components.Position)(world.Get(e, posID))
		p.X, p.Y = float32(x), float32(y)
		st := (*components.StructureComponent)(world.Get(e, structID))
		st.StructureType = uint32(stype)
		st.Integrity = 100
		a := (*components.Affiliation)(world.Get(e, affID))
		a.CityID = cityID
		a.CountryID = country
		if stype == components.StructureWorkshop {
			// Workshops carry a co-located workbench (crafting parity with
			// construction.go completion).
			b := world.NewEntity(benchID, posID)
			bp := (*components.Position)(world.Get(b, posID))
			bp.X, bp.Y = float32(x), float32(y)
		}
	}

	houseOffsets := [5][2]int{{3, 0}, {-3, 0}, {0, 3}, {0, -3}, {3, 3}}
	for _, o := range houseOffsets {
		place(components.StructureHouse, o[0], o[1])
	}
	place(components.StructureWorkshop, -2, 2)
	place(components.StructureShrine, 2, -3)
	place(components.StructureTavern, -3, -2)
	place(components.StructureFarm, 6, 5)
}

// pickGenesisSites scores the map and greedily picks spaced, fertile,
// coast-favoring village sites. Deterministic: fixed scan order + stable sort.
func pickGenesisSites(grid *engine.MapGrid, cfg GenesisConfig) []genesisSite {
	var cands []genesisSite
	step := grid.Width / 128
	if step < 1 {
		step = 1
	}
	for y := step; y < grid.Height-step; y += step {
		for x := step; x < grid.Width-step; x += step {
			t := grid.GetTile(x, y)
			score := 0
			switch t.BiomeID {
			case engine.BiomeGrassland, engine.BiomeTemperateDeciduousForest:
				score = 30
			case engine.BiomeShrubland, engine.BiomeTropicalSeasonalForest:
				score = 20
			case engine.BiomeBeach:
				score = 10
			default:
				continue // ocean, mountains, snow, desert: no genesis city
			}
			r := grid.Resources[y*grid.Width+x]
			score += int(r.FoodValue) + int(r.WoodValue)
			// Coastal bonus: ocean within 5 tiles on an axis.
			for _, d := range [4][2]int{{5, 0}, {-5, 0}, {0, 5}, {0, -5}} {
				if grid.GetTile(x+d[0], y+d[1]).BiomeID == engine.BiomeOcean {
					score += 25
					break
				}
			}
			cands = append(cands, genesisSite{x: x, y: y, score: score})
		}
	}
	// Stable order: score desc, then scan order.
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	minSep := cfg.MinSiteSeparation * cfg.MinSiteSeparation
	var picked []genesisSite
	for _, c := range cands {
		if len(picked) >= cfg.Villages {
			break
		}
		ok := true
		for _, p := range picked {
			dx := float32(c.x - p.x)
			dy := float32(c.y - p.y)
			if dx*dx+dy*dy < minSep {
				ok = false
				break
			}
		}
		if ok {
			picked = append(picked, c)
		}
	}
	return picked
}
