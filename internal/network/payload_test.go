package network

import (
	"testing"
	"unsafe"
)

// Phase 12.2: DOD Alignment Verification
// Verifies that PositionDelta strictly adheres to 16-byte CPU cache bounds.
func TestPositionDeltaSize(t *testing.T) {
	size := unsafe.Sizeof(PositionDelta{})
	if size != 16 {
		t.Errorf("PositionDelta struct size should be exactly 16 bytes, got %d", size)
	}
}

func TestDeltaPayload(t *testing.T) {
	payload := DeltaPayload{
		Tick: 100,
		Deltas: []PositionDelta{
			{EntityID: 1, X: 10.5, Y: 20.5},
			{EntityID: 2, X: 30.0, Y: 40.0},
		},
	}

	if payload.Tick != 100 {
		t.Errorf("Expected Tick 100, got %d", payload.Tick)
	}

	if len(payload.Deltas) != 2 {
		t.Errorf("Expected 2 deltas, got %d", len(payload.Deltas))
	}

	if payload.Deltas[0].EntityID != 1 || payload.Deltas[0].X != 10.5 || payload.Deltas[0].Y != 20.5 {
		t.Errorf("Delta 0 mismatch: %+v", payload.Deltas[0])
	}
}
