package systems

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 15.2: Employment & Wages Test
func TestJobMarketSystem(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize components
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	businessID := ecs.ComponentID[components.BusinessComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)

	// Create a Business
	business := world.NewEntity()
	world.Add(business, businessID, idID, treasuryID)

	id := (*components.Identity)(world.Get(business, idID))
	id.ID = 100 // Business ID
	id.Name = "Test Farm"

	treasury := (*components.TreasuryComponent)(world.Get(business, treasuryID))
	treasury.Wealth = 10.0 // Starting wealth

	// Create 3 Unemployed NPCs
	var npcs []ecs.Entity
	for i := 0; i < 3; i++ {
		npc := world.NewEntity()
		world.Add(npc, npcID, idID, jobID, needsID)

		npcIDComp := (*components.Identity)(world.Get(npc, idID))
		npcIDComp.ID = uint64(i + 1)

		job := (*components.JobComponent)(world.Get(npc, jobID))
		job.JobID = components.JobNone
		job.EmployerID = 0 // Unemployed

		needs := (*components.Needs)(world.Get(npc, needsID))
		needs.Wealth = 10.0 // Low wealth, needs job

		npcs = append(npcs, npc)
	}

	// Initialize the system
	sys := NewJobMarketSystem(&world)

	// --- Test 1: Hiring ---
	// Fast forward to tick 50 (hiring cycle)
	sys.tickStamp = 49
	sys.Update()

	// Check that NPCs are hired
	for _, npc := range npcs {
		job := (*components.JobComponent)(world.Get(npc, jobID))
		if job.EmployerID != 100 {
			t.Errorf("NPC should have been hired by business 100, got %d", job.EmployerID)
		}
		if job.JobID != components.JobArtisan {
			t.Errorf("NPC should be an Artisan, got %d", job.JobID)
		}
	}

	// --- Test 2: Wage Distribution ---
	// Fast forward to tick 60 (wage cycle)
	sys.tickStamp = 59
	sys.Update()

	// Each of the 3 NPCs should have received 1.0 wealth, reducing Business treasury by 3.0
	for _, npc := range npcs {
		needs := (*components.Needs)(world.Get(npc, needsID))
		if needs.Wealth != 11.0 { // 10.0 initial + 1.0 wage
			t.Errorf("NPC should have 11.0 wealth after wage payment, got %f", needs.Wealth)
		}
	}

	if treasury.Wealth != 7.0 { // 10.0 initial - 3.0 total wages paid
		t.Errorf("Business treasury should be 7.0, got %f", treasury.Wealth)
	}

	// --- Test 3: Failure to Pay ---
	// Deplete business treasury to simulate inability to pay
	treasury.Wealth = 0.5 // Cannot afford to pay even one employee (wage = 1.0)

	// Fast forward to tick 70 (next wage cycle)
	sys.tickStamp = 69
	sys.Update()

	// Employees should quit since they couldn't be paid
	for _, npc := range npcs {
		job := (*components.JobComponent)(world.Get(npc, jobID))
		if job.EmployerID != 0 || job.JobID != components.JobNone {
			t.Errorf("NPC should have quit due to unpaid wages, but still employed by %d as %d", job.EmployerID, job.JobID)
		}
	}
}
