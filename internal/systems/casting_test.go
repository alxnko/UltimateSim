package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 20.2: Abstract Physics Testing
func TestCastingSystem(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)

	// Create MapGrid 10x10
	grid := engine.NewMapGrid(10, 10)

	// Inject 100 mana at index (5, 5) which is 5*10 + 5 = 55
	grid.Mana[55].Value = 100
	// Base temperature is 50
	grid.Tiles[55].Temperature = 50

	castingSystem := NewCastingSystem(&world, grid)

	// Create caster entity at 5,5
	caster := world.NewEntity(posID, jobID)
	pos := (*components.Position)(world.Get(caster, posID))
	pos.X = 5
	pos.Y = 5
	job := (*components.JobComponent)(world.Get(caster, jobID))
	job.JobID = components.JobCaster

	// Create non-caster entity at 4,4
	nonCaster := world.NewEntity(posID, jobID)
	ncPos := (*components.Position)(world.Get(nonCaster, posID))
	ncPos.X = 4
	ncPos.Y = 4
	ncJob := (*components.JobComponent)(world.Get(nonCaster, jobID))
	ncJob.JobID = components.JobFarmer

	// Inject 100 mana at 4,4 as well
	grid.Mana[44].Value = 100
	grid.Tiles[44].Temperature = 50

	// Tick 1
	castingSystem.Update(&world)

	// Verify caster consumed mana and raised temp
	if grid.Mana[55].Value != 50 {
		t.Errorf("Expected caster mana to be 50, got %d", grid.Mana[55].Value)
	}
	if grid.Tiles[55].Temperature != 150 {
		t.Errorf("Expected caster tile temp to be 150, got %d", grid.Tiles[55].Temperature)
	}

	// Verify non-caster did not consume mana
	if grid.Mana[44].Value != 100 {
		t.Errorf("Expected non-caster mana to remain 100, got %d", grid.Mana[44].Value)
	}
	if grid.Tiles[44].Temperature != 50 {
		t.Errorf("Expected non-caster tile temp to remain 50, got %d", grid.Tiles[44].Temperature)
	}

	// Tick 2
	castingSystem.Update(&world)

	// Verify caster consumed mana and raised temp again, temp should cap at 250
	if grid.Mana[55].Value != 0 {
		t.Errorf("Expected caster mana to be 0, got %d", grid.Mana[55].Value)
	}
	if grid.Tiles[55].Temperature != 250 {
		t.Errorf("Expected caster tile temp to be 250, got %d", grid.Tiles[55].Temperature)
	}

	// Tick 3
	castingSystem.Update(&world)

	// Verify no more mana is consumed and temp remains 250
	if grid.Mana[55].Value != 0 {
		t.Errorf("Expected caster mana to stay 0, got %d", grid.Mana[55].Value)
	}
	if grid.Tiles[55].Temperature != 250 {
		t.Errorf("Expected caster tile temp to stay 250, got %d", grid.Tiles[55].Temperature)
	}
}
