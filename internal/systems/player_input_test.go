package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: PlayerInputSystem tests. Deterministic and headless — no
// keyboard input is exercised (ebiten.IsKeyPressed returns false outside a
// running game loop), so WASD movement contributes zero velocity and only the
// MoveSpeed pure function and the bridge-driven attack path are asserted.

// TestMoveSpeed covers the pain-based mobility curve at and around the
// high-pain threshold (>80 halves the base speed).
func TestMoveSpeed(t *testing.T) {
	cases := []struct {
		name string
		pain float32
		want float32
	}{
		{"low pain returns base speed", 10, baseMoveSpeed},
		{"zero pain returns base speed", 0, baseMoveSpeed},
		{"exactly at threshold stays base speed", 80, baseMoveSpeed},
		{"high pain halves speed", 90, painSlowMoveSpeed},
		{"just over threshold halves speed", 81, painSlowMoveSpeed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MoveSpeed(tc.pain); got != tc.want {
				t.Errorf("MoveSpeed(%v) = %v, want %v", tc.pain, got, tc.want)
			}
		})
	}

	// Explicit assertions pinning the tuned constants (playtest: 2.0 was
	// oversensitive; 0.7 walks, 0.35 limps).
	if got := MoveSpeed(90); got != painSlowMoveSpeed {
		t.Errorf("MoveSpeed(90) = %v, want %v", got, painSlowMoveSpeed)
	}
	if got := MoveSpeed(10); got != baseMoveSpeed {
		t.Errorf("MoveSpeed(10) = %v, want %v", got, baseMoveSpeed)
	}
}

// playerInputFixture wires a deterministic world with one possessed player and
// two NPC targets, then returns the system ready to Update.
type playerInputFixture struct {
	world    ecs.World
	bridge   *InputBridge
	system   *PlayerInputSystem
	player   ecs.Entity
	combatID ecs.ID
}

