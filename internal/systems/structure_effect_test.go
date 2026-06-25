package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: StructureEffectSystem tests.
//
// The system increments its internal tick first, then fires only when
// tick % structureEffectTickRate (200) == 0. So the effect lands on the
// 200th Update call and is a no-op before that.

// spawnStructure creates a StructureComponent + Position entity.
func spawnStructure(world *ecs.World, stype uint8, dataA uint32, x, y float32) ecs.Entity {
	structID := ecs.ComponentID[components.StructureComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	e := world.NewEntity(structID, posID)
	st := (*components.StructureComponent)(world.Get(e, structID))
	st.StructureType = uint32(stype)
	st.DataA = dataA
	pos := (*components.Position)(world.Get(e, posID))
	pos.X = x
	pos.Y = y
	return e
}

// runToFire calls Update until the system fires once (tick == 200).
func runToFire(s *StructureEffectSystem, world *ecs.World) {
	for i := 0; i < structureEffectTickRate; i++ {
		s.Update(world)
	}
}

// TestStructureEffectHouseRestAura verifies a House grants +5 Rest to a nearby
// NPC with Needs when the system fires, and that nothing happens beforehand.
func TestStructureEffectHouseRestAura(t *testing.T) {
	world := ecs.NewWorld()

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	// House at (10,10).
	spawnStructure(&world, components.StructureHouse, 0, 10, 10)

	// NPC at (11,10): distSq = 1 <= 9, in range.
	npc := world.NewEntity(npcID, posID, needsID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X = 11
	npcPos.Y = 10
	needs := (*components.Needs)(world.Get(npc, needsID))
	needs.Rest = 0

	s := NewStructureEffectSystem()

	// One Update (tick == 1): no effect.
	s.Update(&world)
	if got := needs.Rest; got != 0 {
		t.Fatalf("after 1 Update, Rest = %v, want 0 (no effect before %d ticks)", got, structureEffectTickRate)
	}

	// Continue until the system fires at tick == 200 (199 more calls).
	for i := 1; i < structureEffectTickRate; i++ {
		s.Update(&world)
	}
	if got := needs.Rest; got != 5 {
		t.Fatalf("after %d Updates, Rest = %v, want 5", structureEffectTickRate, got)
	}
}

// TestStructureEffectHouseOutOfRange verifies a House does NOT affect an NPC
// outside the distSq<=9 threshold.
func TestStructureEffectHouseOutOfRange(t *testing.T) {
	world := ecs.NewWorld()

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	spawnStructure(&world, components.StructureHouse, 0, 10, 10)

	// NPC at (14,10): distSq = 16 > 9, out of range.
	npc := world.NewEntity(npcID, posID, needsID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X = 14
	npcPos.Y = 10
	needs := (*components.Needs)(world.Get(npc, needsID))
	needs.Rest = 0

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	if got := needs.Rest; got != 0 {
		t.Fatalf("out-of-range NPC Rest = %v, want 0", got)
	}
}

// TestStructureEffectTavernRestAura verifies a Tavern grants +3 Rest within
// the wider distSq<=25 threshold.
func TestStructureEffectTavernRestAura(t *testing.T) {
	world := ecs.NewWorld()

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	// Tavern at (10,10).
	spawnStructure(&world, components.StructureTavern, 0, 10, 10)

	// NPC at (14,10): distSq = 16 <= 25, in range (but would be out for a House).
	npc := world.NewEntity(npcID, posID, needsID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X = 14
	npcPos.Y = 10
	needs := (*components.Needs)(world.Get(npc, needsID))
	needs.Rest = 0

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	if got := needs.Rest; got != 3 {
		t.Fatalf("Tavern aura Rest = %v, want 3", got)
	}
}

// TestStructureEffectShrineBeliefSpread verifies a Shrine spreads its DataA
// BeliefID to a nearby NPC with a BeliefComponent, granting weight >= 1.
func TestStructureEffectShrineBeliefSpread(t *testing.T) {
	world := ecs.NewWorld()

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)

	// Shrine at (10,10) spreading BeliefID 42.
	spawnStructure(&world, components.StructureShrine, 42, 10, 10)

	// NPC at (13,11): distSq = 9+1 = 10 <= 25, in range.
	npc := world.NewEntity(npcID, posID, beliefID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X = 13
	npcPos.Y = 11

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	belief := (*components.BeliefComponent)(world.Get(npc, beliefID))
	var weight int32 = -1
	for _, b := range belief.Beliefs {
		if b.BeliefID == 42 {
			weight = b.Weight
		}
	}
	if weight < 1 {
		t.Fatalf("Shrine belief spread: belief 42 weight = %v (beliefs=%v), want >= 1", weight, belief.Beliefs)
	}
}

// TestStructureEffectShrineZeroDataNoSpread verifies a Shrine with DataA == 0
// spreads nothing (the guard st.dataA != 0).
func TestStructureEffectShrineZeroDataNoSpread(t *testing.T) {
	world := ecs.NewWorld()

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)

	spawnStructure(&world, components.StructureShrine, 0, 10, 10)

	npc := world.NewEntity(npcID, posID, beliefID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X = 11
	npcPos.Y = 10

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	belief := (*components.BeliefComponent)(world.Get(npc, beliefID))
	if len(belief.Beliefs) != 0 {
		t.Fatalf("Shrine with DataA==0 should not spread: beliefs = %v, want empty", belief.Beliefs)
	}
}

// TestStructureEffectFarmFeedsNearestVillage verifies a Farm adds Food to the
// nearest in-range village's storage.
func TestStructureEffectFarmFeedsNearestVillage(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)

	// Farm at (10,10).
	spawnStructure(&world, components.StructureFarm, 0, 10, 10)

	// Village at (12,10): distSq = 4 <= 25, in range.
	village := world.NewEntity(villageID, posID, storID)
	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X = 12
	vPos.Y = 10
	stor := (*components.StorageComponent)(world.Get(village, storID))
	stor.Food = 0

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	if got := stor.Food; got != 5 {
		t.Fatalf("Farm feed: village Food = %v, want 5", got)
	}
}

// TestStructureEffectFarmNoVillageInRange verifies a Farm with no village
// within distSq<=25 feeds nothing.
func TestStructureEffectFarmNoVillageInRange(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)

	spawnStructure(&world, components.StructureFarm, 0, 10, 10)

	// Village at (20,10): distSq = 100 > 25, out of range.
	village := world.NewEntity(villageID, posID, storID)
	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X = 20
	vPos.Y = 10
	stor := (*components.StorageComponent)(world.Get(village, storID))
	stor.Food = 0

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	if got := stor.Food; got != 0 {
		t.Fatalf("Farm with no village in range: Food = %v, want 0", got)
	}
}

// TestStructureEffectNoStructuresNoOp verifies the system is a no-op (and does
// not panic) when there are no structures, even when the tick fires.
func TestStructureEffectNoStructuresNoOp(t *testing.T) {
	world := ecs.NewWorld()

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	npc := world.NewEntity(npcID, posID, needsID)
	needs := (*components.Needs)(world.Get(npc, needsID))
	needs.Rest = 0

	s := NewStructureEffectSystem()
	runToFire(s, &world)

	if got := needs.Rest; got != 0 {
		t.Fatalf("no structures: Rest = %v, want 0", got)
	}
}
