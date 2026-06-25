package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: AmbitionSystem turns the sandbox into a directed-but-open game.
// It offers goals derived from world state, tracks their progress, and rewards
// completion with Legacy prestige. Pure helpers are UI-callable.

const (
	ambitionTickRate = 400
	ambitionReward   = 25
	maxOffers        = 3
	maxActive        = 3
)

// hasAmbitionType reports whether a list already holds an ambition of a type.
func hasAmbitionType(list []components.Ambition, t uint8) bool {
	for i := range list {
		if list[i].Type == t {
			return true
		}
	}
	return false
}

// GenerateOffers builds up to maxOffers fresh ambitions from world state,
// skipping types already active or offered. Deterministic.
func GenerateOffers(world *ecs.World, hooks *engine.SparseHookGraph, player ecs.Entity, tick uint64) []components.Ambition {
	affID := ecs.ComponentID[components.Affiliation](world)
	needsID := ecs.ComponentID[components.Needs](world)
	ambID := ecs.ComponentID[components.AmbitionsComponent](world)
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	identID := ecs.ComponentID[components.Identity](world)

	var active, offers []components.Ambition
	var builtCount uint32
	if world.Has(player, ambID) {
		a := (*components.AmbitionsComponent)(world.Get(player, ambID))
		active = a.Ambitions
		offers = a.Offers
		builtCount = a.BuiltCount
	}
	taken := func(t uint8) bool { return hasAmbitionType(active, t) || hasAmbitionType(offers, t) }

	var out []components.Ambition
	add := func(a components.Ambition) {
		if len(out) < maxOffers {
			out = append(out, a)
		}
	}

	var cityID uint32
	var familyID uint32
	if world.Has(player, affID) {
		aff := (*components.Affiliation)(world.Get(player, affID))
		cityID = aff.CityID
		familyID = aff.FamilyID
	}

	// 1. Become ruler of own city.
	if cityID != 0 && !world.Has(player, adminID) && !taken(components.AmbitionRuler) {
		add(components.Ambition{Type: components.AmbitionRuler, TargetID: uint64(cityID)})
	}
	// 2. Amass wealth.
	if !taken(components.AmbitionWealth) {
		goal := uint32(100)
		if world.Has(player, needsID) {
			w := uint32((*components.Needs)(world.Get(player, needsID)).Wealth) * 2
			if w > goal {
				goal = w
			}
		}
		add(components.Ambition{Type: components.AmbitionWealth, Goal: goal})
	}
	// 3. Build structures.
	if !taken(components.AmbitionBuilder) {
		add(components.Ambition{Type: components.AmbitionBuilder, Goal: builtCount + 3})
	}
	// 4. Feud: first NPC who dislikes the player.
	if !taken(components.AmbitionFeud) && hooks != nil && world.Has(player, identID) {
		pid := (*components.Identity)(world.Get(player, identID)).ID
		var rival uint64
		for src, val := range hooks.GetAllIncomingHooks(pid) {
			if val <= -2 && (rival == 0 || src < rival) {
				rival = src
			}
		}
		if rival != 0 {
			add(components.Ambition{Type: components.AmbitionFeud, TargetID: rival})
		}
	}
	// 5. Raise an heir.
	if familyID != 0 && !taken(components.AmbitionHeir) {
		add(components.Ambition{Type: components.AmbitionHeir})
	}
	return out
}

// AcceptOffer moves Offers[idx] into the active list.
func AcceptOffer(amb *components.AmbitionsComponent, idx int) bool {
	if idx < 0 || idx >= len(amb.Offers) || len(amb.Ambitions) >= maxActive {
		return false
	}
	a := amb.Offers[idx]
	a.Accepted = true
	amb.Ambitions = append(amb.Ambitions, a)
	amb.Offers = append(amb.Offers[:idx], amb.Offers[idx+1:]...)
	return true
}

// DismissOffer drops Offers[idx].
func DismissOffer(amb *components.AmbitionsComponent, idx int) bool {
	if idx < 0 || idx >= len(amb.Offers) {
		return false
	}
	amb.Offers = append(amb.Offers[:idx], amb.Offers[idx+1:]...)
	return true
}