// newPlayerInputFixture builds the world, registers IDs, spawns the possessed
// player (Identity.ID == 1) and two NPC targets at the given coordinates, and
// initializes the system. Player stamina is configurable for the exhausted path.
func newPlayerInputFixture(t *testing.T, stamina float32, targetAID, targetBID uint64, ax, ay, bx, by float32) *playerInputFixture {
	t.Helper()
	world := ecs.NewWorld()

	possessedID := ecs.ComponentID[components.Possessed](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	combatID := ecs.ComponentID[components.CombatMarker](&world)

	// Possessed player: Possessed + Velocity + Position + Vitals + Identity{ID:1}.
	player := world.NewEntity(possessedID, velID, posID, vitalsID, identID)
	playerPos := (*components.Position)(world.Get(player, posID))
	playerPos.X, playerPos.Y = 0, 0
	pv := (*components.VitalsComponent)(world.Get(player, vitalsID))
	pv.Stamina = stamina
	pv.Blood = 100
	(*components.Identity)(world.Get(player, identID)).ID = 1

	// Two NPC targets: Identity + Vitals + Position at known coords.
	spawnTarget := func(id uint64, x, y float32) {
		e := world.NewEntity(identID, vitalsID, posID)
		(*components.Identity)(world.Get(e, identID)).ID = id
		tv := (*components.VitalsComponent)(world.Get(e, vitalsID))
		tv.Stamina = 50
		tv.Blood = 100
		p := (*components.Position)(world.Get(e, posID))
		p.X, p.Y = x, y
	}
	spawnTarget(targetAID, ax, ay)
	spawnTarget(targetBID, bx, by)

	bridge := &InputBridge{}
	system := NewPlayerInputSystem(bridge)
	system.Initialize(&world)

	// Warm up past the attack cooldown. lastAttack starts at 0, so the very
	// first ticks are still inside the cooldown window (tick-0 < attackCooldown)
	// and would swallow an attack. Advance the tick counter with idle Updates
	// (no attack intent) so the first real swing is allowed.
	for i := 0; i < attackCooldown; i++ {
		system.Update(&world)
	}

	return &playerInputFixture{
		world:    world,
		bridge:   bridge,
		system:   system,
		player:   player,
		combatID: combatID,
	}
}

// countEvents returns how many queued bridge events have the given kind.
func countEvents(evs []BridgeEvent, kind uint8) int {
	n := 0
	for _, e := range evs {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

// TestPlayerInputAttackTargetsNearest verifies a valid attack intent attaches a
// CombatMarker pointing at the NPC nearest the attack point, and that the
// attack cooldown then blocks an immediate re-target.
func TestPlayerInputAttackTargetsNearest(t *testing.T) {
	const targetA, targetB = uint64(10), uint64(20)
	// Target A far at (5,5); target B near at (1,0). Attack point lands on B.
	fx := newPlayerInputFixture(t, 50, targetA, targetB, 5, 5, 1, 0)

	fx.bridge.AttackX = 1
	fx.bridge.AttackY = 0
	fx.bridge.AttackValid = true

	fx.system.Update(&fx.world)

	// Player gained a CombatMarker targeting B (the in-range, nearest NPC).
	if !fx.world.Has(fx.player, fx.combatID) {
		t.Fatalf("player should have a CombatMarker after a valid attack")
	}
	cm := (*components.CombatMarker)(fx.world.Get(fx.player, fx.combatID))
	if cm.TargetID != targetB {
		t.Fatalf("CombatMarker.TargetID = %d, want %d (target B)", cm.TargetID, targetB)
	}

	// A swing event was emitted; no exhaustion.
	if got := countEvents(fx.bridge.Events, EventAttackSwing); got != 1 {
		t.Errorf("EventAttackSwing count = %d, want 1", got)
	}
	if got := countEvents(fx.bridge.Events, EventExhausted); got != 0 {
		t.Errorf("EventExhausted count = %d, want 0", got)
	}
	// Intents cleared after consumption.
	if fx.bridge.AttackValid {
		t.Errorf("AttackValid should be cleared after Update")
	}

	// Cooldown: immediately retarget at A — within attackCooldown ticks the
	// swing is blocked, so the marker target stays B and no new swing fires.
	swingsBefore := countEvents(fx.bridge.Events, EventAttackSwing)
	fx.bridge.AttackX = 5
	fx.bridge.AttackY = 5
	fx.bridge.AttackValid = true
	fx.system.Update(&fx.world)

	cm = (*components.CombatMarker)(fx.world.Get(fx.player, fx.combatID))
	if cm.TargetID != targetB {
		t.Errorf("CombatMarker.TargetID = %d during cooldown, want unchanged %d", cm.TargetID, targetB)
	}
	if swingsAfter := countEvents(fx.bridge.Events, EventAttackSwing); swingsAfter != swingsBefore {
		t.Errorf("EventAttackSwing count = %d during cooldown, want unchanged %d", swingsAfter, swingsBefore)
	}
}

// TestPlayerInputAttackExhausted verifies that with stamina <= 5 a valid attack
// intent produces no CombatMarker and pushes an EventExhausted feedback event.
func TestPlayerInputAttackExhausted(t *testing.T) {
	const targetA, targetB = uint64(10), uint64(20)
	// Both targets in range of the attack point; stamina is too low to swing.
	fx := newPlayerInputFixture(t, 5, targetA, targetB, 5, 5, 1, 0)

	fx.bridge.AttackX = 1
	fx.bridge.AttackY = 0
	fx.bridge.AttackValid = true

	fx.system.Update(&fx.world)

	if fx.world.Has(fx.player, fx.combatID) {
		t.Errorf("exhausted player should not gain a CombatMarker")
	}
	if got := countEvents(fx.bridge.Events, EventExhausted); got != 1 {
		t.Errorf("EventExhausted count = %d, want 1", got)
	}
	if got := countEvents(fx.bridge.Events, EventAttackSwing); got != 0 {
		t.Errorf("EventAttackSwing count = %d, want 0", got)
	}
}
