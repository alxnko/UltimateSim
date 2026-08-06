package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// genesisTestGrid builds a 256x256 grid: grassland with food/wood everywhere,
// an ocean band on the left so coastal scoring has something to find.
func genesisTestGrid() *engine.MapGrid {
	g := engine.NewMapGrid(256, 256)
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			i := y*g.Width + x
			if x < 20 {
				g.Tiles[i].BiomeID = engine.BiomeOcean
			} else {
				g.Tiles[i].BiomeID = engine.BiomeGrassland
				g.Resources[i].FoodValue = 40
				g.Resources[i].WoodValue = 30
			}
		}
	}
	return g
}

func seedGenesisWorld(t *testing.T) (*ecs.World, int, int) {
	t.Helper()
	engine.InitializeRNG([32]byte{7, 7, 7})
	world := ecs.NewWorld()
	cfg := DefaultGenesis()
	cfg.Villages = 6
	cfg.Countries = 2
	cfg.MinSiteSeparation = 40
	v, n := SeedCivilization(&world, genesisTestGrid(), cfg)
	return &world, v, n
}

func TestSeedCivilizationPlantsPoliticalWorld(t *testing.T) {
	world, villages, npcs := seedGenesisWorld(t)
	if villages != 6 {
		t.Fatalf("want 6 villages, got %d", villages)
	}
	if npcs != 6*6*4 {
		t.Fatalf("want %d npcs, got %d", 6*6*4, npcs)
	}

	villageID := ecs.ComponentID[components.Village](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	capID := ecs.ComponentID[components.CapitalComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	jobID := ecs.ComponentID[components.JobComponent](world)

	capitals := 0
	vq := world.Query(ecs.All(villageID, affID))
	for vq.Next() {
		a := (*components.Affiliation)(vq.Get(affID))
		if a.CityID == 0 || a.CountryID == 0 {
			t.Errorf("village missing political affiliation: %+v", *a)
		}
		if vq.Has(capID) {
			capitals++
		}
	}
	if capitals != 2 {
		t.Fatalf("want 2 capitals, got %d", capitals)
	}

	bound, employed, rulers := 0, 0, 0
	nq := world.Query(ecs.All(npcID, affID))
	for nq.Next() {
		a := (*components.Affiliation)(nq.Get(affID))
		if a.CityID != 0 && a.CountryID != 0 {
			bound++
		}
		if nq.Has(jobID) {
			employed++
		}
		if nq.Has(adminID) {
			rulers++
		}
	}
	if bound != npcs {
		t.Fatalf("all %d npcs must be city+country bound, got %d", npcs, bound)
	}
	if employed == 0 {
		t.Fatal("genesis produced zero employed citizens")
	}
	if rulers != villages {
		t.Fatalf("want one ruler per village (%d), got %d", villages, rulers)
	}

	// G4: every village gets a visible town square (houses + civic buildings).
	structID := ecs.ComponentID[components.StructureComponent](world)
	structs := 0
	sq := world.Query(ecs.All(structID))
	for sq.Next() {
		structs++
	}
	if structs < villages*5 {
		t.Fatalf("want at least %d genesis structures, got %d", villages*5, structs)
	}
}

func TestSeedCivilizationDeterministic(t *testing.T) {
	w1, _, _ := seedGenesisWorld(t)
	w2, _, _ := seedGenesisWorld(t)

	collect := func(w *ecs.World) map[uint64][2]float32 {
		idID := ecs.ComponentID[components.Identity](w)
		posID := ecs.ComponentID[components.Position](w)
		npcID := ecs.ComponentID[components.NPC](w)
		out := map[uint64][2]float32{}
		q := w.Query(ecs.All(npcID, idID, posID))
		for q.Next() {
			id := (*components.Identity)(q.Get(idID))
			p := (*components.Position)(q.Get(posID))
			out[id.ID] = [2]float32{p.X, p.Y}
		}
		return out
	}
	a, b := collect(w1), collect(w2)
	if len(a) != len(b) {
		t.Fatalf("npc counts differ: %d vs %d", len(a), len(b))
	}
	for id, pa := range a {
		if pb, ok := b[id]; !ok || pa != pb {
			t.Fatalf("npc %d differs: %v vs %v", id, pa, pb)
		}
	}
}
