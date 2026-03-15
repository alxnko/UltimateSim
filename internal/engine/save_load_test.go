package engine

import (
	"os"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestSaveLoadWorld(t *testing.T) {
	// Setup db file
	dbPath := "test_save.db"
	os.Remove(dbPath)
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Setup world
	world := ecs.NewWorld()
	idID := ecs.ComponentID[components.Identity](&world)
	posID := ecs.ComponentID[components.Position](&world)

	// Add an entity
	ent := world.NewEntity(idID, posID)

	ident := (*components.Identity)(world.Get(ent, idID))
	ident.ID = 42
	ident.Name = "TestEntity"
	ident.BaseTraits = 100
	ident.Age = 35

	pos := (*components.Position)(world.Get(ent, posID))
	pos.X = 10.5
	pos.Y = 20.5

	// Setup TM and Grid wrappers
	tm := &TickManager{World: &world, Ticks: 500}
	grid := NewMapGrid(10, 10)

	// Save
	err = SaveWorld(tm, grid, byte(1), db)
	if err != nil {
		t.Fatalf("Failed to save world: %v", err)
	}

	// Load into a new world
	newWorld := ecs.NewWorld()
	newTm := &TickManager{World: &newWorld}

	newIdID := ecs.ComponentID[components.Identity](&newWorld)
	newPosID := ecs.ComponentID[components.Position](&newWorld)
	_ = newIdID // Ensure registered
	_ = newPosID

	err = LoadWorld(newTm, db)
	if err != nil {
		t.Fatalf("Failed to load world: %v", err)
	}

	if newTm.Ticks != 500 {
		t.Errorf("Expected 500 ticks, got %d", newTm.Ticks)
	}

	// Verify
	query := newWorld.Query(ecs.All(newIdID))
	count := 0
	for query.Next() {
		count++
		newEnt := query.Entity()
		newIdent := (*components.Identity)(newWorld.Get(newEnt, newIdID))

		if newIdent.ID != 42 || newIdent.Name != "TestEntity" || newIdent.BaseTraits != 100 || newIdent.Age != 35 {
			t.Errorf("Loaded identity mismatch: %+v", newIdent)
		}

		if newWorld.Has(newEnt, newPosID) {
			newPos := (*components.Position)(newWorld.Get(newEnt, newPosID))
			if newPos.X != 10.5 || newPos.Y != 20.5 {
				t.Errorf("Loaded position mismatch: %+v", newPos)
			}
		} else {
			t.Errorf("Expected entity to have position component")
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 entity loaded, got %d", count)
	}
}
