package systems_test

import (
	"testing"
	"unsafe"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/network"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 12.2: Delta Extraction Queries E2E Tests
func TestDeltaExtractionSystem(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	idID := ecs.ComponentID[components.Identity](&world)

	extractionSys := systems.NewDeltaExtractionSystem(&world)

	// Spawn 3 Entities
	// Entity 1: Stationary (Velocity = 0, 0)
	e1 := world.NewEntity()
	world.Add(e1, posID, velID, idID)
	(*components.Position)(world.Get(e1, posID)).X = 10
	(*components.Position)(world.Get(e1, posID)).Y = 10
	(*components.Velocity)(world.Get(e1, velID)).X = 0
	(*components.Velocity)(world.Get(e1, velID)).Y = 0
	(*components.Identity)(world.Get(e1, idID)).ID = 100

	// Entity 2: Moving on X axis (Velocity = 1.5, 0)
	e2 := world.NewEntity()
	world.Add(e2, posID, velID, idID)
	(*components.Position)(world.Get(e2, posID)).X = 20
	(*components.Position)(world.Get(e2, posID)).Y = 20
	(*components.Velocity)(world.Get(e2, velID)).X = 1.5
	(*components.Velocity)(world.Get(e2, velID)).Y = 0
	(*components.Identity)(world.Get(e2, idID)).ID = 101

	// Entity 3: Moving on Y axis (Velocity = 0, -2.5)
	e3 := world.NewEntity()
	world.Add(e3, posID, velID, idID)
	(*components.Position)(world.Get(e3, posID)).X = 30
	(*components.Position)(world.Get(e3, posID)).Y = 30
	(*components.Velocity)(world.Get(e3, velID)).X = 0
	(*components.Velocity)(world.Get(e3, velID)).Y = -2.5
	(*components.Identity)(world.Get(e3, idID)).ID = 102

	// Run extraction tick
	extractionSys.Update(&world)

	deltas := extractionSys.GetCurrentDeltas()

	// Verify only moving entities were extracted
	if len(deltas) != 2 {
		t.Fatalf("Expected exactly 2 deltas extracted, got %d", len(deltas))
	}

	// Verify specific deterministic entity extractions
	hasE2 := false
	hasE3 := false

	for _, d := range deltas {
		if d.EntityID == 101 {
			hasE2 = true
			if d.X != 20 || d.Y != 20 {
				t.Errorf("Entity 101 delta coordinates incorrect, got X:%f Y:%f", d.X, d.Y)
			}
		}
		if d.EntityID == 102 {
			hasE3 = true
			if d.X != 30 || d.Y != 30 {
				t.Errorf("Entity 102 delta coordinates incorrect, got X:%f Y:%f", d.X, d.Y)
			}
		}
	}

	if !hasE2 || !hasE3 {
		t.Errorf("Extracted deltas did not match expected IDs (101, 102)")
	}

	// Run another tick to verify s.currentDeltas slices are properly resetting capacity
	(*components.Velocity)(world.Get(e2, velID)).X = 0 // Stop E2

	extractionSys.Update(&world)
	deltas2 := extractionSys.GetCurrentDeltas()

	if len(deltas2) != 1 {
		t.Fatalf("Expected exactly 1 delta extracted on tick 2, got %d", len(deltas2))
	}
	if deltas2[0].EntityID != 102 {
		t.Errorf("Expected Entity 102 to be extracted on tick 2, got %d", deltas2[0].EntityID)
	}
}

func TestPositionDeltaSize(t *testing.T) {
	// Phase 12.2: DOD constraints require PositionDelta payload structures perfectly map to 16-byte bounds
	// EntityID (uint64) = 8 bytes
	// X (float32)       = 4 bytes
	// Y (float32)       = 4 bytes
	// Total             = 16 bytes exactly on cache line

	size := unsafe.Sizeof(network.PositionDelta{})
	if size != 16 {
		t.Errorf("PositionDelta size must strictly be exactly 16 bytes for network cache alignment, got %d bytes", size)
	}
}
