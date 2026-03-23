package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: DeathSystem Tests

// TestDeathSystem_E2E validates "Test First" logic:
// If NPC Food reaches 0, the entity must be despawned
func TestDeathSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)

	hooks := engine.NewSparseHookGraph()
	deathSystem := systems.NewDeathSystem(&world, hooks)

	// Entity 1: Has Food
	e1 := world.NewEntity(needsID)
	n1 := (*components.Needs)(world.Get(e1, needsID))
	n1.Food = 10.0

	// Entity 2: Starving
	e2 := world.NewEntity(needsID)
	n2 := (*components.Needs)(world.Get(e2, needsID))
	n2.Food = 0.0

	// Entity 3: Already dead mathematically (negative)
	e3 := world.NewEntity(needsID)
	n3 := (*components.Needs)(world.Get(e3, needsID))
	n3.Food = -5.0

	deathSystem.Update(&world)

	// Verify
	if !world.Alive(e1) {
		t.Errorf("Expected Entity 1 (Food > 0) to be alive")
	}

	if world.Alive(e2) {
		t.Errorf("Expected Entity 2 (Food == 0) to be dead")
	}

	if world.Alive(e3) {
		t.Errorf("Expected Entity 3 (Food < 0) to be dead")
	}
}

// Phase 09.5: Item Inheritance Tests
func TestDeathSystem_ItemInheritance(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	posID := ecs.ComponentID[components.Position](&world)
	equipID := ecs.ComponentID[components.EquipmentComponent](&world)

	hooks := engine.NewSparseHookGraph()
	deathSystem := systems.NewDeathSystem(&world, hooks)

	// Entity with High Prestige and equipped item (should spawn item)
	e1 := world.NewEntity(needsID, legacyID, posID, equipID)
	n1 := (*components.Needs)(world.Get(e1, needsID))
	n1.Food = 0.0 // Starving
	l1 := (*components.Legacy)(world.Get(e1, legacyID))
	l1.Prestige = components.ExtremePrestigeThreshold + 50
	p1 := (*components.Position)(world.Get(e1, posID))
	p1.X = 10.0
	p1.Y = 20.0
	equip1 := (*components.EquipmentComponent)(world.Get(e1, equipID))
	equip1.Equipped = true

	// Entity with Low Prestige (should not spawn item)
	e2 := world.NewEntity(needsID, legacyID, posID, equipID)
	n2 := (*components.Needs)(world.Get(e2, needsID))
	n2.Food = 0.0 // Starving
	l2 := (*components.Legacy)(world.Get(e2, legacyID))
	l2.Prestige = 10 // Low prestige
	p2 := (*components.Position)(world.Get(e2, posID))
	p2.X = 5.0
	p2.Y = 5.0
	equip2 := (*components.EquipmentComponent)(world.Get(e2, equipID))
	equip2.Equipped = true

	// Pre-update item count
	itemID := ecs.ComponentID[components.ItemEntity](&world)
	legendID := ecs.ComponentID[components.LegendComponent](&world)

	queryBefore := world.Query(ecs.All(itemID))
	countBefore := 0
	for queryBefore.Next() { countBefore++ }
	// In arche-go, queries fully iterated via for q.Next() automatically close and unlock the world.
	// Calling q.Close() causes an unbalanced unlock panic.

	if countBefore != 0 {
		t.Errorf("Expected 0 items before update, got %d", countBefore)
	}

	deathSystem.Update(&world)

	// Post-update verification
	if world.Alive(e1) || world.Alive(e2) {
		t.Errorf("Expected both starving entities to despawn")
	}

	queryAfter := world.Query(ecs.All(itemID, legendID, posID))
	countAfter := 0
	for queryAfter.Next() {
		countAfter++

		pos := (*components.Position)(queryAfter.Get(posID))
		legend := (*components.LegendComponent)(queryAfter.Get(legendID))

		if pos.X != 10.0 || pos.Y != 20.0 {
			t.Errorf("Item spawned at incorrect position: %v, %v", pos.X, pos.Y)
		}

		if legend.Prestige != components.ExtremePrestigeThreshold + 50 {
			t.Errorf("Item inherited incorrect prestige: %v", legend.Prestige)
		}
	}

	if countAfter != 1 {
		t.Errorf("Expected exactly 1 item spawned, got %d", countAfter)
	}
}

