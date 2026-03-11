package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.4: Birth & Genetics Math Testing
func TestBirthSystem_E2E(t *testing.T) {
	// Initialize Deterministic RNG
	seed := [32]byte{1, 2, 3, 4, 5}
	engine.InitializeRNG(seed)

	world := ecs.NewWorld()

	// Register Components
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	genID := ecs.ComponentID[components.Genetics](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	_ = ecs.ComponentID[components.RuinComponent](&world) // needed for the filter

	sys := NewBirthSystem(&world)

	// Spawn entity with exact conditions
	entity := world.NewEntity(storageID, popID, genID, idID)

	storage := (*components.StorageComponent)(world.Get(entity, storageID))
	storage.Food = 50

	pop := (*components.PopulationComponent)(world.Get(entity, popID))
	pop.Count = 10
	pop.Citizens = make([]components.CitizenData, 0)

	gen := (*components.Genetics)(world.Get(entity, genID))
	gen.Strength = 100
	gen.Beauty = 100
	gen.Health = 100
	gen.Intellect = 100

	id := (*components.Identity)(world.Get(entity, idID))
	id.BaseTraits = 0xFFFFFFFF // All bits set

	// Run system
	sys.Update(&world)

	// Assertions
	if storage.Food != 0 {
		t.Errorf("Expected Food to be 0, got %d", storage.Food)
	}

	if pop.Count != 11 {
		t.Errorf("Expected Population to increment to 11, got %d", pop.Count)
	}

	if len(pop.Citizens) != 1 {
		t.Fatalf("Expected 1 CitizenData to be spawned, got %d", len(pop.Citizens))
	}

	citizen := pop.Citizens[0]

	// Verify Genetics mutation limits (+/- 5 from base 100)
	if citizen.Genetics.Strength < 95 || citizen.Genetics.Strength > 105 {
		t.Errorf("Expected Strength near 100, got %d", citizen.Genetics.Strength)
	}
	if citizen.Genetics.Beauty < 95 || citizen.Genetics.Beauty > 105 {
		t.Errorf("Expected Beauty near 100, got %d", citizen.Genetics.Beauty)
	}

	// Because parents are the same base traits (all 1s), child should also be all 1s
	if citizen.BaseTraits != 0xFFFFFFFF {
		t.Errorf("Expected Traits 0xFFFFFFFF, got %d", citizen.BaseTraits)
	}

	// Add more food and another tick to spawn a second citizen to test citizen pairing logic
	storage.Food = 100
	sys.Update(&world)

	// The logic checks if storage.Food >= 50. In a single update it will only trigger once per entity.
	// So len should be 2.
	if len(pop.Citizens) != 2 {
		t.Errorf("Expected 2 citizens after second update, got %d", len(pop.Citizens))
	}
}

func TestBirthSystem_ClampGenetics(t *testing.T) {
	val := clampGenetics(200, 10) // avg=100, mutation=(10%11)-5 = 5. Output: 105
	if val != 105 {
		t.Errorf("clampGenetics failed, got %d", val)
	}

	valBoundsUnder := clampGenetics(0, -6) // negative mod in Go can be weird but let's test absolute underflow
	// 0/2 = 0. -6%11 = -6. -6-5 = -11. Clamped to 0.
	if valBoundsUnder != 0 {
		t.Errorf("Expected 0 clamp, got %d", valBoundsUnder)
	}

	valBoundsOver := clampGenetics(510, 5) // 510/2 = 255. 5%11 = 5. 5-5 = 0. 255+0 = 255.
	if valBoundsOver != 255 {
		t.Errorf("Expected 255 clamp, got %d", valBoundsOver)
	}
}
