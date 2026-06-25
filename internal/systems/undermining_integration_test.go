package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestUnderminingSystem_Integration proves the E2E Butterfly Effect:
// 1. A global WarTrackerComponent exists (siege active).
// 2. A JobMiner NPC spawns near a StructureComponent.
// 3. The Miner creates a TunnelComponent.
// 4. The Tunnel progresses, dropping the Structure's Integrity.
// 5. The Structure is ultimately destroyed.
func TestUnderminingSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register IDs
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	structID := ecs.ComponentID[components.StructureComponent](&world)
	tunnelID := ecs.ComponentID[components.TunnelComponent](&world)

	// Setup Global War Tracker
	warEntity := world.NewEntity()
	world.Add(warEntity, warID)
	war := (*components.WarTrackerComponent)(world.Get(warEntity, warID))
	war.Active = true

	// Setup Miner NPC
	miner := world.NewEntity()
	world.Add(miner, jobID, posID)
	job := (*components.JobComponent)(world.Get(miner, jobID))
	job.JobID = components.JobMiner
	mPos := (*components.Position)(world.Get(miner, posID))
	mPos.X = 10
	mPos.Y = 10

	// Setup Target Structure (e.g., Castle Wall)
	wall := world.NewEntity()
	world.Add(wall, structID, idID, posID)

	ident := (*components.Identity)(world.Get(wall, idID))
	ident.ID = 999

	wPos := (*components.Position)(world.Get(wall, posID))
	wPos.X = 12
	wPos.Y = 12

	strComp := (*components.StructureComponent)(world.Get(wall, structID))
	strComp.Integrity = 15.0

	// Init System
	system := NewUnderminingSystem(&world)

	// TICK 1-9: Nothing happens (evaluate every 10 ticks)
	for i := 1; i < 10; i++ {
		system.Update(&world)
	}

	// TICK 10: Evaluate. Miner detects Wall, spawns Tunnel deferred.
	system.Update(&world)

	// Assert Tunnel was created
	tunnelQuery := world.Query(ecs.All(tunnelID))
	tunnelCount := 0
	for tunnelQuery.Next() {
		tunnelCount++
		tun := (*components.TunnelComponent)(tunnelQuery.Get(tunnelID))
		if tun.TargetID != 999 {
			t.Errorf("Expected Tunnel to target Wall ID 999, got %d", tun.TargetID)
		}
	}
	if tunnelCount != 1 {
		t.Fatalf("Expected 1 Tunnel to be spawned, got %d", tunnelCount)
	}

	// TICK 11-19: Wait
	for i := 11; i < 20; i++ {
		system.Update(&world)
	}

	// TICK 20: Tunnel progresses, damages Wall (Integrity drops 15 -> 10)
	system.Update(&world)
	if !world.Alive(wall) {
		t.Fatalf("Wall should be alive at Tick 20")
	}
	strComp = (*components.StructureComponent)(world.Get(wall, structID))
	if strComp.Integrity != 10.0 {
		t.Errorf("Expected Integrity 10.0, got %f", strComp.Integrity)
	}

	// TICK 21-29: Wait
	for i := 21; i < 30; i++ {
		system.Update(&world)
	}

	// TICK 30: Tunnel progresses, damages Wall (Integrity drops 10 -> 5)
	system.Update(&world)

	// TICK 31-39: Wait
	for i := 31; i < 40; i++ {
		system.Update(&world)
	}

	// TICK 40: Tunnel progresses, damages Wall (Integrity drops 5 -> 0, Destroyed)
	system.Update(&world)

	if world.Alive(wall) {
		t.Errorf("Expected Wall to be destroyed at Integrity <= 0, but it is still alive")
	}
}