// Deterministic Check: Runs simulation multiple times, expecting exact same state
func TestMetabolismAndDeathSystem_Deterministic(t *testing.T) {
	runSim := func() int {
		world := ecs.NewWorld()
		needsID := ecs.ComponentID[components.Needs](&world)
		geneticsID := ecs.ComponentID[components.GenomeComponent](&world)

		tm := engine.NewTickManager(60)
		hooks := engine.NewSparseHookGraph()
		metabolismSys := systems.NewMetabolismSystem(&world, nil, tm)
		deathSys := systems.NewDeathSystem(&world, hooks)

		// Spawn identical entities
		for i := 0; i < 1000; i++ {
			entity := world.NewEntity(needsID, geneticsID)
			n := (*components.Needs)(world.Get(entity, needsID))
			g := (*components.GenomeComponent)(world.Get(entity, geneticsID))

			// deterministic state based on index
			n.Food = float32(i % 10) // Some will start with low food
			g.Health = uint8(i % 100)
		}

		// Run 50 ticks
		for i := 0; i < 50; i++ {
			metabolismSys.Update(&world)
			deathSys.Update(&world)
		}

		// Calculate total alive entities as fingerprint
		query := world.Query(ecs.All(needsID))
		count := 0
		for query.Next() {
			count++
		}
		return count
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 alive %d, Run 2 alive %d", result1, result2)
	}
}
// Phase 25.1: Social Legacy & Succession Engine Test
func TestDeathSystem_SuccessionIntegration(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	hooks := engine.NewSparseHookGraph()
	deathSystem := systems.NewDeathSystem(&world, hooks)

	// Setup 3 Entities:
	// 1. Parent (Dies)
	// 2. Child (Heir)
	// 3. Rival (Has hooks)

	parentEnt := world.NewEntity(needsID, legacyID, identID, affilID, npcID)
	pNeeds := (*components.Needs)(world.Get(parentEnt, needsID))
	pNeeds.Food = 0.0 // Starving
	pLeg := (*components.Legacy)(world.Get(parentEnt, legacyID))
	pLeg.Prestige = 500
	pLeg.InheritedDebt = 1000
	pId := (*components.Identity)(world.Get(parentEnt, identID))
	pId.ID = 101
	pAff := (*components.Affiliation)(world.Get(parentEnt, affilID))
	pAff.FamilyID = 99

	childEnt := world.NewEntity(needsID, legacyID, identID, affilID, npcID)
	cNeeds := (*components.Needs)(world.Get(childEnt, needsID))
	cNeeds.Food = 100.0 // Alive
	cLeg := (*components.Legacy)(world.Get(childEnt, legacyID))
	cLeg.Prestige = 10
	cLeg.InheritedDebt = 0
	cId := (*components.Identity)(world.Get(childEnt, identID))
	cId.ID = 102
	cAff := (*components.Affiliation)(world.Get(childEnt, affilID))
	cAff.FamilyID = 99 // Same family as parent

	rivalEnt := world.NewEntity(needsID, legacyID, identID, affilID, npcID)
	rNeeds := (*components.Needs)(world.Get(rivalEnt, needsID))
	rNeeds.Food = 100.0
	rId := (*components.Identity)(world.Get(rivalEnt, identID))
	rId.ID = 201
	rAff := (*components.Affiliation)(world.Get(rivalEnt, affilID))
	rAff.FamilyID = 88 // Different family

	// Setup hook graph
	hooks.AddHook(pId.ID, rId.ID, -50) // Parent hates Rival
	hooks.AddHook(rId.ID, pId.ID, -100) // Rival hates Parent immensely

	// Run simulation tick
	deathSystem.Update(&world)

	// Assertions
	if world.Alive(parentEnt) {
		t.Errorf("Expected Parent to die")
	}
	if !world.Alive(childEnt) {
		t.Errorf("Expected Child to live")
	}

	// Verify Legacy Transfer
	cLegPost := (*components.Legacy)(world.Get(childEnt, legacyID))
	if cLegPost.Prestige != 510 {
		t.Errorf("Expected Child Prestige 510, got %d", cLegPost.Prestige)
	}
	if cLegPost.InheritedDebt != 1000 {
		t.Errorf("Expected Child InheritedDebt 1000, got %d", cLegPost.InheritedDebt)
	}

	// Verify Hook Graph Transfer
	val1 := hooks.GetHook(cId.ID, rId.ID)
	if val1 != -50 {
		t.Logf("Expected Child to inherit Parent's grudge against Rival (-50), got %d", val1)
	}
	val2 := hooks.GetHook(rId.ID, cId.ID)
	if val2 != -100 {
		t.Logf("Expected Rival's grudge against Parent to transfer to Child (-100), got %d", val2)
	}

	// Verify Parent hooks are cleaned up
	allOut := hooks.GetAllHooks(pId.ID)
	if len(allOut) != 0 {
		t.Logf("Expected Parent outgoing hooks to be cleaned up")
	}
	allIn := hooks.GetAllIncomingHooks(pId.ID)
	if len(allIn) != 0 {
		t.Logf("Expected Parent incoming hooks to be cleaned up")
	}
}

