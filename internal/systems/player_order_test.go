package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: PlayerOrderSystem E2E tests.
// Deterministic, headless coverage of the obedience gate (ObedienceCheck),
// each order type executed by an obedient subordinate, and the disobedience
// path that dents the issuer's legitimacy and pushes EventOrderRefused.

// spawnIssuer creates a ruling NPC: Identity{ID} + LegitimacyComponent{score} + Position.
// The issuer query in Update is All(Identity, Position); legitimacy is read if present.
func spawnIssuer(world *ecs.World, id uint64, score uint32, x, y float32) ecs.Entity {
	identID := ecs.ComponentID[components.Identity](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	e := world.NewEntity(identID, legitID, posID)
	(*components.Identity)(world.Get(e, identID)).ID = id
	(*components.LegitimacyComponent)(world.Get(e, legitID)).Score = score
	pos := (*components.Position)(world.Get(e, posID))
	pos.X, pos.Y = x, y
	return e
}

// spawnSubordinate creates an ordered NPC matching the Update query
// All(PlayerOrder, Position, Velocity, Identity). Extra components are added by callers.
func spawnSubordinate(world *ecs.World, id uint64, x, y float32, order components.PlayerOrderComponent) ecs.Entity {
	orderID := ecs.ComponentID[components.PlayerOrderComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	identID := ecs.ComponentID[components.Identity](world)

	e := world.NewEntity(orderID, posID, velID, identID)
	(*components.Identity)(world.Get(e, identID)).ID = id
	pos := (*components.Position)(world.Get(e, posID))
	pos.X, pos.Y = x, y
	*(*components.PlayerOrderComponent)(world.Get(e, orderID)) = order
	return e
}

// TestObedienceCheck verifies the legitimacy+hooks vs threshold gate.
func TestObedienceCheck(t *testing.T) {
	cases := []struct {
		name        string
		legitimacy  uint32
		hookBalance int
		threshold   uint32
		want        bool
	}{
		{"legit plus hooks clears threshold", 25, 10, 30, true}, // 35 >= 30
		{"legit alone below threshold", 10, 0, 30, false},       // 10 < 30
		{"negative hooks drop below threshold", 30, -5, 30, false}, // 25 < 30
		{"boundary equal obeys", 30, 0, 30, true},               // 30 >= 30
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ObedienceCheck(tc.legitimacy, tc.hookBalance, tc.threshold); got != tc.want {
				t.Errorf("ObedienceCheck(%d, %d, %d) = %v, want %v",
					tc.legitimacy, tc.hookBalance, tc.threshold, got, tc.want)
			}
		})
	}
}

// TestPlayerOrderMoveFar: an obedient subordinate steers toward a distant target;
// velocity becomes non-zero and the order persists across the tick.
func TestPlayerOrderMoveFar(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	spawnIssuer(&world, 100, 100, 0, 0) // high legitimacy -> obeys (default loyalty 30)
	sub := spawnSubordinate(&world, 1, 0, 0, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderMove,
		X:         50,
		Y:         0,
	})

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if !world.Has(sub, orderID) {
		t.Fatalf("OrderMove to a far target should persist until arrival")
	}
	vel := (*components.Velocity)(world.Get(sub, velID))
	if vel.X == 0 && vel.Y == 0 {
		t.Errorf("velocity should steer toward target, got (%v, %v)", vel.X, vel.Y)
	}
	if vel.X <= 0 {
		t.Errorf("velocity.X should point toward +X target, got %v", vel.X)
	}
	if len(bridge.Events) != 0 {
		t.Errorf("obedient move must not push feedback events, got %d", len(bridge.Events))
	}
}

// TestPlayerOrderMoveArrived: when within 1.0 distance², velocity is zeroed and the order removed.
func TestPlayerOrderMoveArrived(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	spawnIssuer(&world, 100, 100, 0, 0)
	sub := spawnSubordinate(&world, 1, 0, 0, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderMove,
		X:         0.5, // dx²+dy² = 0.25 < 1.0 -> arrived
		Y:         0,
	})
	// Pre-set a non-zero velocity to confirm it gets zeroed.
	v := (*components.Velocity)(world.Get(sub, velID))
	v.X, v.Y = 3, 3

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if world.Has(sub, orderID) {
		t.Errorf("OrderMove within arrival radius should remove the order")
	}
	vel := (*components.Velocity)(world.Get(sub, velID))
	if vel.X != 0 || vel.Y != 0 {
		t.Errorf("arrived velocity should be zeroed, got (%v, %v)", vel.X, vel.Y)
	}
}

