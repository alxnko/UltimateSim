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

// Phase 13.2: Labor Rebalancing
const (
	JobNone       uint8 = 0
	JobFarmer     uint8 = 1
	JobLumberjack uint8 = 2
	JobArtisan    uint8 = 3 // A processing job that can be reverted
	JobGuard      uint8 = 4 // Phase 18.2: The Guard System
	JobPreacher   uint8 = 5 // Phase 20.1: Ideological Warfare
	JobCaster     uint8 = 6 // Phase 20.2: Abstract Physics
	JobBandit     uint8 = 7 // Phase 26.1: Caravan Banditry & Supply Chain Collapse
)

// Phase 09.5: Item Inheritance Threshold
const ExtremePrestigeThreshold uint32 = 100

// Belief IDs
const (
	BeliefXenophobia uint32 = 100 // Phase 20.3: Traumatic Traditions
)

// Interaction Types Constants
const (
	InteractionGossip   uint8 = 1
	InteractionLanguage uint8 = 2 // Phase 07.3: Linguistic Drift
	InteractionAssault  uint8 = 3 // Phase 18.1: Law Definitions
	InteractionTheft    uint8 = 4 // Phase 18.1: Law Definitions
	InteractionMurder   uint8 = 5 // Phase 23.1: The Blood Feud Engine
)

// Identity component
// Phase 03.1: Genesis Base Structs
type Identity struct {
	ID         uint64
	Name       string
	BaseTraits uint32
	Age        uint16
}

