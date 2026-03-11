package components

import (
	"testing"
	"unsafe"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

func TestComponentSanity(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[Position](&world)
	velID := ecs.ComponentID[Velocity](&world)

	// Spawn 10 entities
	for i := 0; i < 10; i++ {
		entity := world.NewEntity()
		world.Add(entity, posID)
		world.Add(entity, velID)

		pos := (*Position)(world.Get(entity, posID))
		vel := (*Velocity)(world.Get(entity, velID))

		pos.X = float32(i)
		pos.Y = float32(i * 2)
		vel.X = 1.0
		vel.Y = 0.5
	}

	// Query and verify
	q := world.Query(filter.All(posID, velID))
	count := 0
	for q.Next() {
		pos := (*Position)(q.Get(posID))
		vel := (*Velocity)(q.Get(velID))

		if pos.X != float32(count) || pos.Y != float32(count * 2) {
			t.Errorf("Position mismatch at entity %d", count)
		}

		if vel.X != 1.0 || vel.Y != 0.5 {
			t.Errorf("Velocity mismatch at entity %d", count)
		}

		count++
	}

	if count != 10 {
		t.Errorf("Expected 10 entities, found %d", count)
	}
}

// Phase 03.1 & 03.3: DOD Size Verification
func TestComponentSizes(t *testing.T) {
	// Verify sizes to enforce DOD flat memory limits

	// Identity: uint64 (8) + string (16) + uint32 (4) = 28 bytes normally, but string can cause padding depending on order.
	// Actually: uint64 (8), string (16), uint32 (4) -> 28 + 4 padding = 32 bytes on 64-bit architecture
	idSize := unsafe.Sizeof(Identity{})
	if idSize > 32 {
		t.Errorf("Identity struct size too large: %d bytes (expected <= 32)", idSize)
	}

	// Genetics: 4 * uint8 (1) = 4 bytes
	genSize := unsafe.Sizeof(Genetics{})
	if genSize != 4 {
		t.Errorf("Genetics struct size should be exactly 4 bytes, got %d", genSize)
	}

	// Legacy: 2 * uint32 (4) = 8 bytes
	legSize := unsafe.Sizeof(Legacy{})
	if legSize != 8 {
		t.Errorf("Legacy struct size should be exactly 8 bytes, got %d", legSize)
	}

	// Needs: 4 * float32 (4) = 16 bytes
	needsSize := unsafe.Sizeof(Needs{})
	if needsSize != 16 {
		t.Errorf("Needs struct size should be exactly 16 bytes, got %d", needsSize)
	}

	// Phase 05.1: Settlement Component Sizes
	// SettlementLogic: 1 * uint16 (2) = 2 bytes
	slSize := unsafe.Sizeof(SettlementLogic{})
	if slSize != 2 {
		t.Errorf("SettlementLogic struct size should be exactly 2 bytes, got %d", slSize)
	}

	// StorageComponent: 4 * uint32 (4) = 16 bytes
	storageSize := unsafe.Sizeof(StorageComponent{})
	if storageSize != 16 {
		t.Errorf("StorageComponent struct size should be exactly 16 bytes, got %d", storageSize)
	}

	// PopulationComponent: uint32 (4) + []CitizenData (24) = 28 bytes + 4 padding = 32 bytes
	popSize := unsafe.Sizeof(PopulationComponent{})
	if popSize > 32 {
		t.Errorf("PopulationComponent struct size too large: %d bytes (expected <= 32)", popSize)
	}

	// CitizenData: Genetics (4) + uint32 (4) = 8 bytes
	citizenSize := unsafe.Sizeof(CitizenData{})
	if citizenSize != 8 {
		t.Errorf("CitizenData struct size should be exactly 8 bytes, got %d", citizenSize)
	}

	// Phase 05.2: Ruin Component Size
	// RuinComponent: uint32 (4) + string (16) = 20 bytes normally.
	// 20 + 4 padding = 24 bytes on 64-bit architecture
	ruinSize := unsafe.Sizeof(RuinComponent{})
	if ruinSize > 24 {
		t.Errorf("RuinComponent struct size too large: %d bytes (expected <= 24)", ruinSize)
	}
}