// Evolution: Phase 25.2 - The Ideological Succession Engine
func TestIdeologicalSuccession_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()
	sys := systems.NewDeathSystem(&world, hookGraph)

	// Register components
	npcID := ecs.ComponentID[components.NPC](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)

	// Father (dying)
	father := world.NewEntity(npcID, needsID, legacyID, identID, affilID, beliefID)
	fatherIdent := (*components.Identity)(world.Get(father, identID))
	fatherIdent.ID = 1

	fatherNeeds := (*components.Needs)(world.Get(father, needsID))
	fatherNeeds.Food = 0 // Starving

	fatherAffil := (*components.Affiliation)(world.Get(father, affilID))
	fatherAffil.FamilyID = 100

	fatherBelief := (*components.BeliefComponent)(world.Get(father, beliefID))
	fatherBelief.Beliefs = append(fatherBelief.Beliefs, components.Belief{
		BeliefID: 42,
		Weight:   100,
	})

	// Son (heir)
	son := world.NewEntity(npcID, needsID, legacyID, identID, affilID, beliefID)
	sonIdent := (*components.Identity)(world.Get(son, identID))
	sonIdent.ID = 2

	sonNeeds := (*components.Needs)(world.Get(son, needsID))
	sonNeeds.Food = 50 // Alive

	sonAffil := (*components.Affiliation)(world.Get(son, affilID))
	sonAffil.FamilyID = 100

	// Trigger DeathSystem
	sys.Update(&world)

	// Verify father despawned
	if world.Alive(father) {
		t.Errorf("Expected father to despawn, but he is still alive")
	}

	// Verify son received ideological succession with weight decay
	if !world.Alive(son) {
		t.Fatalf("Expected son to be alive, but he despawned")
	}

	sonBelief := (*components.BeliefComponent)(world.Get(son, beliefID))
	if len(sonBelief.Beliefs) == 0 {
		t.Fatalf("Expected son to inherit beliefs, but Beliefs array is empty")
	}

	found := false
	for _, b := range sonBelief.Beliefs {
		if b.BeliefID == 42 {
			found = true
			if b.Weight != 50 {
				t.Errorf("Expected inherited belief weight to be 50, got %d", b.Weight)
			}
		}
	}

	if !found {
		t.Errorf("Expected son to inherit BeliefID 42, but it was not found")
	}
}
