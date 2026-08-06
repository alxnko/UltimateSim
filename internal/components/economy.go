package components

// Grand Strategy Phase (P2.6): Economy layer components.
// Spec: docs/superpowers/specs/2026-08-06-grand-strategy-playable.md

// MaxTaxRatePercent caps TaxPolicyComponent.Rate (0..50 percent).
const MaxTaxRatePercent uint8 = 50

// TaxPolicyComponent stores the national tax rate on the country capital
// entity (the entity carrying CountryComponent + CapitalComponent, identified
// by its Affiliation.CountryID — same convention as FindCapitalOf). A capital
// without this component taxes at the systems.DefaultTaxRatePercent baseline.
type TaxPolicyComponent struct {
	Rate uint8  // Tax rate percent, clamped 0..MaxTaxRatePercent
	_    uint8  // Padding
	_    uint16 // Padding
	_    uint32 // Padding to exactly 8 bytes
}

// TradeRouteComponent represents an established trade route between two
// cities, stored as its own entity. FromCity/ToCity are CityIDs
// (uint32(village Identity.ID), the CityBinder convention); the route is
// undirected — goods flow toward the pricier endpoint each transfer window.
type TradeRouteComponent struct {
	FromCity uint32
	ToCity   uint32
	Volume   uint16 // Goods moved per good per transfer window
	_        uint16 // Padding
	_        uint32 // Padding to exactly 16 bytes
}
