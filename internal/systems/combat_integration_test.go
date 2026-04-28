package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 64: The Physical Combat Engine (End-to-End Test)
// Demonstrates the "Butterfly Effect": Deep negative hook triggers BloodFeudSystem (intent),
// which adds a CombatMarker, which triggers CombatSystem (biology degradation),
// which finally triggers DeathSystem (termination) via exsanguination.
func TestCombatSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration
	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	combatID := ecs.ComponentID[components.CombatMarker](&world)

	hooks := engine.NewSparseHookGraph()

	// 2. Initialize Systems
	bloodFeudSys := NewBloodFeudSystem(&world, hooks)
	combatSys := NewCombatSystem(&world)
	deathSys := NewDeathSystem(&world, hooks)

	// 3. Spawn Entities
	eKiller := world.NewEntity(posID, identID, affID, memID, needsID, vitalsID)
	eVictim := world.NewEntity(posID, identID, affID, memID, needsID, vitalsID)

	// 4. Setup Killer
	kPos := (*components.Position)(world.Get(eKiller, posID))
	kPos.X, kPos.Y = 10.0, 10.0

	kIdent := (*components.Identity)(world.Get(eKiller, identID))
	kIdent.ID = 101

	kAff := (*components.Affiliation)(world.Get(eKiller, affID))
	kAff.ClanID = 1

	kNeeds := (*components.Needs)(world.Get(eKiller, needsID))
	kNeeds.Food = 100.0

	kVitals := (*components.VitalsComponent)(world.Get(eKiller, vitalsID))
	kVitals.Blood = 100.0
	kVitals.Stamina = 100.0
	kVitals.Pain = 0.0

	// 5. Setup Victim
	vPos := (*components.Position)(world.Get(eVictim, posID))
	vPos.X, vPos.Y = 10.0, 11.0 // Within combat/feud range

	vIdent := (*components.Identity)(world.Get(eVictim, identID))
	vIdent.ID = 202

	vAff := (*components.Affiliation)(world.Get(eVictim, affID))
	vAff.ClanID = 2

	vNeeds := (*components.Needs)(world.Get(eVictim, needsID))
	vNeeds.Food = 100.0 // Ensure they don't die of starvation

	vVitals := (*components.VitalsComponent)(world.Get(eVictim, vitalsID))
	vVitals.Blood = 25.0 // Set to 25 so it takes 3 ticks of combat (-10/tick) to kill
	vVitals.Stamina = 100.0
	vVitals.Pain = 0.0

	// 6. Inject Deep Grudge to trigger Blood Feud
	hooks.AddHook(kIdent.ID, vIdent.ID, -60) // Threshold is -50

	// ---------------------------------------------------------
	// TICK 1: Blood Feud evaluates intent
	// ---------------------------------------------------------
	bloodFeudSys.Update(&world)

	// Verify Blood Feud worked (CombatMarker added)
	if !world.Has(eKiller, combatID) {
		t.Fatalf("Tick 1: BloodFeudSystem failed to apply CombatMarker to Killer")
	}

	// ---------------------------------------------------------
	// TICK 2: Combat System applies physical damage
	// ---------------------------------------------------------
	combatSys.Update(&world)

	// Assert damage was done but victim is still alive
	vVitalsNew := (*components.VitalsComponent)(world.Get(eVictim, vitalsID))
	if vVitalsNew.Blood != 15.0 {
		t.Fatalf("Tick 2: Expected Victim Blood to be 15.0, got %f", vVitalsNew.Blood)
	}
	if vVitalsNew.Pain != 15.0 {
		t.Fatalf("Tick 2: Expected Victim Pain to be 15.0, got %f", vVitalsNew.Pain)
	}
	kVitalsNew := (*components.VitalsComponent)(world.Get(eKiller, vitalsID))
	if kVitalsNew.Stamina != 95.0 {
		t.Fatalf("Tick 2: Expected Killer Stamina to be 95.0, got %f", kVitalsNew.Stamina)
	}
	if !world.Alive(eVictim) {
		t.Fatalf("Tick 2: Victim should still be alive")
	}

	// ---------------------------------------------------------
	// TICK 3: Combat System applies more damage
	// ---------------------------------------------------------
	combatSys.Update(&world)

	// Assert more damage
	vVitalsNew = (*components.VitalsComponent)(world.Get(eVictim, vitalsID))
	if vVitalsNew.Blood != 5.0 {
		t.Fatalf("Tick 3: Expected Victim Blood to be 5.0, got %f", vVitalsNew.Blood)
	}

	// ---------------------------------------------------------
	// TICK 4: Combat System delivers fatal blow
	// ---------------------------------------------------------
	combatSys.Update(&world)

	// Assert fatal blow landed
	vVitalsNew = (*components.VitalsComponent)(world.Get(eVictim, vitalsID))
	if vVitalsNew.Blood > 0.0 {
		t.Fatalf("Tick 4: Expected Victim Blood to be <= 0.0, got %f", vVitalsNew.Blood)
	}

	// Assert CombatMarker was removed due to target death
	if world.Has(eKiller, combatID) {
		t.Fatalf("Tick 4: CombatMarker was not removed from Killer after Victim reached 0 blood")
	}

	// ---------------------------------------------------------
	// TICK 5: Death System reaps the corpse
	// ---------------------------------------------------------
	deathSys.Update(&world)

	// Assert the entity was actually despawned from the ECS due to exsanguination
	if world.Alive(eVictim) {
		t.Fatalf("Tick 5: Victim should be dead and removed from world by DeathSystem")
	}
}
