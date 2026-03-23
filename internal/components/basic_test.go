package components

import (
	"testing"
	"unsafe"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

func TestComponentSanity(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[Position](&world)
	velID := ecs.ComponentID[Velocity](&world)

	// Spawn 10 entities
	for i := 0; i < 10; i++ {
		entity := world.NewEntity()
		world.Add(entity, posID)
		world.Add(entity, velID)

		pos := (*Position)(world.Get(entity, posID))
		vel := (*Velocity)(world.Get(entity, velID))

		pos.X = float32(i)
		pos.Y = float32(i * 2)
		vel.X = 1.0
		vel.Y = 0.5
	}

	// Query and verify
	q := world.Query(filter.All(posID, velID))
	count := 0
	for q.Next() {
		pos := (*Position)(q.Get(posID))
		vel := (*Velocity)(q.Get(velID))

		if pos.X != float32(count) || pos.Y != float32(count * 2) {
			t.Errorf("Position mismatch at entity %d", count)
		}

		if vel.X != 1.0 || vel.Y != 0.5 {
			t.Errorf("Velocity mismatch at entity %d", count)
		}

		count++
	}

	if count != 10 {
		t.Errorf("Expected 10 entities, found %d", count)
	}
}

// Phase 03.1 & 03.3: DOD Size Verification
func TestComponentSizes(t *testing.T) {
	// Verify sizes to enforce DOD flat memory limits

	// Identity: uint64 (8) + string (16) + uint32 (4) + uint16 (2) = 30 bytes normally, padded to 32 bytes.
	// Actually: uint64 (8), string (16), uint32 (4), uint16 (2) -> 30 + 2 padding = 32 bytes on 64-bit architecture
	idSize := unsafe.Sizeof(Identity{})
	if idSize > 32 {
		t.Errorf("Identity struct size too large: %d bytes (expected <= 32)", idSize)
	}

	// GenomeComponent: 4 * uint8 (1) + 2 * uint32 (4) + 2 * uint16 (2) = 16 bytes
	genSize := unsafe.Sizeof(GenomeComponent{})
	if genSize != 16 {
		t.Errorf("GenomeComponent struct size should be exactly 16 bytes, got %d", genSize)
	}

	// Legacy: 2 * uint32 (4) = 8 bytes
	legSize := unsafe.Sizeof(Legacy{})
	if legSize != 8 {
		t.Errorf("Legacy struct size should be exactly 8 bytes, got %d", legSize)
	}

	// Needs: 4 * float32 (4) = 16 bytes
	needsSize := unsafe.Sizeof(Needs{})
	if needsSize != 16 {
		t.Errorf("Needs struct size should be exactly 16 bytes, got %d", needsSize)
	}

	// Phase 06.1 & 06.2: Social Graph Component Sizes
	// Affiliation: 5 * uint32 (4) = 20 bytes -> padded to 24
	affSize := unsafe.Sizeof(Affiliation{})
	if affSize > 24 {
		t.Errorf("Affiliation struct size too large: %d bytes (expected <= 24)", affSize)
	}

	// MemoryEvent: uint64 (8) + uint64 (8) + uint8 (1) + uint16 (2) + int32 (4) + padding = 24 bytes on 64-bit
	meSize := unsafe.Sizeof(MemoryEvent{})
	if meSize > 24 {
		t.Errorf("MemoryEvent struct size too large: %d bytes (expected <= 24)", meSize)
	}

	// Memory: [50]MemoryEvent (50 * 24) + uint8 (1) + padding
	// 50 * 24 = 1200 + 1 = 1201 + 7 padding = 1208 bytes
	memSize := unsafe.Sizeof(Memory{})
	if memSize > 1208 {
		t.Errorf("Memory struct size too large: %d bytes (expected <= 1208)", memSize)
	}

	// Phase 05.1: Settlement Component Sizes
	// SettlementLogic: 1 * uint16 (2) = 2 bytes
	slSize := unsafe.Sizeof(SettlementLogic{})
	if slSize != 2 {
		t.Errorf("SettlementLogic struct size should be exactly 2 bytes, got %d", slSize)
	}

	// StorageComponent: 4 * uint32 (4) = 16 bytes
	storageSize := unsafe.Sizeof(StorageComponent{})
	if storageSize != 16 {
		t.Errorf("StorageComponent struct size should be exactly 16 bytes, got %d", storageSize)
	}

	// Payload: 4 * uint32 (4) = 16 bytes
	payloadSize := unsafe.Sizeof(Payload{})
	if payloadSize != 16 {
		t.Errorf("Payload struct size should be exactly 16 bytes, got %d", payloadSize)
	}

	// PopulationComponent: uint32 (4) + []CitizenData (24) = 28 bytes + 4 padding = 32 bytes
	popSize := unsafe.Sizeof(PopulationComponent{})
	if popSize > 32 {
		t.Errorf("PopulationComponent struct size too large: %d bytes (expected <= 32)", popSize)
	}

	// CitizenData: GenomeComponent (16) + uint32 (4) + Age (2) + padding (2) = 24 bytes
	citizenSize := unsafe.Sizeof(CitizenData{})
	if citizenSize != 24 {
		t.Errorf("CitizenData struct size should be exactly 24 bytes, got %d", citizenSize)
	}

	// Phase 05.2: Ruin Component Size
	// RuinComponent: uint32 (4) + string (16) = 20 bytes normally.
	// 20 + 4 padding = 24 bytes on 64-bit architecture
	ruinSize := unsafe.Sizeof(RuinComponent{})
	if ruinSize > 24 {
		t.Errorf("RuinComponent struct size too large: %d bytes (expected <= 24)", ruinSize)
	}

	// Phase 07.1 & 07.5: Secret Component Sizes
	// Secret: uint64 (8) + uint32 (4) + uint8 (1) + uint32 (4) = 17 bytes -> 24 bytes padded
	secretSize := unsafe.Sizeof(Secret{})
	if secretSize > 24 {
		t.Errorf("Secret struct size too large: %d bytes (expected <= 24)", secretSize)
	}

	// SecretComponent: []Secret (24 bytes for slice header)
	secretCompSize := unsafe.Sizeof(SecretComponent{})
	if secretCompSize > 24 {
		t.Errorf("SecretComponent struct size too large: %d bytes (expected <= 24)", secretCompSize)
	}

	// Phase 13.1: Market Component Size
	// MarketComponent: 5 * float32 (4) + uint32 (4) = 24 bytes
	marketSize := unsafe.Sizeof(MarketComponent{})
	if marketSize != 24 {
		t.Errorf("MarketComponent struct size should be exactly 24 bytes, got %d", marketSize)
	}

	// Phase 24.1: Labor Union Sizes
	// StrikeMarker: uint64 (8) = 8 bytes
	strikeMarkerSize := unsafe.Sizeof(StrikeMarker{})
	if strikeMarkerSize != 8 {
		t.Errorf("StrikeMarker struct size should be exactly 8 bytes, got %d", strikeMarkerSize)
	}

	// Phase 07.5: Belief Component Sizes
	// Belief: uint32 (4) + int32 (4) = 8 bytes
	beliefSize := unsafe.Sizeof(Belief{})
	if beliefSize != 8 {
		t.Errorf("Belief struct size should be exactly 8 bytes, got %d", beliefSize)
	}

	// BeliefComponent: []Belief (24 bytes for slice header)
	beliefCompSize := unsafe.Sizeof(BeliefComponent{})
	if beliefCompSize > 24 {
		t.Errorf("BeliefComponent struct size too large: %d bytes (expected <= 24)", beliefCompSize)
	}

	// Phase 07.3: Linguistic Drift
	// CultureComponent: uint64 (8) + uint32 (4) + 2*uint16 (4) = 16 bytes.
	cultureSize := unsafe.Sizeof(CultureComponent{})
	if cultureSize > 16 {
		t.Errorf("CultureComponent struct size too large: %d bytes (expected <= 16)", cultureSize)
	}

	// Phase 09.4: Physical Legend Components
	// LegendComponent: uint32 (4) + uint32 (4) + []uint32 (24) = 32 bytes
	legendSize := unsafe.Sizeof(LegendComponent{})
	if legendSize > 32 {
		t.Errorf("LegendComponent struct size too large: %d bytes (expected <= 32)", legendSize)
	}

	itemEntitySize := unsafe.Sizeof(ItemEntity{})
	if itemEntitySize > 0 {
		t.Errorf("ItemEntity struct size should be exactly 0 bytes (tag component), got %d", itemEntitySize)
	}

	// Phase 32.1: Artifact Equipment (Auras of Legitimacy)
	equipmentSize := unsafe.Sizeof(EquipmentComponent{})
	if equipmentSize != 40 {
		t.Errorf("EquipmentComponent struct size should be exactly 40 bytes, got %d", equipmentSize)
	}

	// Phase 10.1: Debt Default Execution Component
	// LoanContractComponent: uint64 (8) + uint64 (8) + uint32 (4) = 20 bytes -> padded to 24
	loanSize := unsafe.Sizeof(LoanContractComponent{})
	if loanSize > 24 {
		t.Errorf("LoanContractComponent struct size too large: %d bytes (expected <= 24)", loanSize)
	}

	// Phase 15.3: CurrencyComponent
	// IssuerID uint32 (4) + Value float32 (4) + Debasement float32 (4) = 12 bytes
	currencySize := unsafe.Sizeof(CurrencyComponent{})
	if currencySize > 12 {
		t.Errorf("CurrencyComponent struct size too large: %d bytes (expected <= 12)", currencySize)
	}

	// Phase 16.1: CountryComponent
	// StandardCurrencyID uint32 (4) + Debasement float32 (4) = 8 bytes
	countrySize := unsafe.Sizeof(CountryComponent{})
	if countrySize > 8 {
		t.Errorf("CountryComponent struct size too large: %d bytes (expected <= 8)", countrySize)
	}

	// Phase 29.1: Geopolitical Resource Wars
	// TargetCountryID uint32 (4) + Active bool (1) + padding uint8 (1) + padding uint16 (2) = 8 bytes
	warTrackerSize := unsafe.Sizeof(WarTrackerComponent{})
	if warTrackerSize != 8 {
		t.Errorf("WarTrackerComponent struct size should be exactly 8 bytes, got %d", warTrackerSize)
	}

	// Phase 10.2: Bureaucratic Delay Components
	orderEntitySize := unsafe.Sizeof(OrderEntity{})
	if orderEntitySize > 0 {
		t.Errorf("OrderEntity struct size should be exactly 0 bytes (tag component), got %d", orderEntitySize)
	}

	// OrderComponent: uint64 (8) + uint32 (4) = 12 bytes -> padded to 16
	orderCompSize := unsafe.Sizeof(OrderComponent{})
	if orderCompSize > 16 {
		t.Errorf("OrderComponent struct size too large: %d bytes (expected <= 16)", orderCompSize)
	}

	capitalEntitySize := unsafe.Sizeof(CapitalComponent{})
	if capitalEntitySize > 0 {
		t.Errorf("CapitalComponent struct size should be exactly 0 bytes (tag component), got %d", capitalEntitySize)
	}

	// Phase 35.1: Sovereign Legitimacy Engine
	legitimacyCompSize := unsafe.Sizeof(LegitimacyComponent{})
	if legitimacyCompSize != 8 {
		t.Errorf("LegitimacyComponent size broke DOD alignment: expected exactly 8 bytes, got %d", legitimacyCompSize)
	}

	// LoyaltyComponent: uint32 (4) = 4 bytes
	loyaltyCompSize := unsafe.Sizeof(LoyaltyComponent{})
	if loyaltyCompSize != 4 {
		t.Errorf("LoyaltyComponent struct size should be exactly 4 bytes, got %d", loyaltyCompSize)
	}

	// Phase 10.3: Biological Entropy
	// DiseaseEntity: uint32 (4) + uint8 (1) = 5 bytes -> padded to 8 bytes on 64-bit
	diseaseSize := unsafe.Sizeof(DiseaseEntity{})
	if diseaseSize > 8 {
		t.Errorf("DiseaseEntity struct size too large: %d bytes (expected <= 8)", diseaseSize)
	}

	// ImmunityTag: []uint32 slice header = 24 bytes
	immunitySize := unsafe.Sizeof(ImmunityTag{})
	if immunitySize > 24 {
		t.Errorf("ImmunityTag struct size too large: %d bytes (expected <= 24)", immunitySize)
	}
}

func TestDesperationComponentSize(t *testing.T) {
	desperationSize := unsafe.Sizeof(DesperationComponent{})
	jurisdictionSize := unsafe.Sizeof(JurisdictionComponent{})
	ledgerSize := unsafe.Sizeof(LedgerComponent{})
	if desperationSize > 1 {
		t.Errorf("DesperationComponent too large! Got %d bytes, expected <= 1", desperationSize)
	}
	if jurisdictionSize != 20 {
		t.Errorf("JurisdictionComponent size broke DOD alignment: expected 20, got %d", jurisdictionSize)
	}
	if ledgerSize != 24 {
		t.Errorf("LedgerComponent size broke DOD alignment: expected 24, got %d", ledgerSize)
	}

	vitalsSize := unsafe.Sizeof(VitalsComponent{})
	if vitalsSize != 16 {
		t.Errorf("VitalsComponent size broke DOD alignment: expected 16, got %d", vitalsSize)
	}

	disasterSize := unsafe.Sizeof(DisasterComponent{})
	if disasterSize != 16 {
		t.Errorf("DisasterComponent size broke DOD alignment: expected 16, got %d", disasterSize)
	}
}

func TestScapegoatComponentSize(t *testing.T) {
	// Phase 36.1: The Scapegoat & Witch Hunt Engine
	expected := uintptr(8)
	actual := unsafe.Sizeof(ScapegoatComponent{})
	if actual != expected {
		t.Errorf("ScapegoatComponent size expected %d bytes, got %d bytes", expected, actual)
	}
}

func TestEsotericMarkerSize(t *testing.T) {
	// Phase 49: The Witch Hunt Engine
	expected := uintptr(4)
	actual := unsafe.Sizeof(EsotericMarker{})
	if actual != expected {
		t.Errorf("EsotericMarker size expected %d bytes, got %d bytes", expected, actual)
	}
}

func TestQuarantineComponentSize(t *testing.T) {
	// Phase 37.1: The Quarantine Engine
	expected := uintptr(8)
	actual := unsafe.Sizeof(QuarantineComponent{})
	if actual != expected {
		t.Errorf("Expected QuarantineComponent to be %d bytes for DOD, got %d", expected, actual)
	}
}

func TestMercenaryContractComponentSize(t *testing.T) {
	// Phase 47: The Mercenary Engine
	expected := uintptr(16)
	actual := unsafe.Sizeof(MercenaryContractComponent{})
	if actual != expected {
		t.Errorf("Expected MercenaryContractComponent to be %d bytes for DOD, got %d", expected, actual)
	}
}
