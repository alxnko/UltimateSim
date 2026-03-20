package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.1: Maritime Labor Market (Integration Test)
// Butterfly Effect: Empty Ship -> Hires Desperate Sailor -> Sailor gets Wage -> Ship Treasury Bankrupt -> Sailor Quits -> Ship Stranded
func TestMaritimeLabor_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewMaritimeLaborSystem()

	shipID := ecs.ComponentID[components.ShipComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	// 1. Create Ship
	shipEnt := world.NewEntity(shipID, posID, treasuryID, identID)

	sIdent := (*components.Identity)(world.Get(shipEnt, identID))
	sIdent.ID = 100

	sPos := (*components.Position)(world.Get(shipEnt, posID))
	sPos.X, sPos.Y = 10.0, 10.0

	sShip := (*components.ShipComponent)(world.Get(shipEnt, shipID))
	sShip.CrewRequirements = 1
	sShip.CrewCurrent = 0

	sTreasury := (*components.TreasuryComponent)(world.Get(shipEnt, treasuryID))
	sTreasury.Wealth = 5.0 // Enough for exactly ONE wage tick

	// 2. Create Unemployed Desperate NPC
	npcEnt := world.NewEntity(npcID, posID, jobID, needsID)

	nPos := (*components.Position)(world.Get(npcEnt, posID))
	nPos.X, nPos.Y = 11.0, 11.0 // distSq = 2, close enough to port

	nJob := (*components.JobComponent)(world.Get(npcEnt, jobID))
	nJob.JobID = components.JobNone
	nJob.EmployerID = 0

	nNeeds := (*components.Needs)(world.Get(npcEnt, needsID))
	nNeeds.Wealth = 0.0 // Desperate

	// --- Simulate Tick 50 (Hiring Phase) ---
	sys.tickStamp = 49
	sys.Update(&world) // Tick 50

	if nJob.JobID != components.JobSailor {
		t.Fatalf("Expected NPC to be hired as JobSailor, got %d", nJob.JobID)
	}
	if nJob.EmployerID != sIdent.ID {
		t.Fatalf("Expected NPC to be hired by ship 100, got %d", nJob.EmployerID)
	}
	if sShip.CrewCurrent != 1 {
		t.Fatalf("Expected ship crew current to be 1, got %d", sShip.CrewCurrent)
	}

	// --- Simulate Tick 100 (Wage Phase - Success) ---
	sys.tickStamp = 99
	sys.Update(&world) // Tick 100

	// Also tick hiring, otherwise they might instantly rehire
	// (Wait, hiring is on %50 == 0. Tick 100 is BOTH wage and hiring.)
	// Tick 100 (99+1) is wage and hiring.

	if sTreasury.Wealth != 0.0 {
		t.Fatalf("Expected ship treasury to be depleted (0.0), got %f", sTreasury.Wealth)
	}
	if nNeeds.Wealth != 5.0 {
		t.Fatalf("Expected NPC wealth to increase to 5.0, got %f", nNeeds.Wealth)
	}

	// --- Simulate Tick 200 (Wage Phase - Bankrupt) ---
	sys.tickStamp = 199
	sys.Update(&world) // Tick 200 (199+1 = 200 % 100 == 0)

	nJob = (*components.JobComponent)(world.Get(npcEnt, jobID)) // Reload pointer just in case

	if nJob.JobID != components.JobNone {
		t.Fatalf("Expected NPC to quit (JobNone) due to unpaid wages, got %d", nJob.JobID)
	}
	if sShip.CrewCurrent != 0 {
		t.Fatalf("Expected ship crew to decrement back to 0, got %d", sShip.CrewCurrent)
	}
}
