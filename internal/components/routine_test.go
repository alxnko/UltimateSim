package components

import (
	"testing"
	"unsafe"
)

// Grand Strategy Phase (P5/L1): DOD size verification for RoutineComponent.
func TestRoutineComponentSize(t *testing.T) {
	// AnchorX (4) + AnchorY (4) + Phase (1) + JobSeen (1) + 2 padding = 12 bytes
	if s := unsafe.Sizeof(RoutineComponent{}); s != 12 {
		t.Errorf("RoutineComponent struct size should be exactly 12 bytes, got %d", s)
	}
}
