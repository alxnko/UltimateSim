package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 64: Physical Combat Engine Integration Test
// Tests the Butterfly Effect: BloodFeud (Intent) -> CombatMarker -> CombatSystem (Blood Drain)
func TestCombatSystem_Integration(t *testing.T) {
	world1 := setupCombatTestWorld()
	world2 := setupCombatTestWorld()

	verifyCombatDeterministic(t, &world1, &world2)
}

func setupCombatTestWorld() ecs.World {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	// 1. Systems
	feudSys := systems.NewBloodFeudSystem(&world, hooks)
	combatSys := systems.NewCombatSystem(&world)

	// 2. Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	genomeID := ecs.ComponentID[components.GenomeComponent](&world)

	// 3. Create Killer (Attacker)
	eKiller := world.NewEntity(posID, identID, affID, memID, needsID, vitalsID, genomeID)

	kPos := (*components.Position)(world.Get(eKiller, posID))
	kIdent := (*components.Identity)(world.Get(eKiller, identID))
	kAff := (*components.Affiliation)(world.Get(eKiller, affID))
	kVitals := (*components.VitalsComponent)(world.Get(eKiller, vitalsID))
	kGenome := (*components.GenomeComponent)(world.Get(eKiller, genomeID))

	kPos.X, kPos.Y = 5.0, 5.0
	kIdent.ID = 100
	kAff.ClanID = 1
	kVitals.Stamina = 100.0
	kVitals.Blood = 100.0
	kGenome.Strength = 50 // Base damage = 5.0

	// 4. Create Victim
	eVictim := world.NewEntity(posID, identID, affID, memID, needsID, vitalsID)

	vPos := (*components.Position)(world.Get(eVictim, posID))
	vIdent := (*components.Identity)(world.Get(eVictim, identID))
	vAff := (*components.Affiliation)(world.Get(eVictim, affID))
	vVitals := (*components.VitalsComponent)(world.Get(eVictim, vitalsID))

	vPos.X, vPos.Y = 5.5, 5.0 // Within melee range (distSq < 2.0)
	vIdent.ID = 200
	vAff.ClanID = 2
	vVitals.Blood = 20.0 // 4 hits to kill (4 * 5.0 = 20.0)

	// 5. Seed Grudge
	hooks.AddHook(100, 200, -100) // Deep negative hook triggers Feud

	// 6. Run simulation steps
	// Tick 1: FeudSystem adds CombatMarker
	feudSys.Update(&world)

	// Tick 2-5: CombatSystem resolves combat (4 ticks * 5 damage = 20 damage -> Blood reaches 0)
	for i := 0; i < 4; i++ {
		combatSys.Update(&world)
	}

	return world
}

func verifyCombatDeterministic(t *testing.T, w1, w2 *ecs.World) {
	identID1 := ecs.ComponentID[components.Identity](w1)
	vitalsID1 := ecs.ComponentID[components.VitalsComponent](w1)

	identID2 := ecs.ComponentID[components.Identity](w2)
	vitalsID2 := ecs.ComponentID[components.VitalsComponent](w2)

	query1 := w1.Query(ecs.All(identID1, vitalsID1))
	query2 := w2.Query(ecs.All(identID2, vitalsID2))

	var v1KillerStamina, v1VictimBlood float32
	var v2KillerStamina, v2VictimBlood float32

	for query1.Next() {
		ident := (*components.Identity)(query1.Get(identID1))
		vitals := (*components.VitalsComponent)(query1.Get(vitalsID1))
		if ident.ID == 100 {
			v1KillerStamina = vitals.Stamina
		} else if ident.ID == 200 {
			v1VictimBlood = vitals.Blood
		}
	}

	for query2.Next() {
		ident := (*components.Identity)(query2.Get(identID2))
		vitals := (*components.VitalsComponent)(query2.Get(vitalsID2))
		if ident.ID == 100 {
			v2KillerStamina = vitals.Stamina
		} else if ident.ID == 200 {
			v2VictimBlood = vitals.Blood
		}
	}

	// Determinism Check
	if v1KillerStamina != v2KillerStamina || v1VictimBlood != v2VictimBlood {
		t.Fatalf("Determinism check failed: W1(Stamina=%f, Blood=%f) != W2(Stamina=%f, Blood=%f)", v1KillerStamina, v1VictimBlood, v2KillerStamina, v2VictimBlood)
	}

	// Logic Check
	// Killer attacked 4 times, costing 10 stamina each. Initial was 100.
	if v1KillerStamina != 60.0 {
		t.Fatalf("Expected Killer to have 60 stamina, got %f", v1KillerStamina)
	}

	// Victim was hit 4 times for 5 damage (20 total), initial was 20. Blood should be 0.
	if v1VictimBlood != 0.0 {
		t.Fatalf("Expected Victim to have 0 blood, got %f", v1VictimBlood)
	}
}
