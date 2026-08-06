package components

import (
	"testing"
	"unsafe"
)

// Grand Strategy Phase P2.2: DOD size verification for diplomacy components.
func TestDiplomacyComponentSizes(t *testing.T) {
	// CountryRelation: uint64 (8) + uint64 (8) + uint32 (4) + 2*int16 (2) +
	// 2*bool (1) + uint16 (2) + uint32 (4) = 32 bytes, no compiler padding.
	relSize := unsafe.Sizeof(CountryRelation{})
	if relSize != 32 {
		t.Errorf("CountryRelation struct size should be exactly 32 bytes, got %d", relSize)
	}

	// DiplomacyComponent: one slice header = 24 bytes on 64-bit.
	dipSize := unsafe.Sizeof(DiplomacyComponent{})
	if dipSize != 24 {
		t.Errorf("DiplomacyComponent struct size should be exactly 24 bytes, got %d", dipSize)
	}
}
