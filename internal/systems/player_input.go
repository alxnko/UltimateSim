package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: PlayerInputSystem applies player intent to the possessed entity.
// WASD movement is read directly from the keyboard; mouse intents (attack) come
// through the InputBridge so the simulation stays UI-agnostic.

const (
	attackCooldown    = 30  // ticks between attacks
	attackRange       = 1.5 // world units
	baseMoveSpeed     = 2.0
	painSlowMoveSpeed = 1.0
)

// MoveSpeed returns the WASD speed given the player's current pain level.
// High pain (>80) halves mobility.
func MoveSpeed(pain float32) float32 {
	if pain > 80 {
		return painSlowMoveSpeed
	}
	return baseMoveSpeed
}

type PlayerInputSystem struct {
	bridge     *InputBridge
	filter     ecs.Filter
	tick       uint64
	lastAttack uint64

	possessedID ecs.ID
	velID       ecs.ID
	posID       ecs.ID
	vitalsID    ecs.ID
	identID     ecs.ID
	combatID    ecs.ID
}

// NewPlayerInputSystem builds the system bound to a shared InputBridge.
func NewPlayerInputSystem(bridge *InputBridge) *PlayerInputSystem {
	return &PlayerInputSystem{bridge: bridge}
}

// Initialize caches component IDs and the possessed-entity filter.
func (s *PlayerInputSystem) Initialize(world *ecs.World) {
	s.possessedID = ecs.ComponentID[components.Possessed](world)
	s.velID = ecs.ComponentID[components.Velocity](world)
	s.posID = ecs.ComponentID[components.Position](world)
	s.vitalsID = ecs.ComponentID[components.VitalsComponent](world)
	s.identID = ecs.ComponentID[components.Identity](world)
	s.combatID = ecs.ComponentID[components.CombatMarker](world)
	mask := ecs.All(s.possessedID, s.velID, s.posID)
	s.filter = &mask
}

func (s *PlayerInputSystem) Update(world *ecs.World) {
	s.tick++

	// Locate the possessed entity and gather its state.
	var player ecs.Entity
	var px, py float32
	var pain float32
	var stamina float32
	hasPlayer := false

	q := world.Query(s.filter)
	for q.Next() {
		player = q.Entity()
		pos := (*components.Position)(q.Get(s.posID))
		px, py = pos.X, pos.Y
		vel := (*components.Velocity)(q.Get(s.velID))

		if world.Has(player, s.vitalsID) {
			v := (*components.VitalsComponent)(world.Get(player, s.vitalsID))
			pain = v.Pain
			stamina = v.Stamina
		}

		// WASD movement, unless a modal owns the keyboard.
		vel.X, vel.Y = 0, 0
		if s.bridge == nil || !s.bridge.MovementLocked {
			speed := MoveSpeed(pain)
			if ebiten.IsKeyPressed(ebiten.KeyW) {
				vel.Y = -speed
			}
			if ebiten.IsKeyPressed(ebiten.KeyS) {
				vel.Y = speed
			}
			if ebiten.IsKeyPressed(ebiten.KeyA) {
				vel.X = -speed
			}
			if ebiten.IsKeyPressed(ebiten.KeyD) {
				vel.X = speed
			}
		}
		hasPlayer = true
		break
	}
	q.Close()

	if !hasPlayer || s.bridge == nil {
		if s.bridge != nil {
			s.bridge.ClearIntents()
		}
		return
	}

	// Attack intent.
	if s.bridge.AttackValid {
		s.resolveAttack(world, player, px, py, stamina)
	}
	s.bridge.ClearIntents()
}

// resolveAttack finds the nearest valid target near the attack point and
// attaches a CombatMarker to the player, respecting cooldown and stamina.
func (s *PlayerInputSystem) resolveAttack(world *ecs.World, player ecs.Entity, px, py, stamina float32) {
	if s.tick-s.lastAttack < attackCooldown {
		return
	}
	if stamina <= 5 {
		s.bridge.PushEvent(EventExhausted, px, py, 0)
		return
	}

	ax, ay := s.bridge.AttackX, s.bridge.AttackY

	var bestTarget uint64
	bestDist := float32(attackRange * attackRange)
	found := false

	tMask := ecs.All(s.identID, s.vitalsID, s.posID)
	tq := world.Query(&tMask)
	for tq.Next() {
		if tq.Entity() == player {
			continue
		}
		pos := (*components.Position)(tq.Get(s.posID))
		dx := pos.X - ax
		dy := pos.Y - ay
		d := dx*dx + dy*dy
		if d <= bestDist {
			bestDist = d
			bestTarget = (*components.Identity)(tq.Get(s.identID)).ID
			found = true
		}
	}
	tq.Close()

	if !found {
		return
	}

	if !world.Has(player, s.combatID) {
		world.Add(player, s.combatID)
	}
	cm := (*components.CombatMarker)(world.Get(player, s.combatID))
	cm.TargetID = bestTarget
	s.lastAttack = s.tick
	s.bridge.PushEvent(EventAttackSwing, ax, ay, 0)
}
