package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 70: The Deed Forgery Engine Integration Test
// Demonstrates the "Butterfly Effect": Economic desperation pushes a high-intellect NPC to forge a deed,
// stealing a business. This generates a massive -100 hook against the forger.
// The negative hook naturally triggers the BloodFeudSystem (intent),
// which adds a CombatMarker, bridging into the CombatSystem (biology degradation).
func TestForgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration
	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	combatID := ecs.ComponentID[components.CombatMarker](&world)
	busID := ecs.ComponentID[components.BusinessComponent](&world)

	hooks := engine.NewSparseHookGraph()

	// 2. Initialize Systems
	forgerySys := NewForgerySystem(&world, hooks)
	bloodFeudSys := NewBloodFeudSystem(&world, hooks)
	combatSys := NewCombatSystem(&world)

	// 3. Spawn Entities
	eOwner := world.NewEntity(posID, identID, affID, memID, needsID, vitalsID, genID, despID)
	eForger := world.NewEntity(posID, identID, affID, memID, needsID, vitalsID, genID, despID)
	eBusiness := world.NewEntity(busID)

	// 4. Setup Original Owner (Low Intellect, not desperate)
	oPos := (*components.Position)(world.Get(eOwner, posID))
	oPos.X, oPos.Y = 10.0, 10.0

	oIdent := (*components.Identity)(world.Get(eOwner, identID))
	oIdent.ID = 101

	oGen := (*components.GenomeComponent)(world.Get(eOwner, genID))
	oGen.Intellect = 40 // Uneducated

	oVitals := (*components.VitalsComponent)(world.Get(eOwner, vitalsID))
	oVitals.Blood = 100.0
	oVitals.Stamina = 100.0

	// 5. Setup Business owned by Original Owner
	bus := (*components.BusinessComponent)(world.Get(eBusiness, busID))
	bus.OwnerID = oIdent.ID

	// 6. Setup Forger (High Intellect, Desperate)
	fPos := (*components.Position)(world.Get(eForger, posID))
	fPos.X, fPos.Y = 10.0, 11.0 // Within combat range for later

	fIdent := (*components.Identity)(world.Get(eForger, identID))
	fIdent.ID = 202

	fGen := (*components.GenomeComponent)(world.Get(eForger, genID))
	fGen.Intellect = 85 // Genius

	fDesp := (*components.DesperationComponent)(world.Get(eForger, despID))
	fDesp.Level = 80 // Very desperate

	fVitals := (*components.VitalsComponent)(world.Get(eForger, vitalsID))
	fVitals.Blood = 100.0
	fVitals.Stamina = 100.0

	// ---------------------------------------------------------
	// TICK 1: Forgery System Evaluates and Executes Theft
	// ---------------------------------------------------------
	forgerySys.Update(&world)

	// Assert ownership transferred
	if bus.OwnerID != fIdent.ID {
		t.Fatalf("Tick 1: ForgerySystem failed to transfer ownership. Expected %d, got %d", fIdent.ID, bus.OwnerID)
	}

	// Assert Desperation reset
	if fDesp.Level != 0 {
		t.Fatalf("Tick 1: ForgerySystem failed to reset Desperation level. Expected 0, got %d", fDesp.Level)
	}

	// Assert Memory Event Logged
	fMem := (*components.Memory)(world.Get(eForger, memID))
	lastEvent := fMem.Events[(fMem.Head-1+uint8(len(fMem.Events)))%uint8(len(fMem.Events))]
	if lastEvent.InteractionType != components.InteractionTheft {
		t.Fatalf("Tick 1: ForgerySystem failed to log InteractionTheft in memory.")
	}

	// Assert Hook Generated
	hookValue := hooks.GetHook(oIdent.ID, fIdent.ID)
	if hookValue != -100 {
		t.Fatalf("Tick 1: ForgerySystem failed to generate -100 hook. Got %d", hookValue)
	}

	// ---------------------------------------------------------
	// TICK 2: Blood Feud evaluates intent (Owner wants revenge)
	// ---------------------------------------------------------
	bloodFeudSys.Update(&world)

	// Verify Blood Feud worked (CombatMarker added to Original Owner targeting Forger)
	if !world.Has(eOwner, combatID) {
		t.Fatalf("Tick 2: BloodFeudSystem failed to apply CombatMarker to Owner")
	}

	// ---------------------------------------------------------
	// TICK 3: Combat System applies physical damage (Owner physically attacks Forger)
	// ---------------------------------------------------------
	combatSys.Update(&world)

	// Assert physical violence occurred
	fVitalsNew := (*components.VitalsComponent)(world.Get(eForger, vitalsID))
	if fVitalsNew.Blood != 90.0 {
		t.Fatalf("Tick 3: Expected Forger Blood to decrease to 90.0, got %f", fVitalsNew.Blood)
	}
	if fVitalsNew.Pain != 15.0 {
		t.Fatalf("Tick 3: Expected Forger Pain to spike to 15.0, got %f", fVitalsNew.Pain)
	}
	oVitalsNew := (*components.VitalsComponent)(world.Get(eOwner, vitalsID))
	if oVitalsNew.Stamina != 95.0 {
		t.Fatalf("Tick 3: Expected Owner Stamina to decrease to 95.0 due to attack, got %f", oVitalsNew.Stamina)
	}
}
