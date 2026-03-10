package components

// Phase 01.5: Basic Components and Data-Oriented Design (DOD) Verification
// Structs use flat memory rules and explicit IDs instead of pointers.
// We use float32 instead of float64 for Position and Velocity to minimize memory overhead.

// Identity component
type Identity struct {
	ID uint64
}

// Position component
type Position struct {
	X float32
	Y float32
}

// Velocity component
type Velocity struct {
	X float32
	Y float32
}
