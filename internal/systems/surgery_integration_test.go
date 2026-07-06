package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestSurgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	hooks := engine.NewSparseHookGraph()
	// Set workers to 0 for synchronous resolution in tests
	pathQueue := engine.NewPathRequestQueue(100, 0)

	surgerySys := NewSurgerySystem(&world, pathQueue, hooks)

	// Create Patient
	patient := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.AnatomyComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.SanityComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	patientPos := (*components.Position)(world.Get(patient, ecs.ComponentID[components.Position](&world)))
	patientAnatomy := (*components.AnatomyComponent)(world.Get(patient, ecs.ComponentID[components.AnatomyComponent](&world)))
	patientNeeds := (*components.Needs)(world.Get(patient, ecs.ComponentID[components.Needs](&world)))
	patientSanity := (*components.SanityComponent)(world.Get(patient, ecs.ComponentID[components.SanityComponent](&world)))
	patientIdent := (*components.Identity)(world.Get(patient, ecs.ComponentID[components.Identity](&world)))

	patientPos.X = 10
	patientPos.Y = 10
	patientAnatomy.InfectedLimbs = 1
	patientAnatomy.MissingLimbs = 0
	patientAnatomy.InfectionProg = 50.0
	patientNeeds.Wealth = 500.0
	patientSanity.Stress = 0.0
	patientSanity.MaxStress = 100.0
	patientIdent.ID = 1

	// Create Doctor
	doctor := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	doctorPos := (*components.Position)(world.Get(doctor, ecs.ComponentID[components.Position](&world)))
	doctorJob := (*components.JobComponent)(world.Get(doctor, ecs.ComponentID[components.JobComponent](&world)))
	doctorNeeds := (*components.Needs)(world.Get(doctor, ecs.ComponentID[components.Needs](&world)))
	doctorIdent := (*components.Identity)(world.Get(doctor, ecs.ComponentID[components.Identity](&world)))

	doctorPos.X = 10
	doctorPos.Y = 11 // Adjacent
	doctorJob.JobID = components.JobDoctor
	doctorNeeds.Wealth = 0.0
	doctorIdent.ID = 2

	// Run system
	surgerySys.Update(&world)

	// Assert Patient changes
	if patientAnatomy.InfectedLimbs != 0 {
		t.Errorf("Expected InfectedLimbs to be 0, got %d", patientAnatomy.InfectedLimbs)
	}
	if patientAnatomy.MissingLimbs != 1 {
		t.Errorf("Expected MissingLimbs to be 1, got %d", patientAnatomy.MissingLimbs)
	}
	if patientAnatomy.InfectionProg != 0.0 {
		t.Errorf("Expected InfectionProg to be 0.0, got %f", patientAnatomy.InfectionProg)
	}
	if patientSanity.Stress <= 0.0 {
		t.Errorf("Expected massive stress increase, got %f", patientSanity.Stress)
	}
	if patientNeeds.Wealth >= 500.0 {
		t.Errorf("Expected wealth to be transferred from patient, got %f", patientNeeds.Wealth)
	}

	// Assert Doctor changes
	if doctorNeeds.Wealth <= 0.0 {
		t.Errorf("Expected wealth to be transferred to doctor, got %f", doctorNeeds.Wealth)
	}
}

func TestSurgerySystem_DebtHook(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	pathQueue := engine.NewPathRequestQueue(100, 0)
	surgerySys := NewSurgerySystem(&world, pathQueue, hooks)

	patient := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.AnatomyComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.SanityComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	patientPos := (*components.Position)(world.Get(patient, ecs.ComponentID[components.Position](&world)))
	patientAnatomy := (*components.AnatomyComponent)(world.Get(patient, ecs.ComponentID[components.AnatomyComponent](&world)))
	patientNeeds := (*components.Needs)(world.Get(patient, ecs.ComponentID[components.Needs](&world)))
	patientIdent := (*components.Identity)(world.Get(patient, ecs.ComponentID[components.Identity](&world)))
	patientSanity := (*components.SanityComponent)(world.Get(patient, ecs.ComponentID[components.SanityComponent](&world)))

	patientPos.X = 10
	patientPos.Y = 10
	patientAnatomy.InfectedLimbs = 1
	patientNeeds.Wealth = 0.0 // Bankrupt
	patientIdent.ID = 1
	patientSanity.MaxStress = 100.0

	doctor := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	doctorPos := (*components.Position)(world.Get(doctor, ecs.ComponentID[components.Position](&world)))
	doctorJob := (*components.JobComponent)(world.Get(doctor, ecs.ComponentID[components.JobComponent](&world)))
	doctorIdent := (*components.Identity)(world.Get(doctor, ecs.ComponentID[components.Identity](&world)))

	doctorPos.X = 10
	doctorPos.Y = 10 // Adjacent
	doctorJob.JobID = components.JobDoctor
	doctorIdent.ID = 2

	surgerySys.Update(&world)

	// Check if a negative hook was generated from Doctor (2) to Patient (1)
	val := hooks.GetHook(2, 1)
	if val >= 0 {
		t.Errorf("Expected negative hook value, got %d", val)
	}
}
