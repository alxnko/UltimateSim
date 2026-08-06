package components

import (
	"testing"
	"unsafe"
)

// DOD size verification for the Grand Strategy dynasty/intrigue/council structs.
func TestDynastyComponentSizes(t *testing.T) {
	// DynastyComponent: uint64 (8) + uint16 (2) + bool (1) + [5]byte padding = 16 bytes.
	if size := unsafe.Sizeof(DynastyComponent{}); size != 16 {
		t.Errorf("DynastyComponent size should be exactly 16 bytes, got %d", size)
	}

	// PlotComponent: 2 * uint64 (16) + 2 * uint16 (4) + uint8 (1) + bool (1) + uint16 padding = 24 bytes.
	if size := unsafe.Sizeof(PlotComponent{}); size != 24 {
		t.Errorf("PlotComponent size should be exactly 24 bytes, got %d", size)
	}

	// CouncilComponent: 4 * uint64 = 32 bytes, naturally aligned.
	if size := unsafe.Sizeof(CouncilComponent{}); size != 32 {
		t.Errorf("CouncilComponent size should be exactly 32 bytes, got %d", size)
	}
}
