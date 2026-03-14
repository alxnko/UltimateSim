package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 23.1: The Blood Feud Engine (End-to-End Test)
// Proves the "Butterfly Effect" where one deep grudge causes murder,
// which automatically triggers inherited hatred across clan lines without explicitly iterating relationships.
func TestBloodFeudSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration
	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	hooks := engine.NewSparseHookGraph()

	sys := NewBloodFeudSystem(&world, hooks)

	// 2. Spawn entities
	eKiller := world.NewEntity(posID, identID, affID, memID, needsID)
	eVictim := world.NewEntity(posID, identID, affID, memID, needsID)
	eBystander := world.NewEntity(posID, identID, affID, memID, needsID)
	eKillerCousin := world.NewEntity(posID, identID, affID, memID, needsID)

	// 3. Setup Killer
	kPos := (*components.Position)(world.Get(eKiller, posID))
	kPos.X = 10.0
	kPos.Y = 10.0

	kIdent := (*components.Identity)(world.Get(eKiller, identID))
	kIdent.ID = 101

	kAff := (*components.Affiliation)(world.Get(eKiller, affID))
	kAff.ClanID = 1

	kNeeds := (*components.Needs)(world.Get(eKiller, needsID))
	kNeeds.Food = 100.0

	// 4. Setup Victim
	vPos := (*components.Position)(world.Get(eVictim, posID))
	vPos.X = 10.0
	vPos.Y = 11.0 // Adjacent to Killer

	vIdent := (*components.Identity)(world.Get(eVictim, identID))
	vIdent.ID = 202

	vAff := (*components.Affiliation)(world.Get(eVictim, affID))
	vAff.ClanID = 2

	vNeeds := (*components.Needs)(world.Get(eVictim, needsID))
	vNeeds.Food = 100.0

	// 5. Setup Bystander (Victim's Clan)
	bPos := (*components.Position)(world.Get(eBystander, posID))
	bPos.X = 50.0 // Far away
	bPos.Y = 50.0

	bIdent := (*components.Identity)(world.Get(eBystander, identID))
	bIdent.ID = 303

	bAff := (*components.Affiliation)(world.Get(eBystander, affID))
	bAff.ClanID = 2 // Same clan as Victim

	bNeeds := (*components.Needs)(world.Get(eBystander, needsID))
	bNeeds.Food = 100.0

	// 6. Setup Killer's Cousin (Killer's Clan)
	kcPos := (*components.Position)(world.Get(eKillerCousin, posID))
	kcPos.X = 60.0
	kcPos.Y = 60.0

	kcIdent := (*components.Identity)(world.Get(eKillerCousin, identID))
	kcIdent.ID = 404

	kcAff := (*components.Affiliation)(world.Get(eKillerCousin, affID))
	kcAff.ClanID = 1 // Same clan as Killer

	kcNeeds := (*components.Needs)(world.Get(eKillerCousin, needsID))
	kcNeeds.Food = 100.0

	// 7. Inject Deep Grudge
	hooks.AddHook(kIdent.ID, vIdent.ID, -60) // -60 hook

	if hooks.GetHook(101, 202) != -60 {
		t.Fatalf("Expected starting hook to be -60, got %d", hooks.GetHook(101, 202))
	}

	// 8. Execute System
	sys.Update(&world)

	// 9. Assertions

	// Victim Should Be Starving Natively (DeathSystem Phase)
	if vNeeds.Food != 0 {
		t.Errorf("Victim was not murdered! Expected Needs.Food to be 0, got %f", vNeeds.Food)
	}

	// Killer Memory Logs the Event
	kMem := (*components.Memory)(world.Get(eKiller, memID))
	loggedMurder := false
	for _, ev := range kMem.Events {
		if ev.InteractionType == components.InteractionMurder && ev.TargetID == 202 {
			loggedMurder = true
			break
		}
	}
	if !loggedMurder {
		t.Errorf("Killer did not log InteractionMurder in Memory buffer")
	}

	// Bystander (Clan 2) Should Inherit Deep Grudge Against Killer (ID 101)
	bystanderHatesKiller := hooks.GetHook(303, 101)
	if bystanderHatesKiller > -100 {
		t.Errorf("Feud failed to propagate! Bystander's hook against Killer should be <= -100, got %d", bystanderHatesKiller)
	}

	// Bystander (Clan 2) Should Inherit Secondary Grudge Against Killer's Cousin (ID 404)
	bystanderHatesCousin := hooks.GetHook(303, 404)
	if bystanderHatesCousin > -50 {
		t.Errorf("Feud failed to generalize! Bystander's hook against Killer's Cousin should be <= -50, got %d", bystanderHatesCousin)
	}
}
