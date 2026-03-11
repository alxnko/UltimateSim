package systems

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

func TestLanguageDriftSystem_DialectFormation(t *testing.T) {
	world := ecs.NewWorld()

	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	// Create entity A
	entityA := world.NewEntity()
	world.Add(entityA, cultureID, memoryID, identID)

	cultA := (*components.CultureComponent)(world.Get(entityA, cultureID))
	identA := (*components.Identity)(world.Get(entityA, identID))
	memA := (*components.Memory)(world.Get(entityA, memoryID))

	cultA.LanguageID = 42
	identA.ID = 100

	// Add an old memory event from someone with the same language
	// We'll create another entity to act as the sender
	entityB := world.NewEntity()
	world.Add(entityB, cultureID, identID)
	cultB := (*components.CultureComponent)(world.Get(entityB, cultureID))
	identB := (*components.Identity)(world.Get(entityB, identID))
	cultB.LanguageID = 42
	identB.ID = 200

	memA.Events[0] = components.MemoryEvent{
		TargetID:        identB.ID,
		TickStamp:       50, // Old tick
		InteractionType: components.InteractionGossip,
		Value:           0,
	}

	system := NewLanguageDriftSystem()
	// Fast forward tick counter
	system.tickCounter = 10099 // Next tick will be 10100 (multiple of 100)

	system.Update(&world) // Runs at 10100. 10100 - 50 = 10050 > 10000

	if cultA.LanguageID == 42 {
		t.Fatalf("Expected Dialect Formation to assign new LanguageID, but got original 42")
	}

	if cultA.LanguageID != 1000 {
		t.Errorf("Expected new LanguageID to be 1000, got %d", cultA.LanguageID)
	}

	// Fast forward another 100 ticks, it should NOT mutate again because of lastMutationMap
	system.tickCounter = 10199
	system.Update(&world)

	if cultA.LanguageID != 1000 {
		t.Errorf("Expected LanguageID to remain 1000, but mutated again to %d", cultA.LanguageID)
	}
}

func TestLanguageDriftSystem_PidginCreation(t *testing.T) {
	world := ecs.NewWorld()

	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	// Create entity A (Language 10)
	entityA := world.NewEntity()
	world.Add(entityA, cultureID, memoryID, identID)
	cultA := (*components.CultureComponent)(world.Get(entityA, cultureID))
	identA := (*components.Identity)(world.Get(entityA, identID))
	memA := (*components.Memory)(world.Get(entityA, memoryID))
	cultA.LanguageID = 10
	identA.ID = 100

	// Create entity B (Language 20)
	entityB := world.NewEntity()
	world.Add(entityB, cultureID, memoryID, identID)
	cultB := (*components.CultureComponent)(world.Get(entityB, cultureID))
	identB := (*components.Identity)(world.Get(entityB, identID))
	// memB := (*components.Memory)(world.Get(entityB, memoryID))
	cultB.LanguageID = 20
	identB.ID = 200

	system := NewLanguageDriftSystem()

	// Simulate high volume interaction
	// We need 50,000 interactions to trigger Pidgin Creation.
	// Since we only check recent events (within 100 ticks) and system runs every 100 ticks,
	// we will manually prime the pidginTracker to 49999 to avoid looping 50000 times in test.

	pairKey := (uint32(10) << 16) | uint32(20)
	system.pidginTracker[pairKey] = 49999
	system.tickCounter = 199 // Next tick is 200

	// Add an event just under the current tick window (200 - 100 = 100)
	memA.Events[0] = components.MemoryEvent{
		TargetID:        identB.ID,
		TickStamp:       150, // Recent interaction
		InteractionType: components.InteractionGossip,
		Value:           0,
	}

	// Update system
	system.Update(&world)

	// PIDGIN should be created and assigned to Entity A.
	if cultA.LanguageID == 10 {
		t.Fatalf("Expected Pidgin Creation to assign new LanguageID, but got original 10")
	}

	if cultA.LanguageID != 1000 {
		t.Errorf("Expected Pidgin LanguageID 1000, got %d", cultA.LanguageID)
	}

	// Ensure Tracker reset
	if system.pidginTracker[pairKey] != 0 {
		t.Errorf("Expected tracker to reset to 0, got %d", system.pidginTracker[pairKey])
	}

	// Verify established pidgins are remembered
	if system.establishedPidgins[pairKey] != 1000 {
		t.Errorf("Expected established pidgin 1000, got %d", system.establishedPidgins[pairKey])
	}

	// Now Entity A has Language 1000, let's create Entity C with Language 10
	// to test if it automatically picks up the Pidgin.
	entityC := world.NewEntity()
	world.Add(entityC, cultureID, memoryID, identID)
	cultC := (*components.CultureComponent)(world.Get(entityC, cultureID))
	identC := (*components.Identity)(world.Get(entityC, identID))
	memC := (*components.Memory)(world.Get(entityC, memoryID))
	cultC.LanguageID = 10
	identC.ID = 300

	memC.Events[0] = components.MemoryEvent{
		TargetID:        identB.ID,
		TickStamp:       250, // Recent interaction (tick 300)
		InteractionType: components.InteractionGossip,
		Value:           0,
	}

	system.tickCounter = 299
	system.Update(&world)

	// Entity C should immediately learn the established Pidgin (1000)
	if cultC.LanguageID != 1000 {
		t.Errorf("Expected Entity C to learn established Pidgin 1000, got %d", cultC.LanguageID)
	}
}

func TestLanguageDriftSystem_MaintainsLanguage(t *testing.T) {
	world := ecs.NewWorld()

	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	entityA := world.NewEntity()
	world.Add(entityA, cultureID, memoryID, identID)

	cultA := (*components.CultureComponent)(world.Get(entityA, cultureID))
	identA := (*components.Identity)(world.Get(entityA, identID))
	memA := (*components.Memory)(world.Get(entityA, memoryID))

	cultA.LanguageID = 99
	identA.ID = 100

	entityB := world.NewEntity()
	world.Add(entityB, cultureID, identID)
	cultB := (*components.CultureComponent)(world.Get(entityB, cultureID))
	identB := (*components.Identity)(world.Get(entityB, identID))
	cultB.LanguageID = 99
	identB.ID = 200

	system := NewLanguageDriftSystem()
	system.tickCounter = 10099

	// Provide recent interaction (within 10,000 ticks)
	memA.Events[0] = components.MemoryEvent{
		TargetID:        identB.ID,
		TickStamp:       10000, // Very recent interaction (tickCounter is 10100, so 100 ticks ago)
		InteractionType: components.InteractionGossip,
		Value:           0,
	}

	// Make sure entityB is added to the world and identifiable for language map lookups
	system.Update(&world)

	if cultA.LanguageID != 99 {
		t.Errorf("Expected entity to maintain LanguageID 99, got %d", cultA.LanguageID)
	}
}
