package components

// Phase 01.5: Basic Components and Data-Oriented Design (DOD) Verification
// Structs use flat memory rules and explicit IDs instead of pointers.
// We use float32 instead of float64 for Position and Velocity to minimize memory overhead.

// Traits Bitmask Constants
const (
	TraitRiskTaker uint32 = 1 << 0
	TraitCautious  uint32 = 1 << 1
)

// Identity component
// Phase 03.1: Genesis Base Structs
type Identity struct {
	ID         uint64
	Name       string
	BaseTraits uint32
}

// Genetics component
// Phase 03.1: Genesis Base Structs
type Genetics struct {
	Strength  uint8
	Beauty    uint8
	Health    uint8
	Intellect uint8
}

// Legacy component
// Phase 03.1: Genesis Base Structs
type Legacy struct {
	Prestige      uint32
	InheritedDebt uint32
}

// Needs component
// Phase 03.3: The Metabolic Engine
type Needs struct {
	Food   float32
	Rest   float32
	Safety float32
	Wealth float32
}

// Position component
type Position struct {
	X float32
	Y float32
}

// Path component
// Phase 04.2: Async Path Queue Pool
// Stores the tactical node-to-node float32 positions for MovementSystem to traverse.
type Path struct {
	Nodes   []Position
	HasPath bool
	TargetX float32
	TargetY float32
}

// Velocity component
type Velocity struct {
	X float32
	Y float32
}
