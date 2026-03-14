package systems

import (
	"testing"
	"time"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 15.4: Physical Locations & Workplaces

func TestWorkplaceSystem(t *testing.T) {
	// 1. Setup ECS World and PathQueue
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(10, 1)
	pathQueue.StartWorkers()
	defer pathQueue.Close()

	sys := NewWorkplaceSystem(pathQueue)

	// 2. Create a Business Entity with a Workplace
	business := world.NewEntity()
	world.Add(business,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.BusinessComponent](&world),
		ecs.ComponentID[components.WorkplaceComponent](&world),
		ecs.ComponentID[components.TreasuryComponent](&world),
	)

	employerID := uint64(100)
	initialWealth := float32(100.0)

	idComp := (*components.Identity)(world.Get(business, ecs.ComponentID[components.Identity](&world)))
	idComp.ID = employerID

	wpComp := (*components.WorkplaceComponent)(world.Get(business, ecs.ComponentID[components.WorkplaceComponent](&world)))
	wpComp.X = 10.0
	wpComp.Y = 10.0

	treasuryComp := (*components.TreasuryComponent)(world.Get(business, ecs.ComponentID[components.TreasuryComponent](&world)))
	treasuryComp.Wealth = initialWealth

	// 3. Create an NPC Employee Entity (Unemployed first)
	npc := world.NewEntity()
	world.Add(npc,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Path](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.JobComponent](&world),
	)

	npcIdComp := (*components.Identity)(world.Get(npc, ecs.ComponentID[components.Identity](&world)))
	npcIdComp.ID = 200

	npcPosComp := (*components.Position)(world.Get(npc, ecs.ComponentID[components.Position](&world)))
	npcPosComp.X = 0.0
	npcPosComp.Y = 0.0 // Start away from workplace

	npcJobComp := (*components.JobComponent)(world.Get(npc, ecs.ComponentID[components.JobComponent](&world)))
	npcJobComp.JobID = components.JobArtisan
	npcJobComp.EmployerID = employerID

	npcGeneticsComp := (*components.GenomeComponent)(world.Get(npc, ecs.ComponentID[components.GenomeComponent](&world)))
	npcGeneticsComp.Strength = 50
	npcGeneticsComp.Intellect = 50 // Boost should be 0.5 + 0.5 = 1.0 per tick at work

	// 4. Test logic without NPC at work
	// Tick < 3600 so no path is sent, distance > 1.0 so no productivity
	sys.Update(&world)

	treasury := (*components.TreasuryComponent)(world.Get(business, ecs.ComponentID[components.TreasuryComponent](&world)))
	if treasury.Wealth != initialWealth {
		t.Fatalf("Expected wealth to be %f, got %f (NPC is not at work)", initialWealth, treasury.Wealth)
	}

	// 5. Test PathRequest dispatch
	sys.tickStamp = 3599 // Next tick will be 3600
	sys.Update(&world)

	// Verify Path is updated
	path := (*components.Path)(world.Get(npc, ecs.ComponentID[components.Path](&world)))
	if !path.HasPath || path.TargetX != 10.0 || path.TargetY != 10.0 {
		t.Fatalf("Expected NPC to have path towards Workplace (10, 10), got HasPath: %v, Target: (%f, %f)", path.HasPath, path.TargetX, path.TargetY)
	}

	// Wait for queue worker to mock process if we needed to test movement, but let's test productivity direct.
	time.Sleep(10 * time.Millisecond) // Let goroutine run mock

	// 6. Test Productivity when NPC arrives at Workplace
	pos := (*components.Position)(world.Get(npc, ecs.ComponentID[components.Position](&world)))
	pos.X = 10.0
	pos.Y = 10.0

	// Move NPC to workplace
	sys.Update(&world)

	// Productivity boost = 50 * 0.01 + 50 * 0.01 = 1.0
	expectedWealth := initialWealth + 1.0
	if treasury.Wealth != expectedWealth {
		t.Fatalf("Expected wealth to increase to %f due to productivity, got %f", expectedWealth, treasury.Wealth)
	}
}
