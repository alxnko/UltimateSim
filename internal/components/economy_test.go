package components

import (
	"testing"
	"unsafe"
)

// Grand Strategy Phase (P2.6): DOD size verification for economy components.
func TestEconomyComponentSizes(t *testing.T) {
	// TaxPolicyComponent: uint8 (1) + 1 + 2 + 4 padding = 8 bytes
	if s := unsafe.Sizeof(TaxPolicyComponent{}); s != 8 {
		t.Errorf("TaxPolicyComponent struct size should be exactly 8 bytes, got %d", s)
	}

	// TradeRouteComponent: 2 * uint32 (8) + uint16 (2) + 2 + 4 padding = 16 bytes
	if s := unsafe.Sizeof(TradeRouteComponent{}); s != 16 {
		t.Errorf("TradeRouteComponent struct size should be exactly 16 bytes, got %d", s)
	}
}
