package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 48: The Ecological Rot & Plague Bridge Testing
// Prove the "Butterfly Effect": Mass Hoarding -> Spoilage -> DiseaseEntity Spawn
func TestSpoilagePlagueBridge_Integration(t *testing.T) {
	// Initialize Deterministic RNG to a seed known to hit the 1% chance quickly
	// We'll iterate ticks to give it a good chance anyway, but seeding is required
	engine.InitializeRNG([32]byte{42, 0, 0, 0})

	tm := engine.NewTickManager(60)
	sys := systems.NewSpoilageSystem()
	tm.AddSystem(sys, engine.PhaseResolution)

	world := tm.World

	storageID := ecs.ComponentID[components.StorageComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	diseaseID := ecs.ComponentID[components.DiseaseEntity](world)

	// Create a Hoarding Village with a massive amount of Food
	villageEntity := world.NewEntity(storageID, posID)

	storage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	storage.Food = 50000 // A massive stockpile

	pos := (*components.Position)(world.Get(villageEntity, posID))
	pos.X = 15.5
	pos.Y = 22.3

	plagueSpawned := false
	lethality := uint8(0)

	// Spoilage system checks every 10 ticks.
	// At 50,000 food, 5% is 2500 spoiled, which is > 500, triggering the roll.
	// 2500 is > 2000, so lethality should be 50.
	// We'll run it for up to 1000 ticks (100 checks), which should statistically hit the 1% chance.
	// Or we can just mock the random generator if it was injectable, but here we'll just run it.
	for i := 0; i < 2000; i++ {
		// Keep resetting food so it continues to spoil enough to trigger the check
		storage.Food = 50000

		tm.Tick()

		// Check if any plague entities spawned
		filter := ecs.All(diseaseID, posID)
		query := world.Query(filter)

		for query.Next() {
			d := (*components.DiseaseEntity)(query.Get(diseaseID))
			p := (*components.Position)(query.Get(posID))

			if p.X == 15.5 && p.Y == 22.3 {
				plagueSpawned = true
				lethality = d.Lethality
			}
		}

		if plagueSpawned {
			break
		}
	}

	if !plagueSpawned {
		t.Fatalf("Expected massive hoarding to spawn a DiseaseEntity, but none spawned after 2000 ticks")
	}

	if lethality != 50 {
		t.Errorf("Expected massive rot (>2000) to spawn plague with lethality 50, got %d", lethality)
	}
}
