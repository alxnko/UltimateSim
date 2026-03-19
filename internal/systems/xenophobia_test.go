package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestNewXenophobiaSystem(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewXenophobiaSystem(&world, hooks)

	if sys == nil {
		t.Fatal("Expected NewXenophobiaSystem to return a non-nil system")
	}

	if sys.hooks != hooks {
		t.Errorf("Expected hooks to be set correctly")
	}
}

func TestXenophobiaSystem_Update(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewXenophobiaSystem(&world, hooks)

	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	belID := ecs.ComponentID[components.BeliefComponent](&world)
	cultID := ecs.ComponentID[components.CultureComponent](&world)

	// Helper to create an NPC
	createNPC := func(x, y float32, lang uint16, xenophobe bool, id uint64) ecs.Entity {
		e := world.NewEntity()
		world.Add(e, posID, identID, belID, cultID)

		pos := (*components.Position)(world.Get(e, posID))
		pos.X = x
		pos.Y = y

		ident := (*components.Identity)(world.Get(e, identID))
		ident.ID = id

		cult := (*components.CultureComponent)(world.Get(e, cultID))
		cult.LanguageID = lang

		bel := (*components.BeliefComponent)(world.Get(e, belID))
		if xenophobe {
			bel.Beliefs = append(bel.Beliefs, components.Belief{BeliefID: components.BeliefXenophobia, Weight: 50})
		}

		return e
	}

	// 1. Happy Path: Xenophobe vs Foreigner (Close)
	xenophobe := createNPC(0, 0, 1, true, 1)
	foreigner := createNPC(1, 1, 2, false, 2) // distSq = 2 < 10

	sys.tickCounter = 9 // Next update will be tick 10
	sys.Update(&world)

	hook := hooks.GetHook(1, 2)
	if hook != -100 {
		t.Errorf("Expected hook of -100 from xenophobe to foreigner, got %d", hook)
	}

	// 2. Same Language: No hook
	hooks.RemoveAllHooks(1)
	compatriot := createNPC(1, 1, 1, false, 3) // Same language as 1
	sys.tickCounter = 19
	sys.Update(&world)

	hook = hooks.GetHook(1, 3)
	if hook != 0 {
		t.Errorf("Expected no hook for same language, got %d", hook)
	}

	// 3. Non-Xenophobe: No hook
	foreigner2 := createNPC(0, 0, 3, false, 4)
	sys.tickCounter = 29
	sys.Update(&world)
	hook = hooks.GetHook(3, 4) // NPC 3 is not a xenophobe
	if hook != 0 {
		t.Errorf("Expected no hook from non-xenophobe, got %d", hook)
	}

	// 4. Distance Check: Far foreigner, no hook
	farForeigner := createNPC(10, 10, 4, false, 5) // distSq = 200 > 10
	sys.tickCounter = 39
	sys.Update(&world)
	hook = hooks.GetHook(1, 5)
	if hook != 0 {
		t.Errorf("Expected no hook for far foreigner, got %d", hook)
	}

	// 5. Tick Throttling: Update on tick 41 (should do nothing)
	hooks.RemoveAllHooks(1)
	sys.tickCounter = 40
	sys.Update(&world) // tickCounter becomes 41, 41%10 != 0
	hook = hooks.GetHook(1, 2)
	if hook != 0 {
		t.Errorf("Expected no hook due to tick throttling, got %d", hook)
	}

	// 6. Existing Hook: If <= -50, don't add more
	hooks.AddHook(1, 2, -50)
	sys.tickCounter = 49
	sys.Update(&world) // tickCounter becomes 50
	hook = hooks.GetHook(1, 2)
	if hook != -50 {
		t.Errorf("Expected hook to remain -50 (no redundant add), got %d", hook)
	}

	// If hook is > -50, it should add
	hooks.RemoveAllHooks(1)
	hooks.AddHook(1, 2, -49)
	sys.tickCounter = 59
	sys.Update(&world) // tickCounter becomes 60
	hook = hooks.GetHook(1, 2)
	if hook != -149 {
		t.Errorf("Expected hook to be updated from -49 to -149, got %d", hook)
	}
}
