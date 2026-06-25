package systems

import (
	"errors"
	"sort"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

var errNoTarget = errors.New("possession target is not alive")

// Shell Phase: heir & possession transfer. On player death the UI presents the
// dynasty's living members (surfaced here) and re-possesses the chosen heir;
// DeathSystem already migrates Legacy/debt/artifacts to the new body.

// HeirInfo summarizes a candidate heir for the selection UI.
type HeirInfo struct {
	Entity    ecs.Entity
	ID        uint64
	Name      string
	Age       uint16
	JobID     uint8
	Strength  uint8
	Intellect uint8
}

// FindHeirs lists living family members (excluding the dead player), eldest first.
func FindHeirs(world *ecs.World, familyID uint32, excludeID uint64) []HeirInfo {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)

	var heirs []HeirInfo
	q := world.Query(ecs.All(npcID, identID, affID))
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(affID))
		if aff.FamilyID != familyID {
			continue
		}
		ident := (*components.Identity)(q.Get(identID))
		if ident.ID == excludeID {
			continue
		}
		info := HeirInfo{Entity: q.Entity(), ID: ident.ID, Name: ident.Name, Age: ident.Age}
		if q.Has(jobID) {
			info.JobID = (*components.JobComponent)(q.Get(jobID)).JobID
		}
		if q.Has(genomeID) {
			g := (*components.GenomeComponent)(q.Get(genomeID))
			info.Strength = g.Strength
			info.Intellect = g.Intellect
		}
		heirs = append(heirs, info)
	}
	sort.SliceStable(heirs, func(i, j int) bool {
		if heirs[i].Age != heirs[j].Age {
			return heirs[i].Age > heirs[j].Age
		}
		return heirs[i].ID < heirs[j].ID
	})
	return heirs
}

// PossessEntity transfers the Possessed tag to target, ensuring it can move.
func PossessEntity(world *ecs.World, target ecs.Entity) error {
	possessedID := ecs.ComponentID[components.Possessed](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// Strip Possessed from any current holder.
	var holders []ecs.Entity
	q := world.Query(ecs.All(possessedID))
	for q.Next() {
		holders = append(holders, q.Entity())
	}
	for _, h := range holders {
		if h != target && world.Alive(h) {
			world.Remove(h, possessedID)
		}
	}
	if !world.Alive(target) {
		return errNoTarget
	}
	if !world.Has(target, possessedID) {
		world.Add(target, possessedID)
	}
	if !world.Has(target, velID) {
		world.Add(target, velID)
	}
	return nil
}

// FindStartCandidate picks a deterministic young, family-bound NPC to begin as.
func FindStartCandidate(world *ecs.World) (ecs.Entity, bool) {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	var best ecs.Entity
	var bestID uint64
	found := false
	q := world.Query(ecs.All(npcID, identID, affID))
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(affID))
		ident := (*components.Identity)(q.Get(identID))
		if aff.FamilyID == 0 || ident.Age < 16 || ident.Age > 40 {
			continue
		}
		if !found || ident.ID < bestID {
			best = q.Entity()
			bestID = ident.ID
			found = true
		}
	}
	return best, found
}
