package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: PlayerOrderSystem executes direct commands a ruling player issues
// to subordinate NPCs. Obedience is gated by the issuer's Legitimacy versus the
// subordinate's loyalty and hook balance — refusal dents the issuer's authority,
// feeding the existing rebellion/legitimacy systems.

const (
	orderSeekSpeed       = 1.5
	defaultLoyalty       = 30
	defaultLegitimacy    = 25
	disobeyLegitimacyHit = 5
)

// ObedienceCheck reports whether a subordinate obeys given the inputs.
func ObedienceCheck(legitimacy uint32, hookBalance int, loyaltyThreshold uint32) bool {
	return int64(legitimacy)+int64(hookBalance) >= int64(loyaltyThreshold)
}

type issuerData struct {
	legit  uint32
	x, y   float32
	entity ecs.Entity
	hasEnt bool
}

type PlayerOrderSystem struct {
	hooks  *engine.SparseHookGraph
	bridge *InputBridge
}

// NewPlayerOrderSystem builds the order executor.
func NewPlayerOrderSystem(hooks *engine.SparseHookGraph, bridge *InputBridge) *PlayerOrderSystem {
	return &PlayerOrderSystem{hooks: hooks, bridge: bridge}
}

func (s *PlayerOrderSystem) Update(world *ecs.World) {
	orderID := ecs.ComponentID[components.PlayerOrderComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	identID := ecs.ComponentID[components.Identity](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)
	combatID := ecs.ComponentID[components.CombatMarker](world)

	// 1. Pre-cache issuers (rulers) by Identity.ID.
	issuers := make(map[uint64]issuerData)
	iq := world.Query(ecs.All(identID, posID))
	for iq.Next() {
		id := (*components.Identity)(iq.Get(identID)).ID
		pos := (*components.Position)(iq.Get(posID))
		d := issuerData{legit: defaultLegitimacy, x: pos.X, y: pos.Y, entity: iq.Entity(), hasEnt: true}
		if iq.Has(legitID) {
			d.legit = (*components.LegitimacyComponent)(iq.Get(legitID)).Score
		}
		issuers[id] = d
	}

	// 2. Evaluate ordered NPCs.
	type removal struct{ ent ecs.Entity }
	type combatAdd struct {
		ent    ecs.Entity
		target uint64
	}
	type legitDent struct{ issuer ecs.Entity }

	var toRemoveOrder []ecs.Entity
	var toAddCombat []combatAdd
	var dents []legitDent

	q := world.Query(ecs.All(orderID, posID, velID, identID))
	for q.Next() {
		order := (*components.PlayerOrderComponent)(q.Get(orderID))
		pos := (*components.Position)(q.Get(posID))
		vel := (*components.Velocity)(q.Get(velID))
		npcID := (*components.Identity)(q.Get(identID)).ID
		ent := q.Entity()

		// Obedience.
		loyalty := uint32(defaultLoyalty)
		if q.Has(loyaltyID) {
			loyalty = (*components.LoyaltyComponent)(q.Get(loyaltyID)).Value
		}
		issuer := issuers[order.IssuerID]
		balance := 0
		if s.hooks != nil {
			balance = s.hooks.GetHook(npcID, order.IssuerID) + s.hooks.GetHook(order.IssuerID, npcID)
		}
		if !ObedienceCheck(issuer.legit, balance, loyalty) {
			toRemoveOrder = append(toRemoveOrder, ent)
			if issuer.hasEnt {
				dents = append(dents, legitDent{issuer.entity})
			}
			if s.bridge != nil {
				s.bridge.PushEvent(EventOrderRefused, pos.X, pos.Y, int32(npcID))
			}
			continue
		}

		// Execute by order type.
		switch order.OrderType {
		case components.OrderMove, components.OrderBuildAt:
			dx := order.X - pos.X
			dy := order.Y - pos.Y
			if dx*dx+dy*dy < 1.0 {
				if order.OrderType == components.OrderBuildAt && q.Has(jobID) {
					(*components.JobComponent)(q.Get(jobID)).JobID = components.JobBuilder
				}
				vel.X, vel.Y = 0, 0
				toRemoveOrder = append(toRemoveOrder, ent)
			} else {
				steer(vel, dx, dy)
			}
		case components.OrderAssignJob:
			if q.Has(jobID) {
				(*components.JobComponent)(q.Get(jobID)).JobID = order.JobID
			}
			toRemoveOrder = append(toRemoveOrder, ent)
		case components.OrderAttack:
			if !world.Has(ent, combatID) {
				toAddCombat = append(toAddCombat, combatAdd{ent, order.TargetID})
			}
			toRemoveOrder = append(toRemoveOrder, ent)
		case components.OrderFollow:
			if issuer.hasEnt {
				dx := issuer.x - pos.X
				dy := issuer.y - pos.Y
				if dx*dx+dy*dy > 2.0 {
					steer(vel, dx, dy)
				} else {
					vel.X, vel.Y = 0, 0
				}
			}
		}
	}

	// 3. Apply deferred mutations.
	for _, c := range toAddCombat {
		if world.Alive(c.ent) {
			if !world.Has(c.ent, combatID) {
				world.Add(c.ent, combatID)
			}
			(*components.CombatMarker)(world.Get(c.ent, combatID)).TargetID = c.target
		}
	}
	for _, d := range dents {
		if world.Alive(d.issuer) && world.Has(d.issuer, legitID) {
			lg := (*components.LegitimacyComponent)(world.Get(d.issuer, legitID))
			if lg.Score >= disobeyLegitimacyHit {
				lg.Score -= disobeyLegitimacyHit
			} else {
				lg.Score = 0
			}
		}
	}
	for _, e := range toRemoveOrder {
		if world.Alive(e) && world.Has(e, orderID) {
			world.Remove(e, orderID)
		}
	}
}

// steer points a velocity toward (dx,dy) at the order seek speed.
func steer(vel *components.Velocity, dx, dy float32) {
	dist := dx*dx + dy*dy
	if dist <= 0 {
		vel.X, vel.Y = 0, 0
		return
	}
	inv := orderSeekSpeed / fastSqrt(dist)
	vel.X = dx * inv
	vel.Y = dy * inv
}

func fastSqrt(v float32) float32 {
	if v <= 0 {
		return 1
	}
	x := v
	for i := 0; i < 8; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
