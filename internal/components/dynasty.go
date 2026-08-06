package components

// Grand Strategy Phase (spec P2.3, P2.4, P2.5): dynasty, intrigue, council.
// Flat DOD structs keyed by Identity.ID; size tests live in dynasty_test.go.

// Plot kinds for PlotComponent.Kind.
const (
	PlotSeizeRule   uint8 = 1 // Usurp the plotter's city leadership
	PlotAssassinate uint8 = 2 // Kill the target through the combat/death path
)

// Council seat identifiers for CouncilComponent.
const (
	SeatSteward   uint8 = 1
	SeatMarshal   uint8 = 2
	SeatDiplomat  uint8 = 3
	SeatSpymaster uint8 = 4
)

// DynastyComponent tracks the marital state of an NPC. SpouseID stores the
// spouse's Identity.ID (0 when unmarried). Children counts offspring credited
// to this union.
type DynastyComponent struct {
	SpouseID uint64
	Children uint16
	Married  bool
	_        [5]byte // Padding to exactly 16 bytes
}

// PlotComponent lives on the plotting NPC. TargetID is the victim's
// Identity.ID. Progress accumulates Power every plot cycle; at >= 100 the plot
// resolves. Exposed marks a discovered plot pending removal.
type PlotComponent struct {
	TargetID  uint64
	StartTick uint64
	Progress  uint16
	Power     uint16
	Kind      uint8
	Exposed   bool
	_         uint16 // Padding to exactly 24 bytes
}

// CouncilComponent hangs on the country capital entity (the Village carrying
// CapitalComponent + CountryComponent, same entity diplomacy/taxation use for
// country state). Each seat stores the appointee's Identity.ID (0 = vacant).
type CouncilComponent struct {
	Steward   uint64
	Marshal   uint64
	Diplomat  uint64
	Spymaster uint64
}
