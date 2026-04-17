package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 60 - The Physical Crafting Engine
func TestCraftingSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	craftSystem := NewCraftingSystem(&world)

	// 1. Create Employer (Business)
	employer := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.TreasuryComponent](&world),
	)

	employerIdent := (*components.Identity)(world.Get(employer, ecs.ComponentID[components.Identity](&world)))
	employerIdent.ID = 101

	employerStorage := (*components.StorageComponent)(world.Get(employer, ecs.ComponentID[components.StorageComponent](&world)))
	employerStorage.Iron = 12 // Enough for 2 crafts (5 iron each)

	employerTreasury := (*components.TreasuryComponent)(world.Get(employer, ecs.ComponentID[components.TreasuryComponent](&world)))
	employerTreasury.Wealth = 0.0

	// 2. Create Workbench for Employer
	workbench := world.NewEntity(
		ecs.ComponentID[components.WorkbenchEntity](&world),
		ecs.ComponentID[components.WorkbenchComponent](&world),
	)

	wb := (*components.WorkbenchComponent)(world.Get(workbench, ecs.ComponentID[components.WorkbenchComponent](&world)))
	wb.EmployerID = 101
	wb.X = 10.0
	wb.Y = 10.0

	// 3. Create Artisan NPC at Workbench location
	artisan := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
	)

	pos := (*components.Position)(world.Get(artisan, ecs.ComponentID[components.Position](&world)))
	pos.X = 10.0
	pos.Y = 10.5 // Within 1.0 distance squared: (0^2 + 0.5^2) = 0.25 <= 1.0

	job := (*components.JobComponent)(world.Get(artisan, ecs.ComponentID[components.JobComponent](&world)))
	job.JobID = components.JobArtisan
	job.EmployerID = 101

	// Check state before Update
	if employerStorage.Iron != 12 {
		t.Fatalf("Expected 12 iron before test, got %d", employerStorage.Iron)
	}

	// Update the system for 20 ticks (1 trigger)
	for i := 0; i < 20; i++ {
		craftSystem.Update(&world)
	}

	// Assertions for first craft
	if employerStorage.Iron != 7 {
		t.Fatalf("Expected Iron to drop to 7, got %d", employerStorage.Iron)
	}
	if employerTreasury.Wealth != 50.0 {
		t.Fatalf("Expected Wealth to increase to 50.0, got %f", employerTreasury.Wealth)
	}

	// Update for another 20 ticks (2nd trigger)
	for i := 0; i < 20; i++ {
		craftSystem.Update(&world)
	}

	// Assertions for second craft
	if employerStorage.Iron != 2 {
		t.Fatalf("Expected Iron to drop to 2, got %d", employerStorage.Iron)
	}
	if employerTreasury.Wealth != 100.0 {
		t.Fatalf("Expected Wealth to increase to 100.0, got %f", employerTreasury.Wealth)
	}

	// Update for another 20 ticks (3rd trigger)
	for i := 0; i < 20; i++ {
		craftSystem.Update(&world)
	}

	// Should NOT craft because iron is 2 (< 5)
	if employerStorage.Iron != 2 {
		t.Fatalf("Expected Iron to remain at 2, got %d", employerStorage.Iron)
	}
	if employerTreasury.Wealth != 100.0 {
		t.Fatalf("Expected Wealth to remain at 100.0, got %f", employerTreasury.Wealth)
	}
}