// RecordPlayerBuild registers a completed player-funded structure.
func RecordPlayerBuild(amb *components.AmbitionsComponent) {
	amb.BuiltCount++
	for i := range amb.Ambitions {
		if amb.Ambitions[i].Type == components.AmbitionBuilder && !amb.Ambitions[i].Done {
			amb.Ambitions[i].Progress = amb.BuiltCount
		}
	}
}

type AmbitionSystem struct {
	hooks *engine.SparseHookGraph
	tick  uint64
}

func NewAmbitionSystem(hooks *engine.SparseHookGraph) *AmbitionSystem {
	return &AmbitionSystem{hooks: hooks}
}

// IsExpensive throttles the system during fast-forward.
func (s *AmbitionSystem) IsExpensive() bool { return true }

func (s *AmbitionSystem) Update(world *ecs.World) {
	s.tick++
	if s.tick%ambitionTickRate != 0 {
		return
	}

	possessedID := ecs.ComponentID[components.Possessed](world)
	ambID := ecs.ComponentID[components.AmbitionsComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	needsID := ecs.ComponentID[components.Needs](world)
	legacyID := ecs.ComponentID[components.Legacy](world)
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	identID := ecs.ComponentID[components.Identity](world)

	// Iterate to completion (arche auto-closes on exhaustion); calling Close on
	// an exhausted query double-unlocks. Capture the first possessed+ambition.
	q := world.Query(ecs.All(possessedID, ambID))
	var player ecs.Entity
	found := false
	for q.Next() {
		if !found {
			player = q.Entity()
			found = true
		}
	}
	if !found {
		return
	}

	amb := (*components.AmbitionsComponent)(world.Get(player, ambID))

	// Refresh offers if room.
	if len(amb.Offers) == 0 && len(amb.Ambitions) < maxActive {
		amb.Offers = GenerateOffers(world, s.hooks, player, s.tick)
	}

	// Family snapshot for heir ambitions.
	var familyID uint32
	if world.Has(player, affID) {
		familyID = (*components.Affiliation)(world.Get(player, affID)).FamilyID
	}
	familyCount := func() uint32 {
		if familyID == 0 {
			return 0
		}
		npcID := ecs.ComponentID[components.NPC](world)
		var n uint32
		fq := world.Query(ecs.All(npcID, affID))
		for fq.Next() {
			if (*components.Affiliation)(fq.Get(affID)).FamilyID == familyID {
				n++
			}
		}
		return n
	}

	// Progress + completion.
	for i := range amb.Ambitions {
		a := &amb.Ambitions[i]
		if a.Done {
			continue
		}
		switch a.Type {
		case components.AmbitionWealth:
			if world.Has(player, needsID) {
				a.Progress = uint32((*components.Needs)(world.Get(player, needsID)).Wealth)
			}
			if a.Progress >= a.Goal {
				s.complete(world, player, legacyID, a)
			}
		case components.AmbitionRuler:
			if world.Has(player, adminID) {
				if aff := (*components.Affiliation)(world.Get(player, affID)); uint64(aff.CityID) == a.TargetID {
					a.Progress = 1
					s.complete(world, player, legacyID, a)
				}
			}
		case components.AmbitionBuilder:
			a.Progress = amb.BuiltCount
			if a.Progress >= a.Goal {
				s.complete(world, player, legacyID, a)
			}
		case components.AmbitionFeud:
			if !rivalAlive(world, identID, a.TargetID) {
				a.Progress = 1
				s.complete(world, player, legacyID, a)
			}
		case components.AmbitionHeir:
			if amb.FamilyBase == 0 {
				amb.FamilyBase = familyCount()
			} else if familyCount() > amb.FamilyBase {
				a.Progress = 1
				s.complete(world, player, legacyID, a)
			}
		}
	}
}

func (s *AmbitionSystem) complete(world *ecs.World, player ecs.Entity, legacyID ecs.ID, a *components.Ambition) {
	a.Done = true
	if world.Has(player, legacyID) {
		(*components.Legacy)(world.Get(player, legacyID)).Prestige += ambitionReward
	}
}

func rivalAlive(world *ecs.World, identID ecs.ID, rivalID uint64) bool {
	q := world.Query(ecs.All(identID))
	for q.Next() {
		if (*components.Identity)(q.Get(identID)).ID == rivalID {
			q.Close()
			return true
		}
	}
	return false
}
