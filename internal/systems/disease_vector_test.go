package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 10.3: Biological Entropy

func TestDiseaseVectorSystem_Generation(t *testing.T) {
	// engine.InitializeRNG is not used directly in the new system which uses local seeding based on tick,
	// but we'll test for deterministic outputs.

	world1 := ecs.NewWorld()
	grid1 := engine.NewMapGrid(10, 10)
	// Force a high traffic tile
	grid1.TileStates[55].FootTraffic = 1000

	sys1 := NewDiseaseVectorSystem(&world1, grid1)
	sys1.spawnChance = 1.0 // Force spawn

	sys1.Update(&world1)

	diseaseFilter1 := ecs.All(ecs.ComponentID[components.DiseaseEntity](&world1))
	query1 := world1.Query(diseaseFilter1)
	count1 := 0
	var id1 uint32
	for query1.Next() {
		count1++
		d := (*components.DiseaseEntity)(query1.Get(ecs.ComponentID[components.DiseaseEntity](&world1)))
		id1 = d.ID
	}

	if count1 == 0 {
		t.Fatalf("Expected at least one disease to spawn")
	}

	// Deterministic Check
	world2 := ecs.NewWorld()
	grid2 := engine.NewMapGrid(10, 10)
	grid2.TileStates[55].FootTraffic = 1000

	sys2 := NewDiseaseVectorSystem(&world2, grid2)
	sys2.spawnChance = 1.0 // Force spawn

	sys2.Update(&world2)

	diseaseFilter2 := ecs.All(ecs.ComponentID[components.DiseaseEntity](&world2))
	query2 := world2.Query(diseaseFilter2)
	count2 := 0
	var id2 uint32
	for query2.Next() {
		count2++
		d := (*components.DiseaseEntity)(query2.Get(ecs.ComponentID[components.DiseaseEntity](&world2)))
		id2 = d.ID
	}

	if count1 != count2 {
		t.Fatalf("Deterministic check failed: count mismatch %d vs %d", count1, count2)
	}

	if id1 != id2 {
		t.Fatalf("Deterministic check failed: ID mismatch %d vs %d", id1, id2)
	}
}

func TestDiseaseVectorSystem_Lethality(t *testing.T) {
	world := ecs.NewWorld()
	grid := engine.NewMapGrid(10, 10)

	// Spawn a disease manually
	dEnt := world.NewEntity()
	dPosID := ecs.ComponentID[components.Position](&world)
	dID := ecs.ComponentID[components.DiseaseEntity](&world)
	world.Add(dEnt, dPosID, dID)
	dPos := (*components.Position)(world.Get(dEnt, dPosID))
	dPos.X, dPos.Y = 5, 5
	disease := (*components.DiseaseEntity)(world.Get(dEnt, dID))
	disease.ID = 100
	disease.Lethality = 255 // Extreme lethality

	// Target 1: Will die (Low health)
	t1 := world.NewEntity()
	posID := ecs.ComponentID[components.Position](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	world.Add(t1, posID, genID)
	pos1 := (*components.Position)(world.Get(t1, posID))
	pos1.X, pos1.Y = 5, 5
	gen1 := (*components.GenomeComponent)(world.Get(t1, genID))
	gen1.Health = 1

	// Target 2: Will survive due to artificially impossible health to guarantee survival if possible (though 255 lethality means it will fail if prng+health < 255. Let's make lethality lower to test survival).
	disease.Lethality = 100

	t2 := world.NewEntity()
	world.Add(t2, posID, genID)
	pos2 := (*components.Position)(world.Get(t2, posID))
	pos2.X, pos2.Y = 5, 5
	gen2 := (*components.GenomeComponent)(world.Get(t2, genID))
	gen2.Health = 255 // Guaranteed survival: prng.IntN(100) + 255 >= 100

	// Target 3: Immune
	t3 := world.NewEntity()
	immID := ecs.ComponentID[components.ImmunityTag](&world)
	world.Add(t3, posID, genID, immID)
	pos3 := (*components.Position)(world.Get(t3, posID))
	pos3.X, pos3.Y = 5, 5
	gen3 := (*components.GenomeComponent)(world.Get(t3, genID))
	gen3.Health = 1 // Low health, would die if not immune
	imm := (*components.ImmunityTag)(world.Get(t3, immID))
	imm.ImmuneTo = []uint32{100}

	sys := NewDiseaseVectorSystem(&world, grid)
	sys.Update(&world)

	// t1 should be dead
	if world.Alive(t1) {
		t.Fatalf("Entity 1 should have died")
	}

	// t2 should be alive and have immunity
	if !world.Alive(t2) {
		t.Fatalf("Entity 2 should have survived")
	}
	if !world.Has(t2, immID) {
		t.Fatalf("Entity 2 should have gained ImmunityTag")
	}

	// t3 should be alive due to prior immunity
	if !world.Alive(t3) {
		t.Fatalf("Entity 3 should have survived due to prior immunity")
	}
}
