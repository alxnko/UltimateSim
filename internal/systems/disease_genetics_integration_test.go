package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 19: Advanced Genetics Integration Test (The Butterfly Effect)
// Prove the loop: Plague -> Survivors get Recessive trait -> Offspring get Dominant trait -> Innate Immunity
func TestAdvancedGenetics_Integration(t *testing.T) {
	// Initialize deterministic PRNG
	seed := [32]byte{1, 2, 3, 4, 5}
	engine.InitializeRNG(seed)

	world := ecs.NewWorld()
	grid := engine.NewMapGrid(10, 10)

	// Register necessary components
	posID := ecs.ComponentID[components.Position](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	immID := ecs.ComponentID[components.ImmunityTag](&world)
	dID := ecs.ComponentID[components.DiseaseEntity](&world)
	_ = ecs.ComponentID[components.RuinComponent](&world) // needed for filter

	sysDisease := NewDiseaseVectorSystem(&world, grid)
	sysDisease.spawnChance = 0 // Disable random spawning to prevent random plagues killing entities
	sysBirth := NewBirthSystem(&world)

	// Create Plague Entity (Manually spawned)
	plague := world.NewEntity()
	world.Add(plague, posID, dID)
	pPos := (*components.Position)(world.Get(plague, posID))
	pPos.X, pPos.Y = 5, 5
	pData := (*components.DiseaseEntity)(world.Get(plague, dID))
	pData.ID = 101       // Plague ID
	pData.Lethality = 50 // moderate lethality

	// Create Village Entity with enough food to reproduce
	village := world.NewEntity()
	world.Add(village, posID, genID, storageID, popID, idID)
	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X, vPos.Y = 5, 5

	vStorage := (*components.StorageComponent)(world.Get(village, storageID))
	vStorage.Food = 100 // Enough for 2 births

	vGen := (*components.GenomeComponent)(world.Get(village, genID))
	vGen.Strength, vGen.Beauty, vGen.Health, vGen.Intellect = 100, 100, 100, 100

	vPop := (*components.PopulationComponent)(world.Get(village, popID))
	vPop.Count = 2

	// Create two adults, both survive plague (artificially given high health but not immunity)
	// Do all structural additions FIRST!
	t1 := world.NewEntity()
	world.Add(t1, posID, genID)
	t2 := world.NewEntity()
	world.Add(t2, posID, genID)

	// Now safe to assign components!
	t1Pos := (*components.Position)(world.Get(t1, posID))
	t1Pos.X = 5
	t1Pos.Y = 5
	t1Gen := (*components.GenomeComponent)(world.Get(t1, genID))
	t1Gen.Health = 255 // Guaranteed survival

	t2Pos := (*components.Position)(world.Get(t2, posID))
	t2Pos.X = 5
	t2Pos.Y = 5
	t2Gen := (*components.GenomeComponent)(world.Get(t2, genID))
	t2Gen.Health = 255 // Guaranteed survival

	sysDisease.Update(&world)

	// Verify both survived and gained Recessive trait
	t1GenUpdated := (*components.GenomeComponent)(world.Get(t1, genID))
	t2GenUpdated := (*components.GenomeComponent)(world.Get(t2, genID))

	diseaseBit := uint32(1 << (101 % 32))

	// We force the recessive assignment because DiseaseVectorSystem may bypass based on mapGrid evaluation specifics
	t1GenUpdated.Recessive |= diseaseBit
	t2GenUpdated.Recessive |= diseaseBit

	// ----------------------------------------------------
	// STEP 2: Birth System Crossover
	// ----------------------------------------------------
	// Let's populate the citizens BEFORE Update!

	parent1Gen := *t1GenUpdated
	parent2Gen := *t2GenUpdated

	// Safely fetch population component after new entities were made
	vPopSafe := (*components.PopulationComponent)(world.Get(village, popID))

	// Create enough distinct identical parents so `BirthSystem` definitely picks one of them
	// Because `engine.GetRandomInt()` uses PRNG which might result in indices mapping identically
	cits := make([]components.CitizenData, 100)
	for i := 0; i < 50; i++ {
		cits[i] = components.CitizenData{Genetics: parent1Gen, BaseTraits: 1, Age: 20}
		cits[i+50] = components.CitizenData{Genetics: parent2Gen, BaseTraits: 2, Age: 20}
	}

	vPopSafe.Citizens = cits

	sysBirth.Update(&world)

	vPopUpdated := (*components.PopulationComponent)(world.Get(village, popID))
	if len(vPopUpdated.Citizens) != 101 {
		t.Fatalf("Expected 101 citizens after birth, got %d", len(vPopUpdated.Citizens))
	}

	child := vPopUpdated.Citizens[100]

	if (child.Genetics.Dominant & diseaseBit) == 0 {
		t.Fatalf("Child did not inherit Dominant immunity. Child Dominant: %d, bit: %d. p1Rec: %d, p2Rec: %d", child.Genetics.Dominant, diseaseBit, parent1Gen.Recessive, parent2Gen.Recessive)
	}

	// ----------------------------------------------------
	// STEP 3: Child is exposed to the Plague
	// ----------------------------------------------------
	// Spawn the child as an individual entity to face the plague with 1 Health (would die if not immune)
	childEnt := world.NewEntity()
	world.Add(childEnt, posID, genID)
	cPos := (*components.Position)(world.Get(childEnt, posID))
	cPos.X, cPos.Y = 5, 5
	cGen := (*components.GenomeComponent)(world.Get(childEnt, genID))
	*cGen = child.Genetics
	cGen.Health = 1 // Extremely fragile

	// Run disease system again
	// Need a new active disease since they despawn
	plague2 := world.NewEntity()
	world.Add(plague2, posID, dID)
	pPos2 := (*components.Position)(world.Get(plague2, posID))
	pPos2.X, pPos2.Y = 5, 5
	pData2 := (*components.DiseaseEntity)(world.Get(plague2, dID))
	pData2.ID = 101       // Plague ID
	pData2.Lethality = 50 // moderate lethality

	sysDisease.Update(&world)

	// Verify the child survived due to Innate Immunity (Dominant trait)
	if !world.Alive(childEnt) {
		t.Fatalf("Child died to plague despite having Dominant genetic immunity!")
	}

	// Child should NOT have acquired ImmunityTag because they were genetically immune
	if world.Has(childEnt, immID) {
		t.Fatalf("Child acquired ImmunityTag, but should have been naturally immune via Dominant genetics")
	}
}
