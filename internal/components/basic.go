package components

// Phase 01.5: Basic Components and Data-Oriented Design (DOD) Verification
// Structs use flat memory rules and explicit IDs instead of pointers.
// We use float32 instead of float64 for Position and Velocity to minimize memory overhead.

// Traits Bitmask Constants
const (
	TraitRiskTaker uint32 = 1 << 0
	TraitCautious  uint32 = 1 << 1
	TraitGossip    uint32 = 1 << 2
)

// Phase 09.5: Item Inheritance Threshold
const ExtremePrestigeThreshold uint32 = 100

// Interaction Types Constants
const (
	InteractionGossip   uint8 = 1
	InteractionLanguage uint8 = 2 // Phase 07.3: Linguistic Drift
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

// Phase 10.1: Debt Default Execution (The Hook Trap)
type LoanContractComponent struct {
	CreditorID uint64
	DueTick    uint64
	AssetID    uint32
}

// Phase 10.3: Biological Entropy (Plagues & Immune Arrays)

// DiseaseEntity identifies a plague instance on the map.
type DiseaseEntity struct {
	ID        uint32
	Lethality uint8
}

// ImmunityTag identifies an entity that survived a plague and ignores subsequent identical evaluations.
type ImmunityTag struct {
	ImmuneTo []uint32
}

// Phase 10.2: Bureaucratic Delay (Administrative Entropy)

// OrderEntity is a tag component identifying administrative couriers traversing the map.
type OrderEntity struct{}

// OrderComponent tracks the destination and creation time of a specific political action.
type OrderComponent struct {
	CreationTick uint64
	TargetCityID uint32
}

// CapitalComponent is a tag component identifying a central governing city.
type CapitalComponent struct{}

// LoyaltyComponent determines the threshold before a city ignores or intercepts orders.
type LoyaltyComponent struct {
	Value uint32
}

// Phase 06.2: Interaction Telemetry
type MemoryEvent struct {
	TargetID        uint64
	TickStamp       uint64
	InteractionType uint8
	LanguageID      uint16 // Phase 07.3: Linguistic Drift - Storing Language of interaction
	Value           int32  // Increased from int8 to int32 to store SecretID while preserving 24-byte padding limit
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

// Phase 09.4: Physical Legend Components
// ItemEntity is a tag component identifying physical legendary items on the map.
type ItemEntity struct{}

// LegendComponent represents an artifact with physical existence and history.
type LegendComponent struct {
	NameID   uint32   // Unique ID from SecretRegistry representing the item's name
	Prestige uint32   // Legacy prestige from the original holder
	History  []uint32 // Array of EventIDs tracking the item's history
}

// Phase 09.1: The Caravan Entity

// Caravan is a tag component identifying mobile logistics units.
type Caravan struct{}

// Payload tracks trade goods limits on a Caravan in flat memory arrays.
// Mirrors StorageComponent exactly for DOD 16-byte alignment.
type Payload struct {
	Wood  uint32
	Stone uint32
	Iron  uint32
	Food  uint32
}

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
	BeliefID uint32 // Phase 07.5: Ideological Infection metadata flag
}

// SecretComponent holds the known secrets for an entity.
type SecretComponent struct {
	Secrets []Secret
}

// Phase 07.5: Ideological Infection (The Memetic Engine)

// Belief represents a specific ideological or cultural dogma an entity adheres to.
type Belief struct {
	BeliefID uint32
	Weight   int32
}

// BeliefComponent tracks the ideas an entity has been exposed to or follows.
type BeliefComponent struct {
	Beliefs []Belief // Kept as a flat slice for DOD instead of a Go map
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

// Phase 11.2: Instanced 3D Control (raylib-go)

// Possessed is a tag component identifying an entity possessed by the player.
type Possessed struct{}

// Phase 07.3: Linguistic Drift

// CultureComponent tracks language mutation and dialect formation over extended ticks.
type CultureComponent struct {
	DialectTickStamp        uint64 // The tick stamp of the last interaction with the same language
	ForeignInteractionTicks uint32 // Ticks spent interacting with the dominant foreign language
	LanguageID              uint16 // Current LanguageID
	ForeignLanguageID       uint16 // Tracked distinct LanguageID for potential Pidgin creation
}
