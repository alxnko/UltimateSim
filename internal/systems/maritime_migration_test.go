package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 40.1: The Maritime Migration Engine (Butterfly Effect E2E Test)
// Proves that Desperate NPCs (Famine/Crime) with Wealth will naturally board Ships,
// despawning from land and entering the Naval Logistics system.

func TestMaritimeMigrationSystem_Integration(t *testing.T) {
	world1 := setupMaritimeWorld()
	world2 := setupMaritimeWorld() // For determinism check

	sys1 := NewMaritimeMigrationSystem(&world1)
	sys2 := NewMaritimeMigrationSystem(&world2)

	// Tick until offset is hit
	for i := 0; i < 20; i++ {
		sys1.Update(&world1)
		sys2.Update(&world2)
	}

	// Verify NPC was despawned and appended to passenger hold in World 1
	verifyBoarding(t, &world1, sys1)
	// Verify determinism in World 2
	verifyBoarding(t, &world2, sys2)
}

func verifyBoarding(t *testing.T, world *ecs.World, sys *MaritimeMigrationSystem) {
	shipFilter := ecs.All(sys.shipID, sys.passengerID)
	query := world.Query(&shipFilter)

	if !query.Next() {
		t.Fatalf("Expected ship to still exist")
	}

	passComp := (*components.PassengerComponent)(query.Get(sys.passengerID))

	if len(passComp.Passengers) != 1 {
		t.Errorf("Expected 1 passenger aboard ship, got %d", len(passComp.Passengers))
	} else if passComp.Passengers[0].EntityID != 999 {
		t.Errorf("Expected passenger EntityID to be 999, got %d", passComp.Passengers[0].EntityID)
	}

	query.Close()

	// Verify the physical NPC entity was removed from the world
	npcFilter := ecs.All(sys.identID, sys.desperationID)
	npcQuery := world.Query(&npcFilter)

	count := 0
	for npcQuery.Next() {
		count++
	}

	if count != 0 {
		t.Errorf("Expected NPC to be despawned from land map after boarding, but found %d NPCs still active", count)
	}
}

func setupMaritimeWorld() ecs.World {
	world := ecs.NewWorld()

	// Register IDs
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.DesperationComponent](&world)
	ecs.ComponentID[components.Needs](&world)
	ecs.ComponentID[components.ShipComponent](&world)
	ecs.ComponentID[components.PassengerComponent](&world)
	ecs.ComponentID[components.BusinessComponent](&world)
	ecs.ComponentID[components.Identity](&world)

	// Create Ship
	ship := world.NewEntity()
	world.Add(ship,
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.ShipComponent](&world),
		ecs.ComponentID[components.PassengerComponent](&world),
	)

	shipPos := (*components.Position)(world.Get(ship, ecs.ComponentID[components.Position](&world)))
	shipPos.X = 10.0
	shipPos.Y = 10.0

	passComp := (*components.PassengerComponent)(world.Get(ship, ecs.ComponentID[components.PassengerComponent](&world)))
	passComp.Passengers = make([]components.Passenger, 0)

	// Create Desperate NPC near ship
	npc := world.NewEntity()
	world.Add(npc,
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	npcPos := (*components.Position)(world.Get(npc, ecs.ComponentID[components.Position](&world)))
	npcPos.X = 11.0
	npcPos.Y = 11.0 // distSq = 2.0 (under 10.0 threshold)

	desp := (*components.DesperationComponent)(world.Get(npc, ecs.ComponentID[components.DesperationComponent](&world)))
	desp.Level = 50 // Desperate

	needs := (*components.Needs)(world.Get(npc, ecs.ComponentID[components.Needs](&world)))
	needs.Wealth = 100.0 // Can afford passage

	ident := (*components.Identity)(world.Get(npc, ecs.ComponentID[components.Identity](&world)))
	ident.ID = 999

	return world
}
