package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 45: The Penal Labor Engine (Testing & Validation)
func TestPenalLaborSystem_Integration(t *testing.T) {
	// 1. Setup World
	seed := [32]byte{1, 2, 3, 4, 5}
	engine.InitializeRNG(seed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	// 2. Register components
	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	villID := ecs.ComponentID[components.Village](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	penalID := ecs.ComponentID[components.PenalLaborComponent](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	// 3. Create City (State)
	cityEnt := world.NewEntity()
	world.Add(cityEnt, villID, posID, storID, affID, idID)

	cPos := (*components.Position)(world.Get(cityEnt, posID))
	cPos.X, cPos.Y = 10.0, 10.0

	cAff := (*components.Affiliation)(world.Get(cityEnt, affID))
	cAff.CityID = 1

	cIdent := (*components.Identity)(world.Get(cityEnt, idID))
	cIdent.ID = 100 // Ruler ID

	cStor := (*components.StorageComponent)(world.Get(cityEnt, storID))
	cStor.Stone = 0.0

	// 4. Create Convict (NPC with PenalLaborComponent)
	convict := world.NewEntity()
	world.Add(convict, penalID, posID, idID, jobID, needsID)

	cvPos := (*components.Position)(world.Get(convict, posID))
	cvPos.X, cvPos.Y = 10.0, 10.0 // Same location

	cvIdent := (*components.Identity)(world.Get(convict, idID))
	cvIdent.ID = 200

	cvJob := (*components.JobComponent)(world.Get(convict, jobID))
	cvJob.JobID = components.JobPenalLabor

	cvNeeds := (*components.Needs)(world.Get(convict, needsID))
	cvNeeds.Food = 0.0

	cvPenal := (*components.PenalLaborComponent)(world.Get(convict, penalID))
	cvPenal.StateCityID = 1
	cvPenal.RemainingSentence = 2

	// 5. Initialize System
	sys := NewPenalLaborSystem(&world, hooks)

	// --- TICK 1 ---
	sys.Update(&world)

	if cvPenal.RemainingSentence != 1 {
		t.Errorf("Expected remaining sentence 1, got %d", cvPenal.RemainingSentence)
	}

	if cvNeeds.Food != 0.5 {
		t.Errorf("Expected convict food 0.5 (sustenance), got %f", cvNeeds.Food)
	}

	if cStor.Stone != 1 {
		t.Errorf("Expected city stone to increase by 1, got %d", cStor.Stone)
	}

	// --- TICK 2 ---
	// Wait, we can test sentence completion
	sys.Update(&world)

	if world.Has(convict, penalID) {
		t.Errorf("Expected PenalLaborComponent to be removed on sentence completion")
	}

	if cvJob.JobID != components.JobNone {
		t.Errorf("Expected JobID to revert to JobNone after sentence, got %d", cvJob.JobID)
	}
}

func TestPenalLaborSystem_AbolitionistBacklash(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	villID := ecs.ComponentID[components.Village](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	penalID := ecs.ComponentID[components.PenalLaborComponent](&world)

	// City
	cityEnt := world.NewEntity()
	world.Add(cityEnt, villID, posID, storID, affID, idID)

	cAff := (*components.Affiliation)(world.Get(cityEnt, affID))
	cAff.CityID = 1
	cIdent := (*components.Identity)(world.Get(cityEnt, idID))
	cIdent.ID = 100 // Ruler ID

	// Convict
	convict := world.NewEntity()
	world.Add(convict, penalID, posID, idID)

	cvPenal := (*components.PenalLaborComponent)(world.Get(convict, penalID))
	cvPenal.StateCityID = 1
	cvPenal.RemainingSentence = 20

	// Abolitionist
	abol := world.NewEntity()
	world.Add(abol, posID, idID)

	abIdent := (*components.Identity)(world.Get(abol, idID))
	abIdent.ID = 300
	abIdent.BaseTraits = components.TraitAbolitionist

	sys := NewPenalLaborSystem(&world, hooks)
	sys.tickCounter = 9 // next tick is 10, will trigger hook event

	sys.Update(&world)

	// Check if hook was added (-50 grudge against State Ruler)
	allHooks := hooks.GetAllHooks(300)
	if points, ok := allHooks[100]; !ok || points != -50 {
		t.Errorf("Expected -50 hook against Ruler (100) from Abolitionist (300), got %v", points)
	}
}
