package systems

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

// TestLanguageDriftDialectFormation verifies an entity generating a new dialect
// after 10,000 ticks of isolation (no memory events matching their language).
func TestLanguageDriftDialectFormation(t *testing.T) {
	world := ecs.NewWorld()

	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)

	entity := world.NewEntity()
	world.Add(entity, cultureID, memoryID)

	culture := (*components.CultureComponent)(world.Get(entity, cultureID))
	memory := (*components.Memory)(world.Get(entity, memoryID))

	// Initial State
	culture.LanguageID = 5
	culture.DialectTickStamp = 0

	// Create system
	sys := &LanguageDriftSystem{}

	// Fast forward 9900 ticks (no interaction)
	for i := 0; i < 9900; i++ {
		sys.Update(&world)
	}

	if culture.LanguageID != 5 {
		t.Fatalf("LanguageID changed prematurely to %d", culture.LanguageID)
	}

	// Add an interaction event just before 10k to prevent dialect formation
	memory.Events[0] = components.MemoryEvent{
		TickStamp:       9950,
		InteractionType: components.InteractionLanguage,
		LanguageID:      5,
	}

	// Fast forward to 15,000 ticks (5,050 ticks since last interaction)
	for i := 0; i < 5100; i++ {
		sys.Update(&world)
	}

	if culture.LanguageID != 5 {
		t.Fatalf("LanguageID changed despite recent interaction. Got %d", culture.LanguageID)
	}

	// Clear memory and fast forward 10,000 more ticks from last interaction
	// 9950 + 10000 = 19950. System runs every 100 ticks.
	for i := 0; i < 5000; i++ {
		sys.Update(&world)
	}

	// System tick counter is around 20,000
	if culture.LanguageID == 5 {
		t.Fatalf("LanguageID did not change after 10,000 ticks of isolation!")
	}
	if culture.LanguageID <= 1000 {
		t.Errorf("Expected new global dialect ID > 1000, got %d", culture.LanguageID)
	}
}

// TestLanguageDriftPidginCreation verifies two entities forming a shared pidgin
// after 50,000 ticks of foreign interaction.
func TestLanguageDriftPidginCreation(t *testing.T) {
	world := ecs.NewWorld()

	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)

	entity := world.NewEntity()
	world.Add(entity, cultureID, memoryID)

	culture := (*components.CultureComponent)(world.Get(entity, cultureID))
	memory := (*components.Memory)(world.Get(entity, memoryID))

	// Initial State
	culture.LanguageID = 10
	culture.DialectTickStamp = 0

	sys := &LanguageDriftSystem{}

	// Provide constant interaction with LanguageID 10 to prevent dialect formation,
	// and also foreign LanguageID 20 to trigger Pidgin creation.

	// We run system 500 times, but the system's tickCounter only hits % 100 == 0 every 100 ticks.
	// So we need to run it 50,000 times!
	for i := 1; i <= 50000; i++ {
		// Periodically update memory to reflect interaction this cycle (e.g. every tick, or every 10 ticks)
		if i%10 == 0 {
			for j := range memory.Events {
				memory.Events[j] = components.MemoryEvent{}
			}
			memory.Events[0] = components.MemoryEvent{
				TickStamp:  uint64(i),
				LanguageID: 10, // Prevent dialect shift
			}
			memory.Events[1] = components.MemoryEvent{
				TickStamp:  uint64(i),
				LanguageID: 20, // Foreign interaction
			}
		}

		sys.Update(&world)

		// Check intermediate state
		if i == 25000 {
			if culture.LanguageID != 10 {
				t.Fatalf("Language changed to Pidgin prematurely at tick %d", i)
			}
			if culture.ForeignLanguageID != 20 {
				t.Fatalf("Foreign language not tracked correctly, got %d", culture.ForeignLanguageID)
			}
			if culture.ForeignInteractionTicks != 25000 {
				t.Fatalf("Foreign interaction ticks tracking failed, got %d", culture.ForeignInteractionTicks)
			}
		}
	}

	// At 50,000 ticks, Pidgin should be created.
	if culture.LanguageID == 10 {
		t.Fatalf("Pidgin language not created after 50,000 ticks of foreign interaction")
	}

	// Mathematically verify Pidgin ID
	// min=10, max=20 -> 50000 + 10 + (20 * 3) = 50070
	expectedPidginID := uint16(50000) + 10 + (20 * 3)
	if culture.LanguageID != expectedPidginID {
		t.Errorf("Expected Pidgin ID %d, got %d", expectedPidginID, culture.LanguageID)
	}

	if culture.ForeignInteractionTicks != 0 {
		t.Errorf("ForeignInteractionTicks should reset after Pidgin creation, got %d", culture.ForeignInteractionTicks)
	}
}
