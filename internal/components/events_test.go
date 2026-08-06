package components

import (
	"testing"
	"unsafe"
)

// Grand Strategy Phase G1: DOD size verification for event components.
func TestEventComponentSizes(t *testing.T) {
	// GameEvent: 4*uint64 (32) + uint32 + int32 (8) + 2*uint8 (2) +
	// uint16 (2) + uint32 (4) explicit padding = 48 bytes, 8-aligned.
	if s := unsafe.Sizeof(GameEvent{}); s != 48 {
		t.Errorf("GameEvent struct size should be exactly 48 bytes, got %d", s)
	}

	// PendingEventsComponent: one slice header = 24 bytes on 64-bit.
	if s := unsafe.Sizeof(PendingEventsComponent{}); s != 24 {
		t.Errorf("PendingEventsComponent struct size should be exactly 24 bytes, got %d", s)
	}
}