// GenomeComponent component
// Phase 03.1: Genesis Base Structs
// Phase 19.1: Deep Genetics expansion
type GenomeComponent struct {
	Strength  uint8
	Beauty    uint8
	Health    uint8
	Intellect uint8
	Dominant  uint32
	Recessive uint32
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

// Phase 04.5: The Epistemological Layer (Physical History)

// Ledger is a tag component identifying physical records on the map.
type Ledger struct{}

// LedgerComponent represents materialized information (history, propaganda).
type LedgerComponent struct {
	Secrets []uint32
}

type Position struct {
	X float32
	Y float32
}

// Phase 06.1: Societal Hierarchies
// Phase 14: True Individual NPCs
type Affiliation struct {
	FamilyID  uint32
	ClanID    uint32
	GuildID   uint32
	CityID    uint32
	CountryID uint32
	_         uint32 // Padding to maintain 24-byte alignment
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

// Phase 14: True Individual NPCs
// NPC is a tag component replacing FamilyCluster, identifies a single human actor.
type NPC struct{}

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
	Genetics   GenomeComponent
	BaseTraits uint32
	Age        uint16
	_          uint16 // Padding to exactly 20 bytes
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

// Phase 32.1: Artifact Equipment (Auras of Legitimacy)
type EquipmentComponent struct {
	Weapon   LegendComponent // Embedded 32-byte artifact
	Equipped bool            // 1 byte
	_        uint8           // 1 byte padding
	_        uint16          // 2 bytes padding
	_        uint32          // 4 bytes padding to 40 bytes exactly
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

// Phase 33: The Refugee Crisis
// RefugeeCluster is a tag component identifying displaced populations.
type RefugeeCluster struct{}

// RefugeeData tracks a moving population that has lost its Village entity.
type RefugeeData struct {
	Count    uint32           // 4 bytes
	_        uint32           // 4 bytes padding to align CultureComponent to 8 bytes
	Culture  CultureComponent // 16 bytes
	Citizens []CitizenData    // 24 bytes
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

// Item IDs for contraband mapping
const (
	ItemWood  uint8 = 1
	ItemStone uint8 = 2
	ItemIron  uint8 = 3
	ItemFood  uint8 = 4
)

// Phase 13.1: Local Price Discovery (Market Logic)
// MarketComponent maintains local trade pricing determined by local demand and supply.
type MarketComponent struct {
	WoodPrice  float32
	StonePrice float32
	IronPrice  float32
	FoodPrice  float32
	WageRate   float32
	_          uint32 // Padding to exactly 24 bytes
}

// Phase 18.1: Contraband Logic
// ContrabandComponent maintains local laws regarding illegal items.
type ContrabandComponent struct {
	Contraband uint32 // Bitmask flagging illegal ItemIDs
}


// Phase 24.1: The Labor Union Engine
// StrikeMarker tags an NPC that has quit due to unpaid wages and is striking against a specific employer.
type StrikeMarker struct {
	TargetEmployerID uint64
}

// Phase 21.1: DesperationSystem
// DesperationComponent represents an NPC's inclination towards crime due to starvation or poverty.
type DesperationComponent struct {
	Level uint8 // 0-100 bounds
}

// Phase 13.2: Labor Rebalancing
// Phase 15.2: Employment & Wages (expanded)
type JobComponent struct {
	JobID      uint8
	EmployerID uint64
}

// Phase 15.1: Individual Economic Agency

// BusinessEntity is a tag component identifying an entity as a business.
type BusinessEntity struct{}

// BusinessComponent tracks ownership and business details.
type BusinessComponent struct {
	OwnerID uint64
}

// TreasuryComponent tracks wealth of an entity, such as a Business.
type TreasuryComponent struct {
	Wealth float32
}

// Phase 15.4: Physical Locations & Workplaces

// WorkplaceComponent defines the physical map grid location of a business.
type WorkplaceComponent struct {
	X float32
	Y float32
}

// Phase 07.3: Linguistic Drift

// CultureComponent tracks language mutation and dialect formation over extended ticks.
type CultureComponent struct {
	DialectTickStamp        uint64 // The tick stamp of the last interaction with the same language
	ForeignInteractionTicks uint32 // Ticks spent interacting with the dominant foreign language
	LanguageID              uint16 // Current LanguageID
	ForeignLanguageID       uint16 // Tracked distinct LanguageID for potential Pidgin creation
}

// Phase 15.3: Currency & Debt

// CoinEntity is a tag component identifying physical coin objects.
type CoinEntity struct{}

// CurrencyComponent represents physical coins and their market value.
type CurrencyComponent struct {
	IssuerID   uint32
	Value      float32
	Debasement float32
}

// Phase 16.1: The Country Entity (Macro-State)

// CountryComponent is a higher-level tag attached to a Capital entity that manages sub-affiliations.
type CountryComponent struct {
	StandardCurrencyID uint32
	Debasement         float32
}

// Phase 29.1: Geopolitical Resource Wars
type WarTrackerComponent struct {
	TargetCountryID uint32
	Active          bool
	_               uint8
	_               uint16
}

// Phase 16.2: Strategic Unions & Pacts

// UnionType constants
const (
	UnionDefensePact  uint8 = 0
	UnionCurrency     uint8 = 1
	UnionEconomicBloc uint8 = 2
)

// UnionEntity is a specialized non-physical entity representing a treaty or agreement between independent Countries or Cities.
type UnionEntity struct{}

// UnionComponent stores the data of a strategic union.
type UnionComponent struct {
	UnionType        uint8
	SharedCurrencyID uint32
	MemberIDs        []uint32 // Array of City/Country IDs
}

// MilitaryForce represents the armed forces of a city/country.
// Used for Defense Pacts.
type MilitaryForce struct{}

// Phase 17.1: Maritime Reach & Naval Logistics

// PortComponent is a tag component attached to VillageEntity structures resting adjacent to Ocean tiles.
type PortComponent struct{}

// ShipComponent is a component attached to a vessel entity.
type ShipComponent struct {
	Hull uint32
}

// Passenger tracks passenger slots for trans-oceanic migration.
type Passenger struct {
	EntityID uint64
}

// PassengerComponent holds the passenger array for Ship entities.
// Slices map to 24 bytes in 64-bit Go, adhering to DOD constraints.
type PassengerComponent struct {
	Passengers []Passenger
}

// Phase 16.4: Administrative Reach & Friction
// (Entities will unilaterally fracture if distance from Capital exceeds max thresholds, removing their CountryID).

// Phase 18.1: Jurisdiction & Law Definitions

// JurisdictionComponent defines the geometric bounds and the legal parameters.
// This is typically attached to a VillageEntity or a Capital.
type JurisdictionComponent struct {
	RadiusSquared    float32 // Squared radius mapped around the entity's Position
	IllegalActionIDs uint32  // Bitmask of interaction types that are considered crimes (e.g. 1<<InteractionAssault)
	Corruption       uint32  // Phase 22.1: The Corruption Engine
	BannedSecretID   uint32  // Phase 04.5: The Epistemological Layer (Propaganda Erasure Target)
	Trauma           uint16  // Phase 20.3: Traumatic Traditions
	_                uint16  // Padding to maintain 4-byte alignment
}

// Phase 18.2: Detection & The Guard System

// CrimeMarker is tagged onto an entity observed committing a crime within a Jurisdiction.
type CrimeMarker struct {
	CrimeLevel uint8
	Bounty     uint32
}

// Phase 20.1: Ideological Warfare

// CrusaderEntity is a tag component attached to aggressive entities spawned during a Holy War.
type CrusaderEntity struct{}

// CrusadeComponent tracks the target city for aggressive spawns.
type CrusadeComponent struct {
	TargetCityID uint32
}

// Phase 19.4: Advanced Biology (Vitals)
type VitalsComponent struct {
	Stamina       float32
	Blood         float32
	Pain          float32
	Consciousness float32
}

// Phase 31: Systemic Entropy (Natural Disasters)

// NaturalDisasterEntity is a tag component identifying an active disaster event on the map.
type NaturalDisasterEntity struct{}

// DisasterComponent tracks the properties of a natural disaster.
type DisasterComponent struct {
	RadiusSquared float32
	Strength      float32
	Type          uint32 // e.g., Earthquake, Flood, etc.
	_             uint32 // Padding to exactly 16 bytes
}
