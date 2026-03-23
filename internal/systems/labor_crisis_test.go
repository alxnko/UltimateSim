package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 47: The Plague-Labor Economics Bridge (Butterfly Effect E2E Test)
func TestLaborCrisisSystem_ButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewLaborCrisisSystem(&world, hooks)

	// Force tick processing for testing
	sys.tickCounter = 99

	villageID := ecs.ComponentID[components.Village](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	demoID := ecs.ComponentID[components.DemographicsComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	adminID := ecs.ComponentID[components.AdministrationMarker](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	strikeID := ecs.ComponentID[components.StrikeMarker](&world)

	// 1. Setup Village with massive population drop (Simulating Phase 10.3 Plague)
	cityEnt := world.NewEntity(villageID, popID, marketID, demoID, affilID)
	pop := (*components.PopulationComponent)(world.Get(cityEnt, popID))
	pop.Count = 50 // Dropped

	demo := (*components.DemographicsComponent)(world.Get(cityEnt, demoID))
	demo.PeakPopulation = 100 // 50 is < 80% of 100, triggers LaborCrisis

	market := (*components.MarketComponent)(world.Get(cityEnt, marketID))
	market.WageRate = 1.0

	cityAffil := (*components.Affiliation)(world.Get(cityEnt, affilID))
	cityAffil.CityID = 1

	// 2. Setup Administration/Employer (The State/Ruler)
	// Too poor to pay extorted wages
	rulerEnt := world.NewEntity(adminID, idID, treasuryID)
	rulerId := (*components.Identity)(world.Get(rulerEnt, idID))
	rulerId.ID = 1

	rulerTreasury := (*components.TreasuryComponent)(world.Get(rulerEnt, treasuryID))
	rulerTreasury.Wealth = 10.0 // Very poor

	// 3. Setup Worker (NPC, TraitAmbitious, Employed by Ruler)
	workerEnt := world.NewEntity(npcID, idID, jobID, affilID)
	workerId := (*components.Identity)(world.Get(workerEnt, idID))
	workerId.ID = 2
	workerId.BaseTraits = components.TraitAmbitious

	workerJob := (*components.JobComponent)(world.Get(workerEnt, jobID))
	workerJob.JobID = components.JobFarmer
	workerJob.EmployerID = 1 // Works for Ruler

	workerAffil := (*components.Affiliation)(world.Get(workerEnt, affilID))
	workerAffil.CityID = 1 // In the crisis city

	// --- Execute System ---
	sys.Update(&world)

	// --- Asserts ---

	// Check 1: Crisis Active & Wage Spike
	newDemo := (*components.DemographicsComponent)(world.Get(cityEnt, demoID))
	if !newDemo.LaborCrisisActive {
		t.Errorf("Expected LaborCrisisActive to be true due to 50%% population drop")
	}

	newMarket := (*components.MarketComponent)(world.Get(cityEnt, marketID))
	if newMarket.WageRate <= 1.0 {
		t.Errorf("Expected WageRate to spike during labor crisis, got %f", newMarket.WageRate)
	}

	// Check 2: Worker Quits & Strikes
	newJob := (*components.JobComponent)(world.Get(workerEnt, jobID))
	if newJob.JobID != components.JobNone || newJob.EmployerID != 0 {
		t.Errorf("Expected worker to quit (JobID=0, EmployerID=0), got %d, %d", newJob.JobID, newJob.EmployerID)
	}

	if !world.Has(workerEnt, strikeID) {
		t.Errorf("Expected worker to gain a StrikeMarker")
	} else {
		strike := (*components.StrikeMarker)(world.Get(workerEnt, strikeID))
		if strike.TargetEmployerID != 1 {
			t.Errorf("Expected strike target to be 1, got %d", strike.TargetEmployerID)
		}
	}

	// Check 3: The Butterfly Effect (Blood Feud Hook)
	hook := hooks.GetHook(2, 1)
	if hook != -50 {
		t.Errorf("Expected deep negative hook (-50) against employer, got %d", hook)
	}
}
