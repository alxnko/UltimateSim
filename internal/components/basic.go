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

// Phase 06.1: Societal Hierarchies
type Affiliation struct {
	ClanID    uint32
	GuildID   uint32
	CityID    uint32
	CountryID uint32
}

// Phase 06.2: Interaction Telemetry
type MemoryEvent struct {
	TargetID        uint64
	TickStamp       uint64
	InteractionType uint8
	Value           int8
}

type Memory struct {
	Events [50]MemoryEvent
	Head   uint8
}

// Phase 05.1: Settlement Conversion Components

// FamilyCluster is a tag component identifying migrating groups.
type FamilyCluster struct{}

// SettlementLogic tracks consecutive ticks at 0 velocity.
type SettlementLogic struct {
	TicksAtZeroVelocity uint16
}

// StorageComponent tracks inventory in flat memory arrays.
type StorageComponent struct {
	Wood  uint32
	Stone uint32
	Iron  uint32
	Food  uint32
}

// CitizenData stores genetic and trait data for individuals born within a settlement.
type CitizenData struct {
	Genetics   Genetics
	BaseTraits uint32
}

// PopulationComponent tracks headcount abstracting AI nodes inside city limits.
type PopulationComponent struct {
	Count    uint32
	Citizens []CitizenData
}

// Village is a tag component identifying stationary settlements.
type Village struct{}

// RuinComponent identifies a dead settlement to avoid processing its needs.
// Phase 05.2: The Ruin Transformation
type RuinComponent struct {
	Decay      uint32
	FormerName string
}

// Phase 07.1: Secret Registry (String Interning)

// Secret represents a known piece of information mapped from the SecretRegistry.
type Secret struct {
	OriginID uint64
	SecretID uint32
	Virality uint8
}

// SecretComponent holds the known secrets for an entity.
type SecretComponent struct {
	Secrets []Secret
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
