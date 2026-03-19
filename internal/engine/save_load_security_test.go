package engine

import (
	"os"
	"testing"

	"github.com/mlange-42/arche/ecs"
)

func TestLoadWorldCorruptJSON(t *testing.T) {
	// Setup db file
	dbPath := "test_corrupt.db"
	os.Remove(dbPath)
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Insert game state
	_, err = db.Exec("INSERT INTO game_state (id, ticks, grid_width, grid_height, seed_val) VALUES (1, 100, 256, 256, 1)")
	if err != nil {
		t.Fatalf("Failed to insert game state: %v", err)
	}

	// Insert valid entity UID
	uid := uint64(123)
	_, err = db.Exec("INSERT INTO entities (uid) VALUES (?)", uid)
	if err != nil {
		t.Fatalf("Failed to insert entity: %v", err)
	}

	// Insert identity
	_, err = db.Exec("INSERT INTO identity (uid, name, basetraits, age) VALUES (?, 'CorruptNode', 0, 20)", uid)
	if err != nil {
		t.Fatalf("Failed to insert identity: %v", err)
	}

	// Insert CORRUPT JSON in memory table
	_, err = db.Exec("INSERT INTO memory (uid, events_json, head) VALUES (?, '[[{invalid_json', 0)", uid)
	if err != nil {
		t.Fatalf("Failed to insert corrupt memory: %v", err)
	}

	// Try to load
	world := ecs.NewWorld()
	tm := &TickManager{World: &world}

	err = LoadWorld(tm, db)

	// Expect error once fixed. Currently it likely returns nil.
	if err == nil {
		t.Errorf("Expected LoadWorld to return an error for corrupted JSON, but it returned nil")
	}
}