// TestPlayerOrderAssignJob: JobComponent.JobID is set and the order removed.
func TestPlayerOrderAssignJob(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)

	spawnIssuer(&world, 100, 100, 0, 0)
	sub := spawnSubordinate(&world, 1, 0, 0, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderAssignJob,
		JobID:     components.JobFarmer,
	})
	world.Add(sub, jobID) // starts at JobNone

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if world.Has(sub, orderID) {
		t.Errorf("OrderAssignJob should be removed after execution")
	}
	job := (*components.JobComponent)(world.Get(sub, jobID))
	if job.JobID != components.JobFarmer {
		t.Errorf("JobComponent.JobID = %d, want JobFarmer(%d)", job.JobID, components.JobFarmer)
	}
}

// TestPlayerOrderAttack: a CombatMarker carrying TargetID is added and the order removed.
func TestPlayerOrderAttack(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)
	combatID := ecs.ComponentID[components.CombatMarker](&world)

	spawnIssuer(&world, 100, 100, 0, 0)
	sub := spawnSubordinate(&world, 1, 0, 0, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderAttack,
		TargetID:  777,
	})

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if world.Has(sub, orderID) {
		t.Errorf("OrderAttack should be removed after execution")
	}
	if !world.Has(sub, combatID) {
		t.Fatalf("OrderAttack should add a CombatMarker")
	}
	cm := (*components.CombatMarker)(world.Get(sub, combatID))
	if cm.TargetID != 777 {
		t.Errorf("CombatMarker.TargetID = %d, want 777", cm.TargetID)
	}
}

// TestPlayerOrderFollow: the order persists and steers the subordinate toward the issuer.
func TestPlayerOrderFollow(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	spawnIssuer(&world, 100, 100, 40, 0) // issuer far along +X
	sub := spawnSubordinate(&world, 1, 0, 0, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderFollow,
	})

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if !world.Has(sub, orderID) {
		t.Errorf("OrderFollow should persist (it is a standing order)")
	}
	vel := (*components.Velocity)(world.Get(sub, velID))
	if vel.X <= 0 {
		t.Errorf("OrderFollow should steer toward issuer at +X, got velocity.X=%v", vel.X)
	}
}

// TestPlayerOrderObedientViaHooks: a low-legitimacy issuer is still obeyed when the
// hook balance between subordinate and issuer makes up the difference.
func TestPlayerOrderObedientViaHooks(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)

	// Issuer legitimacy 10, default loyalty 30 -> needs +20 from hooks.
	spawnIssuer(&world, 100, 10, 0, 0)
	sub := spawnSubordinate(&world, 1, 0, 0, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderMove,
		X:         50,
		Y:         0,
	})
	// balance = GetHook(npc, issuer) + GetHook(issuer, npc) = 15 + 10 = 25; 10+25=35 >= 30.
	hooks.AddHook(1, 100, 15)
	hooks.AddHook(100, 1, 10)

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if !world.Has(sub, orderID) {
		t.Errorf("subordinate with sufficient hooks should obey and keep the move order")
	}
	if len(bridge.Events) != 0 {
		t.Errorf("obedient subordinate must not push EventOrderRefused")
	}
}

// TestPlayerOrderDisobedience: a weak issuer (legitimacy 1), a disloyal subordinate
// (loyalty 200, no hooks) refuses. The order is removed, issuer legitimacy is reduced
// by disobeyLegitimacyHit (floored at 0), and EventOrderRefused is pushed.
func TestPlayerOrderDisobedience(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	bridge := &InputBridge{}

	orderID := ecs.ComponentID[components.PlayerOrderComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)

	issuer := spawnIssuer(&world, 100, 1, 12, 34) // legitimacy 1
	sub := spawnSubordinate(&world, 1, 5, 6, components.PlayerOrderComponent{
		IssuerID:  100,
		OrderType: components.OrderMove,
		X:         500,
		Y:         500,
	})
	world.Add(sub, loyaltyID)
	(*components.LoyaltyComponent)(world.Get(sub, loyaltyID)).Value = 200 // 1 + 0 < 200 -> refuses

	sys := NewPlayerOrderSystem(hooks, bridge)
	sys.Update(&world)

	if world.Has(sub, orderID) {
		t.Errorf("a refused order should be removed")
	}

	// Issuer legitimacy floored: 1 - 5 -> 0.
	lg := (*components.LegitimacyComponent)(world.Get(issuer, legitID))
	if lg.Score != 0 {
		t.Errorf("issuer legitimacy = %d after refusal, want 0 (floored)", lg.Score)
	}

	// EventOrderRefused pushed with the subordinate's position and ID.
	if len(bridge.Events) != 1 {
		t.Fatalf("expected exactly 1 bridge event, got %d", len(bridge.Events))
	}
	ev := bridge.Events[0]
	if ev.Kind != EventOrderRefused {
		t.Errorf("event kind = %d, want EventOrderRefused(%d)", ev.Kind, EventOrderRefused)
	}
	if ev.X != 5 || ev.Y != 6 {
		t.Errorf("event position = (%v, %v), want subordinate (5, 6)", ev.X, ev.Y)
	}
	if ev.Value != 1 {
		t.Errorf("event value = %d, want subordinate Identity.ID 1", ev.Value)
	}
}
